# Makefile for building Go tools

# Variables
GO = go
SRC_DIR = src
BIN_DIR = bin
INSTALL_DIR = $(HOME)/bin
TARGETS = bsg uptest
BINARIES = $(addprefix $(BIN_DIR)/, $(TARGETS))
SCRIPTS = $(filter-out $(BINARIES), $(wildcard $(BIN_DIR)/*))

# Default target
all: $(BINARIES)

# Build bsg
$(BIN_DIR)/bsg: $(SRC_DIR)/bsg.go | $(BIN_DIR)
	$(GO) build -o $@ $<

# Build uptest
$(BIN_DIR)/uptest: $(SRC_DIR)/uptest.go | $(BIN_DIR)
	$(GO) build -o $@ $<
	@echo "Built uptest successfully"

# Create bin directory if it doesn't exist
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Clean built binaries
clean:
	rm -f $(BINARIES)

# Install binaries to ~/bin
install: all | $(INSTALL_DIR)
	cp $(BINARIES) $(SCRIPTS) $(INSTALL_DIR)/
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

# Build only uptest
uptest: $(BIN_DIR)/uptest

# Docker targets
docker-build:
	docker build -f network/Dockerfile -t uptest:latest .

docker-run:
	cd network && docker-compose up -d

docker-stop:
	cd network && docker-compose down

docker-logs:
	cd network && docker-compose logs -f uptest

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Build all binaries (default)"
	@echo "  uptest       - Build uptest only"
	@echo "  clean        - Remove built binaries"
	@echo "  install      - Install all binaries from ./bin to ~/bin"
	@echo "  deps         - Download Go module dependencies"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format Go source code"
	@echo "  vet          - Run Go vet static analysis"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with docker-compose"
	@echo "  docker-stop  - Stop Docker container"
	@echo "  docker-logs  - View Docker logs"
	@echo "  help         - Show this help message"

# Declare phony targets
.PHONY: all clean install deps test fmt vet uptest docker-build docker-run docker-stop docker-logs help
