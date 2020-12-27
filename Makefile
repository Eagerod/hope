GO := go

MAIN_FILE := main.go

BUILD_DIR := build
EXECUTABLE := hope
BIN_NAME := $(BUILD_DIR)/$(EXECUTABLE)
INSTALLED_NAME := /usr/local/bin/$(EXECUTABLE)

CMD_PACKAGE_DIR := ./cmd/hope $(dir $(wildcard ./cmd/hope/*/))
PKG_PACKAGE_DIR := ./pkg/*
PACKAGE_PATHS := $(CMD_PACKAGE_DIR) $(PKG_PACKAGE_DIR)

AUTOGEN_VERSION_FILENAME=./cmd/hope/version-temp.go

ALL_GO_DIRS = $(shell find . -iname "*.go" -exec dirname {} \; | sort | uniq)
SRC := $(shell find . -iname "*.go" -and -not -name "*_test.go") $(AUTOGEN_VERSION_FILENAME)
PUBLISH = publish/linux-amd64 publish/darwin-amd64

.PHONY: all
all: $(BIN_NAME)

$(BIN_NAME): $(SRC)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BIN_NAME) $(MAIN_FILE)


.PHONY: publish
publish: $(PUBLISH)

.PHONY: publish/linux-amd64
publish/linux-amd64:
	# Force build; don't let existing versions interfere.
	rm -f $(BIN_NAME)
	GOOS=linux GOARCH=amd64 $(MAKE) $(BIN_NAME)
	mkdir -p $$(dirname "$@")
	mv $(BIN_NAME) $@

.PHONY: publish/darwin-amd64
publish/darwin-amd64:
	# Force build; don't let existing versions interfere.
	rm -f $(BIN_NAME)
	GOOS=darwin GOARCH=amd64 $(MAKE) $(BIN_NAME)
	mkdir -p $$(dirname "$@")
	mv $(BIN_NAME) $@


.PHONY: install isntall
install isntall: $(INSTALLED_NAME)

$(INSTALLED_NAME): $(BIN_NAME)
	cp $(BIN_NAME) $(INSTALLED_NAME)

.PHONY: test
test: $(SRC)
	@if [ -z $$T ]; then \
		$(GO) test -v $(PACKAGE_PATHS); \
	else \
		$(GO) test -v $(PACKAGE_PATHS) -run $$T; \
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
	$(BIN_NAME) --config hope.yaml vm image beast1 -f test-kubernetes-node
	$(MAKE) system-test-2

.PHONY: system-test-2
system-test-2: $(BIN_NAME)
	@if [ -z $$ESXI_ROOT_PASSWORD ]; then \
		echo >&2 "Must set ESXI_ROOT_PASSWORD, or this process will require manual intervention."; \
		exit 1; \
	fi

	$(BIN_NAME) --config hope.yaml vm create beast1 test-kubernetes-node -n test-load-balancer -c 2 -m 512
	$(BIN_NAME) --config hope.yaml vm start beast1 test-load-balancer

	$(BIN_NAME) --config hope.yaml vm create beast1 test-kubernetes-node --name test-master-01 --cpu 2 --memory 2048
	$(BIN_NAME) --config hope.yaml vm start beast1 test-master-01
	$(BIN_NAME) --config hope.yaml vm create beast1 test-kubernetes-node --name test-master-02 --cpu 2 --memory 2048
	$(BIN_NAME) --config hope.yaml vm start beast1 test-master-02
	$(BIN_NAME) --config hope.yaml vm create beast1 test-kubernetes-node --name test-master-03 --cpu 2 --memory 2048
	$(BIN_NAME) --config hope.yaml vm start beast1 test-master-03

	$(BIN_NAME) --config hope.yaml vm create beast1 test-kubernetes-node -n test-node-01 -c 2 -m 4096
	$(BIN_NAME) --config hope.yaml vm start beast1 test-node-01

	$(MAKE) system-test-3

.PHONY: system-test-2-clean
system-test-2-clean: $(BIN_NAME)
	$(BIN_NAME) --config hope.yaml vm stop beast1 test-load-balancer
	$(BIN_NAME) --config hope.yaml vm delete beast1 test-load-balancer
	$(BIN_NAME) --config hope.yaml vm stop beast1 test-master-01
	$(BIN_NAME) --config hope.yaml vm delete beast1 test-master-01
	$(BIN_NAME) --config hope.yaml vm stop beast1 test-master-02
	$(BIN_NAME) --config hope.yaml vm delete beast1 test-master-02
	$(BIN_NAME) --config hope.yaml vm stop beast1 test-master-03
	$(BIN_NAME) --config hope.yaml vm delete beast1 test-master-03
	$(BIN_NAME) --config hope.yaml vm stop beast1 test-node-01
	$(BIN_NAME) --config hope.yaml vm delete beast1 test-node-01

.PHONY: system-test-3
system-test-3: $(BIN_NAME)
	@# Wait for the VM to finish powering on, and getting an IP address...
	$(BIN_NAME) --config hope.yaml vm ip beast1 test-load-balancer
	sshpass -p packer $(BIN_NAME) --config hope.yaml node ssh test-load-balancer

	$(BIN_NAME) --config hope.yaml vm ip beast1 test-master-01
	sshpass -p packer $(BIN_NAME) --config hope.yaml node ssh test-master-01
	$(BIN_NAME) --config hope.yaml vm ip beast1 test-master-02
	sshpass -p packer $(BIN_NAME) --config hope.yaml node ssh test-master-02
	$(BIN_NAME) --config hope.yaml vm ip beast1 test-master-03
	sshpass -p packer $(BIN_NAME) --config hope.yaml node ssh test-master-03

	$(BIN_NAME) --config hope.yaml vm ip beast1 test-node-01
	sshpass -p packer $(BIN_NAME) --config hope.yaml node ssh test-node-01

	$(MAKE) system-test-4

.PHONY: system-test-4
system-test-4: $(BIN_NAME)
	$(BIN_NAME) --config hope.yaml node hostname test-load-balancer testapi
	$(BIN_NAME) --config hope.yaml node hostname test-master-01 test-master-01
	$(BIN_NAME) --config hope.yaml node hostname test-master-02 test-master-02
	$(BIN_NAME) --config hope.yaml node hostname test-master-03 test-master-03
	$(BIN_NAME) --config hope.yaml node hostname test-node-01 test-node-01

	$(BIN_NAME) --config hope.yaml node init -f test-load-balancer
	$(BIN_NAME) --config hope.yaml node init -f test-master-01
	$(BIN_NAME) --config hope.yaml node init -f test-master-02
	$(BIN_NAME) --config hope.yaml node init -f test-master-03
	$(BIN_NAME) --config hope.yaml node init -f test-node-01

	$(MAKE) system-test-5

.PHONY: system-test-4-clean
system-test-4-clean: $(BIN_NAME)
	$(BIN_NAME) --config hope.yaml node reset -f test-node-01
	$(BIN_NAME) --config hope.yaml node reset -f test-master-01
	$(BIN_NAME) --config hope.yaml node reset -f test-master-02
	$(BIN_NAME) --config hope.yaml node reset -f test-master-03


.PHONY: system-test-5
system-test-5: $(BIN_NAME)
	@if [ $$($(BIN_NAME) --config hope.yaml list | wc -l) -ne 8 ]; then \
		echo >&2 "Incorrect number of resources found ($$($(BIN_NAME) --config hope.yaml list | wc -l))"; \
		exit 1; \
	fi

	$(BIN_NAME) --config hope.yaml deploy calico
	METALLB_SYSTEM_MEMBERLIST_SECRET_KEY="$$(openssl rand -base64 128 | tr -d '\n')" $(BIN_NAME) --config hope.yaml deploy -t network

	@# Wait a bit until nodes register as ready.
	@# It can take a while for nodes to register as ready after installing the
	@#   network plugin.
	while true; do \
		n_ready_nodes="$$($(BIN_NAME) --config hope.yaml -- kubectl get nodes -o template='{{range .items}}{{range .status.conditions}}{{if eq .reason "KubeletReady"}}{{.status}}{{"\n"}}{{end}}{{end}}{{end}}' | grep "True" | wc -l)"; \
		if [ $$n_ready_nodes -eq 4 ]; then \
		    break; \
		else \
			echo >&2 "Only $$n_ready_nodes/4 nodes are ready. Waiting 5 seconds before next poll"; \
			sleep 5; \
		fi; \
	done

	$(BIN_NAME) --config hope.yaml deploy -t database

	$(BIN_NAME) --config hope.yaml shell -l app=mysql -- mysql -u root -e "SELECT * FROM test.abc;"

.PHONY: system-test-5-clean
system-test-5-clean: $(BIN_NAME)
	$(BIN_NAME) --config hope.yaml remove -t database
	$(BIN_NAME) --config hope.yaml remove calico
	METALLB_SYSTEM_MEMBERLIST_SECRET_KEY="$$(openssl rand -base64 128 | tr -d '\n')" $(BIN_NAME) --config hope.yaml remove -t network

.PHONY: interface-test
interface-test: $(BIN_NAME)
	@if [ -z $$T ]; then \
		$(GO) test -v main_test.go; \
	else \
		$(GO) test -v main_test.go -run $$T; \
	fi

.PHONY: test-cover
test-cover: $(SRC)
	$(GO) test -v --coverprofile=coverage.out $(PACKAGE_PATHS)

.PHONY: coverage
coverage: test-cover
	$(GO) tool cover -func=coverage.out

.INTERMEDIATE: $(AUTOGEN_VERSION_FILENAME)
$(AUTOGEN_VERSION_FILENAME):
	@version="v$$(cat VERSION)" && \
	build="$$(if [ "$$(git describe)" != "$$version" ]; then echo "-$$(git rev-parse --short HEAD)"; fi)" && \
	dirty="$$(if [ ! -z "$$(git diff; git diff --cached)" ]; then echo "-dirty"; fi)" && \
	printf "package cmd\n\nconst VersionBuild = \"%s%s%s\"" $$version $$build $$dirty > $@

.PHONY: pretty-coverage
pretty-coverage: test-cover
	$(GO) tool cover -html=coverage.out

.PHONY: fmt
fmt:
	@$(GO) fmt $(ALL_GO_DIRS)

.PHONY: clean
clean:
	rm -rf coverage.out $(BUILD_DIR)
