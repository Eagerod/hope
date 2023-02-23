GO := go

MAIN_FILE := main.go

BUILD_DIR := build
EXECUTABLE := hope
BIN_NAME := $(BUILD_DIR)/$(EXECUTABLE)
INSTALLED_NAME := /usr/local/bin/$(EXECUTABLE)

CMD_PACKAGE_DIR := ./cmd/hope $(dir $(wildcard ./cmd/hope/*/))
PKG_PACKAGE_DIR := ./pkg/*
PACKAGE_PATHS := $(CMD_PACKAGE_DIR) $(PKG_PACKAGE_DIR)

COVERAGE_FILE=./coverage.out

ALL_GO_DIRS = $(shell find . -iname "*.go" -exec dirname {} \; | sort | uniq)
SRC := $(shell find . -iname "*.go" -and -not -name "*_test.go")
SRC_WITH_TESTS := $(shell find . -iname "*.go")

# Publish targets are treated as phony to force rebuilds.
PUBLISH_DIR=publish
PUBLISH := \
	$(PUBLISH_DIR)/linux-amd64 \
	$(PUBLISH_DIR)/darwin-amd64 \
	$(PUBLISH_DIR)/darwin-arm64

.PHONY: $(PUBLISH)

DOCKER_IMAGE_NAME := hope

IN_CONTAINER := docker run --rm -it --env-file=.env -w /src -v "ssh:/root/.ssh" -p 8067:8067 registry.internal.aleemhaji.com/$(DOCKER_IMAGE_NAME)
HOPE_CONTAINER := $(IN_CONTAINER) hope --config hope.yaml

.PHONY: all
all: $(BIN_NAME)

$(BIN_NAME): $(SRC)
	@mkdir -p $(BUILD_DIR)
	version="$${VERSION:-$$(git describe --dirty)}"; \
	$(GO) build -o $(BIN_NAME) -ldflags="-X github.com/Eagerod/hope/cmd/hope.VersionBuild=$$version" $(MAIN_FILE)


.PHONY: publish
publish: $(PUBLISH)

$(PUBLISH):
	rm -f $(BIN_NAME)
	GOOS_GOARCH="$$(basename $@)" \
	GOOS="$$(cut -d '-' -f 1 <<< "$$GOOS_GOARCH")" \
	GOARCH="$$(cut -d '-' -f 2 <<< "$$GOOS_GOARCH")" \
		$(MAKE) $(BIN_NAME)
	mkdir -p $$(dirname "$@")
	mv $(BIN_NAME) $@


.PHONY: install isntall
install isntall: $(INSTALLED_NAME)

$(INSTALLED_NAME): $(BIN_NAME)
	cp $(BIN_NAME) $(INSTALLED_NAME)

.PHONY: test
test: $(SRC) $(BIN_NAME)
	@$(GO) vet ./...
	@staticcheck ./...
	@if [ -z $$T ]; then \
		$(GO) test -v ./...; \
	else \
		$(GO) test -v ./... -run $$T; \
	fi

# Run a full suite of tests to make sure more than just the most basic of
#   boundaries is functional.
# Create a bunch of resources using a reasonably well defined process, and
#   clean them up when done.
# Tests are broken down in terms of their complexity, so that long-running
#   tasks, like imaging fresh VMs can be optionally ignored for routine
#   testing.
.PHONY: system-test
system-test: system-test-1
	$(MAKE) system-test-clean

.PHONY: system-test-clean
system-test-clean: system-test-5-clean system-test-4-clean system-test-2-clean

.PHONY: system-test-1
system-test-1: $(BIN_NAME)
	$(HOPE_CONTAINER) vm image -f test-kubernetes-node
	$(MAKE) system-test-2

.PHONY: system-test-2
system-test-2: $(BIN_NAME)
	$(HOPE_CONTAINER) vm create test-kubernetes-node test-load-balancer
	$(HOPE_CONTAINER) vm start test-load-balancer

	$(HOPE_CONTAINER) vm create test-kubernetes-node test-master-01
	$(HOPE_CONTAINER) vm start test-master-01
	$(HOPE_CONTAINER) vm create test-kubernetes-node test-master-02
	$(HOPE_CONTAINER) vm start test-master-02
	$(HOPE_CONTAINER) vm create test-kubernetes-node test-master-03
	$(HOPE_CONTAINER) vm start test-master-03

	$(HOPE_CONTAINER) vm create test-kubernetes-node test-node-01
	$(HOPE_CONTAINER) vm start test-node-01

	$(MAKE) system-test-3

.PHONY: system-test-2-clean
system-test-2-clean: $(BIN_NAME)
	$(HOPE_CONTAINER) vm stop test-load-balancer
	$(HOPE_CONTAINER) vm delete test-load-balancer
	$(HOPE_CONTAINER) vm stop test-master-01
	$(HOPE_CONTAINER) vm delete test-master-01
	$(HOPE_CONTAINER) vm stop test-master-02
	$(HOPE_CONTAINER) vm delete test-master-02
	$(HOPE_CONTAINER) vm stop test-master-03
	$(HOPE_CONTAINER) vm delete test-master-03
	$(HOPE_CONTAINER) vm stop test-node-01
	$(HOPE_CONTAINER) vm delete test-node-01

.PHONY: system-test-3
system-test-3: $(BIN_NAME)
	@# Wait for the VM to finish powering on, and getting an IP address...
	$(HOPE_CONTAINER) vm ip test-load-balancer
	$(IN_CONTAINER) bash -c "sshpass -p packer hope --config hope.yaml node ssh test-load-balancer"

	$(HOPE_CONTAINER) vm ip test-master-01
	$(IN_CONTAINER) bash -c "sshpass -p packer hope --config hope.yaml node ssh test-master-01"
	$(HOPE_CONTAINER) vm ip test-master-02
	$(IN_CONTAINER) bash -c "sshpass -p packer hope --config hope.yaml node ssh test-master-02"
	$(HOPE_CONTAINER) vm ip test-master-03
	$(IN_CONTAINER) bash -c "sshpass -p packer hope --config hope.yaml node ssh test-master-03"

	$(HOPE_CONTAINER) vm ip test-node-01
	$(IN_CONTAINER) bash -c "sshpass -p packer hope --config hope.yaml node ssh test-node-01"

	$(MAKE) system-test-4

.PHONY: system-test-4
system-test-4: $(BIN_NAME)
	$(HOPE_CONTAINER) node hostname test-load-balancer testapi
	$(HOPE_CONTAINER) node hostname test-master-01 test-master-01
	$(HOPE_CONTAINER) node hostname test-master-02 test-master-02
	$(HOPE_CONTAINER) node hostname test-master-03 test-master-03
	$(HOPE_CONTAINER) node hostname test-node-01 test-node-01

	$(HOPE_CONTAINER) node init -f test-load-balancer
	$(HOPE_CONTAINER) node init -f test-master-01
	$(HOPE_CONTAINER) node init -f test-master-02
	$(HOPE_CONTAINER) node init -f test-master-03
	$(HOPE_CONTAINER) node init -f test-node-01

	$(MAKE) system-test-5

.PHONY: system-test-4-clean
system-test-4-clean: $(BIN_NAME)
	$(HOPE_CONTAINER) node reset -f test-node-01
	$(HOPE_CONTAINER) node reset -f test-master-01
	$(HOPE_CONTAINER) node reset -f test-master-02
	$(HOPE_CONTAINER) node reset -f test-master-03


.PHONY: system-test-5
system-test-5: $(BIN_NAME)
	@# For whatever reason, docker output doesn't do stderr properly.
	@# Stderr messages are included in the output here.
	@n_resources="$$($(HOPE_CONTAINER) list | wc -l | tr -d ' ')"; \
	if [ $$n_resources -ne 11 ]; then \
		echo >&2 "Incorrect number of resources found ($$n_resources)"; \
		exit 1; \
	fi

	$(HOPE_CONTAINER) deploy calico
	METALLB_SYSTEM_MEMBERLIST_SECRET_KEY="$$(openssl rand -base64 128 | tr -d '\n')" $(HOPE_CONTAINER) deploy -t network

	@# Wait a bit until nodes register as ready.
	@# It can take a while for nodes to register as ready after installing the
	@#   network plugin.
	while true; do \
		n_ready_nodes="$$($(HOPE_CONTAINER) -- kubectl get nodes -o template='{{range .items}}{{range .status.conditions}}{{if eq .reason "KubeletReady"}}{{.status}}{{"\n"}}{{end}}{{end}}{{end}}' | grep "True" | wc -l)"; \
		if [ $$n_ready_nodes -eq 4 ]; then \
			break; \
		else \
			echo >&2 "Only $$n_ready_nodes/4 nodes are ready. Waiting 5 seconds before next poll"; \
			sleep 5; \
		fi; \
	done

	$(HOPE_CONTAINER) deploy -t database

	$(HOPE_CONTAINER) shell -l app=mysql -- mysql -u root -e "SELECT * FROM test.abc;"

.PHONY: system-test-5-clean
system-test-5-clean: $(BIN_NAME)
	$(HOPE_CONTAINER) remove -t database
	$(HOPE_CONTAINER) remove calico
	METALLB_SYSTEM_MEMBERLIST_SECRET_KEY="$$(openssl rand -base64 128 | tr -d '\n')" $(HOPE_CONTAINER) remove -t network

.PHONY: interface-test
interface-test: $(BIN_NAME)
	@if [ -z $$T ]; then \
		$(GO) test -v main_test.go; \
	else \
		$(GO) test -v main_test.go -run $$T; \
	fi

$(COVERAGE_FILE): $(SRC_WITH_TESTS)
	$(GO) test -v --coverprofile=$(COVERAGE_FILE) ./...

.PHONY: coverage
coverage: $(COVERAGE_FILE)
	$(GO) tool cover -func=$(COVERAGE_FILE)

.PHONY: pretty-coverage
pretty-coverage: $(COVERAGE_FILE)
	$(GO) tool cover -html=$(COVERAGE_FILE)

.PHONY: fmt
fmt:
	@$(GO) fmt ./...

.PHONY: clean
clean:
	rm -rf $(COVERAGE_FILE) $(BUILD_DIR) $(PUBLISH_DIR)

.PHONY: container
container: $(BIN_NAME)
	@version="$$(git describe --dirty | sed 's/^v//')"; \
	docker build . --build-arg VERSION="v$$version" -t "registry.internal.aleemhaji.com/$(DOCKER_IMAGE_NAME):$$version"
