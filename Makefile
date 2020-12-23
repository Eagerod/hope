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
system-test: $(BIN_NAME) system-test-1

.PHONY: system-test-1
system-test-1: $(BIN_NAME)
	$(BIN_NAME) --config hope.yaml vm image beast1 -f some-image
	$(MAKE) system-test-2

.PHONY: system-test-2
system-test-2: $(BIN_NAME)
	@if [ -z $$ESXI_ROOT_PASSWORD ]; then \
		echo >&2 "Must set ESXI_ROOT_PASSWORD, or this process will require manual intervention."; \
		exit 1; \
	fi

	$(BIN_NAME) --config hope.yaml vm create beast1 some-image -n test-master-01
	$(BIN_NAME) --config hope.yaml vm start beast1 test-master-01

	@# Wait for the VM to finish powering on, and getting an IP address...
	$(BIN_NAME) --config hope.yaml vm ip beast1 test-master-01
	sshpass -p packer $(BIN_NAME) --config hope.yaml node ssh test-master-01
	$(BIN_NAME) --config hope.yaml vm stop beast1 test-master-01
	$(BIN_NAME) --config hope.yaml vm delete beast1 test-master-01


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
