GO := go

MAIN_FILE := main.go

BUILD_DIR := build
EXECUTABLE := hope
BIN_NAME := $(BUILD_DIR)/$(EXECUTABLE)
INSTALLED_NAME := /usr/local/bin/$(EXECUTABLE)

CMD_PACKAGE_DIR := ./cmd/hope
PKG_PACKAGE_DIR := ./pkg/*
PACKAGE_PATHS := $(CMD_PACKAGE_DIR) $(PKG_PACKAGE_DIR)

AUTOGEN_VERSION_FILENAME=$(CMD_PACKAGE_DIR)/version-temp.go

SRC := $(shell find . -iname "*.go" -and -not -name "*_test.go") $(AUTOGEN_VERSION_FILENAME)
PUBLISH := publish/linux-amd64 publish/darwin-amd64

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

.PHONY: system-test
system-test: $(BIN_NAME)
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
	@$(GO) fmt .
	@$(GO) fmt $(CMD_PACKAGE_DIR)
	@$(GO) fmt $(PKG_PACKAGE_DIR)

.PHONY: clean
clean:
	rm -rf coverage.out $(BUILD_DIR)
