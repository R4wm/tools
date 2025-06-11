# Makefile for building Go tools

# Variables
GO = go
SRC_DIR = src
BIN_DIR = bin
INSTALL_DIR = $(HOME)/bin
TARGET = bs
SOURCE = $(SRC_DIR)/$(TARGET).go
BINARY = $(BIN_DIR)/bsg

# Default target
all: $(BINARY)

# Build the binary
$(BINARY): $(SOURCE) | $(BIN_DIR)
	$(GO) build -o $(BINARY) $(SOURCE)

# Create bin directory if it doesn't exist
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Clean built binaries
clean:
	rm -f $(BINARY)

# Install binaries to ~/bin
install: | $(INSTALL_DIR)
	cp -r $(BIN_DIR)/* $(INSTALL_DIR)/
	@echo "Installed all binaries from $(BIN_DIR) to $(INSTALL_DIR)"

# Create install directory if it doesn't exist
$(INSTALL_DIR):
	mkdir -p $(INSTALL_DIR)

# Install dependencies (if any)
deps:
	$(GO) mod download

# Run tests (if any)
test:
	$(GO) test ./...

# Format Go source code
fmt:
	$(GO) fmt ./$(SRC_DIR)/...

# Run Go vet for static analysis
vet:
	$(GO) vet ./$(SRC_DIR)/...

# Show help
help:
	@echo "Available targets:"
	@echo "  all     - Build the binary (default)"
	@echo "  clean   - Remove built binaries"
	@echo "  install - Install all binaries from ./bin to ~/bin"
	@echo "  deps    - Download Go module dependencies"
	@echo "  test    - Run tests"
	@echo "  fmt     - Format Go source code"
	@echo "  vet     - Run Go vet static analysis"
	@echo "  help    - Show this help message"

# Declare phony targets
.PHONY: all clean install deps test fmt vet help
