# Variables
GOLANGCI_LINT := $(shell which golangci-lint)

BUILD_DIR := build
CMD_DIR = ./cmd/pprs
MAIN_FILE := $(CMD_DIR)/main.go

BINARY_NAME := pprs
ALIAS_NAME := peppers
INSTALL_DIR := $(shell go env GOPATH)/bin

# Default target
all: build

# Build the binary
build:
	mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

# Run the binary
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Tidy: format and vet the code
tidy:
	@go fmt $$(go list ./...)
	@go vet $$(go list ./...)

# Install golangci-lint only if it's not already installed
lint-install:
	@if ! [ -x "$(GOLANGCI_LINT)" ]; then \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

# Lint the code using golangci-lint
# todo reuse var if possible
lint: lint-install
	$(shell which golangci-lint) run

# Install the binary globally with aliases
install:
	@go install ./$(MAIN_FILE)
	ln -sf $(INSTALL_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(ALIAS_NAME)

# Uninstall the binary and remove the alias
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	rm -f $(INSTALL_DIR)/$(ALIAS_NAME)

# Phony targets
.PHONY: all build run tidy lint-install lint install uninstall