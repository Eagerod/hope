GO := go

MAIN_FILE := main.go

BUILD_DIR := build
EXECUTABLE := hope
BIN_NAME := $(BUILD_DIR)/$(EXECUTABLE)
INSTALLED_NAME := /usr/local/bin/$(EXECUTABLE)

CMD_PACKAGE_DIR := ./cmd/hope
PACKAGE_PATHS := $(CMD_PACKAGE_DIR)

UPLOAD_DIR=files

AUTOGEN_VERSION_FILENAME=$(CMD_PACKAGE_DIR)/version-temp.go

SRC := $(shell find . -iname "*.go" -and -not -name "*_test.go") $(AUTOGEN_VERSION_FILENAME)

.PHONY: all
all: $(BIN_NAME)

$(BIN_NAME): $(SRC)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BIN_NAME) $(MAIN_FILE)

.PHONY: install
install: $(INSTALLED_NAME)

$(INSTALLED_NAME): $(BIN_NAME)
	cp $(BIN_NAME) $(INSTALLED_NAME)

.PHONY: test
test: $(SRC)
	@if [ -z $$T ]; then \
		$(GO) test -v $(PACKAGE_PATHS); \
	else \
		$(GO) test -v $(PACKAGE_PATHS) -run $$T; \
	fi

.PHONY: upload
upload:
	@# Uploads all files to the blobstore to be downloaded as needed.
	@if [ -f .env ]; then . .env; fi && set -e && find $(UPLOAD_DIR) -type f | sed 's:$(UPLOAD_DIR)/::' | while read f; do \
		echo >&2 "$(UPLOAD_DIR)/$$f"; \
		blob cp -f "$(UPLOAD_DIR)/$$f" "blob:/hope/$$f"; \
	done

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
	@version=$$(cat VERSION) && \
	build=$$(git rev-parse --short HEAD && if [ ! -z "$$(git diff)" ]; then echo "- dirty"; fi) && \
	printf \
		"%s\n\n%s\n%s\n\n%s\n" \
		"package cmd" \
		"const Version string = \"v$$(printf '%s' $$version)\"" \
		"const Build string = \"$$(printf '%s' $$build)\"" \
		"const VersionBuild string = Version + \"-\" + Build" > $@

.PHONY: pretty-coverage
pretty-coverage: test-cover
	$(GO) tool cover -html=coverage.out

.PHONY: fmt
fmt:
	@$(GO) fmt .
	@$(GO) fmt $(CMD_PACKAGE_DIR)

.PHONY: clean
clean:
	rm -rf coverage.out $(BUILD_DIR)
