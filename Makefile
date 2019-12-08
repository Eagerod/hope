GO := go

MAIN_FILE := main.go

BUILD_DIR := build
EXECUTABLE := hope
BIN_NAME := $(BUILD_DIR)/$(EXECUTABLE)

CMD_PACKAGE_DIR := ./cmd/hope
PACKAGE_PATHS := $(CMD_PACKAGE_DIR)

SRC := $(shell find . -iname "*.go" -and -not -name "*_test.go") $(CMD_PACKAGE_DIR)/version.go

.PHONY: all
all: $(BIN_NAME)

$(BIN_NAME): $(SRC)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BIN_NAME) $(MAIN_FILE)

.PHONY: install
install: $(BIN_NAME)
	cp $(BIN_NAME) /usr/local/bin/$(EXECUTABLE)

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
test-cover:
	$(GO) test -v --coverprofile=coverage.out $(PACKAGE_PATHS)

.PHONY: coverage
coverage: test-cover
	$(GO) tool cover -func=coverage.out

.INTERMEDIATE: $(CMD_PACKAGE_DIR)/version.go
$(CMD_PACKAGE_DIR)/version.go:
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
