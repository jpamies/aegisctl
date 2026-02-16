# ============================================================================
# Aegis — Makefile (Development)
# ============================================================================
# Development-only targets: build, test, lint, format, and clean.
# For aegisctl CLI usage, run the compiled binary directly — see README.md.
# ============================================================================

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------
APP_NAME   := aegisctl
SRC_DIR    := src
BIN_DIR    := bin
BINARY     := $(BIN_DIR)/$(APP_NAME)
GO         := go
GOFLAGS    := -v
INFRA_DIR  := infra

# Build metadata
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# OS-aware binary extension
ifeq ($(OS),Windows_NT)
  BINARY := $(BIN_DIR)/$(APP_NAME).exe
endif

# ---------------------------------------------------------------------------
# Default
# ---------------------------------------------------------------------------
.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------
# Targets
# ---------------------------------------------------------------------------

## help: Show this help message
.PHONY: help
help:
	@echo ""
	@echo "Aegis (aegisctl) — development targets"
	@echo "========================================"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort
	@echo ""

## all: Run fmt, vet, lint, test, and build (full local CI)
.PHONY: all
all: fmt vet lint test build

## build: Compile the aegisctl binary into bin/ (native)
.PHONY: build
build:
	@echo "==> Building $(BINARY)…"
	cd $(SRC_DIR) && $(GO) build $(GOFLAGS) -o ../$(BINARY) ./cmd/aegisctl
	@echo "==> Binary: $(BINARY)"

## build-linux: Cross-compile for Linux amd64
.PHONY: build-linux
build-linux:
	@echo "==> Cross-compiling for linux/amd64…"
	cd $(SRC_DIR) && GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o ../$(BIN_DIR)/$(APP_NAME)-linux-amd64 ./cmd/aegisctl
	@echo "==> Binary: $(BIN_DIR)/$(APP_NAME)-linux-amd64"

## build-windows: Cross-compile for Windows amd64
.PHONY: build-windows
build-windows:
	@echo "==> Cross-compiling for windows/amd64…"
	cd $(SRC_DIR) && GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o ../$(BIN_DIR)/$(APP_NAME).exe ./cmd/aegisctl
	@echo "==> Binary: $(BIN_DIR)/$(APP_NAME).exe"

## build-darwin: Cross-compile for macOS amd64
.PHONY: build-darwin
build-darwin:
	@echo "==> Cross-compiling for darwin/amd64…"
	cd $(SRC_DIR) && GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o ../$(BIN_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/aegisctl
	@echo "==> Binary: $(BIN_DIR)/$(APP_NAME)-darwin-amd64"

## build-all: Cross-compile for Linux, Windows, and macOS (amd64)
.PHONY: build-all
build-all: build-linux build-windows build-darwin
	@echo "==> All cross-compiled binaries in $(BIN_DIR)/"

## test: Run all Go tests with race detection
.PHONY: test
test:
	@echo "==> Running tests…"
	cd $(SRC_DIR) && $(GO) test -race $(GOFLAGS) ./...

## test-cover: Run tests and print coverage summary
.PHONY: test-cover
test-cover:
	@echo "==> Running tests with coverage…"
	cd $(SRC_DIR) && $(GO) test -race -coverprofile=coverage.out ./...
	cd $(SRC_DIR) && $(GO) tool cover -func=coverage.out
	@echo "==> Coverage report: $(SRC_DIR)/coverage.out"

## test-cover-html: Generate HTML coverage report
.PHONY: test-cover-html
test-cover-html: test-cover
	cd $(SRC_DIR) && $(GO) tool cover -html=coverage.out -o coverage.html
	@echo "==> HTML report: $(SRC_DIR)/coverage.html"

## vet: Run go vet static analysis
.PHONY: vet
vet:
	@echo "==> Running go vet…"
	cd $(SRC_DIR) && $(GO) vet ./...

## fmt: Check that all Go files are gofmt-formatted
.PHONY: fmt
fmt:
	@echo "==> Checking gofmt…"
	@cd $(SRC_DIR) && unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "ERROR: unformatted files:"; \
		echo "$$unformatted"; \
		echo "Run 'make fmt-fix' to fix."; \
		exit 1; \
	fi
	@echo "==> All files formatted."

## fmt-fix: Auto-format all Go source files in place
.PHONY: fmt-fix
fmt-fix:
	@echo "==> Formatting Go files…"
	cd $(SRC_DIR) && gofmt -w .
	@echo "==> Done."

## lint: Run staticcheck linter (auto-installs if missing)
.PHONY: lint
lint:
	@echo "==> Running staticcheck…"
	@command -v staticcheck >/dev/null 2>&1 || { \
		echo "==> Installing staticcheck…"; \
		$(GO) install honnef.co/go/tools/cmd/staticcheck@latest; \
	}
	cd $(SRC_DIR) && staticcheck ./...

## clean: Remove build artefacts and coverage files
.PHONY: clean
clean:
	@echo "==> Cleaning…"
	rm -rf $(BIN_DIR)/$(APP_NAME) $(BIN_DIR)/$(APP_NAME).exe $(BIN_DIR)/release
	rm -f $(SRC_DIR)/coverage.out $(SRC_DIR)/coverage.html
	@echo "==> Clean."

## install: Build and install aegisctl into /usr/local/bin (Linux/WSL)
.PHONY: install
install: build
	@echo "==> Installing $(APP_NAME) to /usr/local/bin/…"
	sudo cp $(BIN_DIR)/$(APP_NAME) /usr/local/bin/$(APP_NAME)
	sudo chmod +x /usr/local/bin/$(APP_NAME)
	@echo "==> Installed: /usr/local/bin/$(APP_NAME)"

## install-windows: Cross-compile and install .exe into ~/bin on the Windows host
WINDOWS_BIN := $(HOME)/bin
.PHONY: install-windows
install-windows: build-windows
	@echo "==> Installing $(APP_NAME).exe to $(WINDOWS_BIN)/…"
	@mkdir -p $(WINDOWS_BIN)
	cp $(BIN_DIR)/$(APP_NAME).exe $(WINDOWS_BIN)/$(APP_NAME).exe
	@echo "==> Installed: $(WINDOWS_BIN)/$(APP_NAME).exe"
	@echo ""
	@echo "    Make sure $(WINDOWS_BIN) is in your Windows PATH."
	@echo "    PowerShell (run once):"
	@echo '      $$p = [Environment]::GetEnvironmentVariable("PATH","User")'
	@echo '      [Environment]::SetEnvironmentVariable("PATH","$$p;%USERPROFILE%\bin","User")'

# ---------------------------------------------------------------------------
# Bicep / IaC validation (requires Azure CLI with Bicep)
# ---------------------------------------------------------------------------

## bicep-build: Validate Bicep templates
.PHONY: bicep-build
bicep-build:
	@echo "==> Validating Bicep templates…"
	az bicep build --file $(INFRA_DIR)/main.bicep
	@echo "==> Bicep validation passed."

## bicep-lint: Lint Bicep templates
.PHONY: bicep-lint
bicep-lint:
	@echo "==> Linting Bicep templates…"
	az bicep lint --file $(INFRA_DIR)/main.bicep
	@echo "==> Bicep lint passed."

# ---------------------------------------------------------------------------
# Release / cross-compile
# ---------------------------------------------------------------------------

## release: Cross-compile for linux, macOS, and Windows (amd64 + arm64)
.PHONY: release
release: clean
	@echo "==> Cross-compiling release binaries…"
	@mkdir -p $(BIN_DIR)/release
	cd $(SRC_DIR) && GOOS=linux   GOARCH=amd64 $(GO) build -o ../$(BIN_DIR)/release/$(APP_NAME)-linux-amd64       ./cmd/aegisctl
	cd $(SRC_DIR) && GOOS=linux   GOARCH=arm64 $(GO) build -o ../$(BIN_DIR)/release/$(APP_NAME)-linux-arm64       ./cmd/aegisctl
	cd $(SRC_DIR) && GOOS=darwin  GOARCH=amd64 $(GO) build -o ../$(BIN_DIR)/release/$(APP_NAME)-darwin-amd64      ./cmd/aegisctl
	cd $(SRC_DIR) && GOOS=darwin  GOARCH=arm64 $(GO) build -o ../$(BIN_DIR)/release/$(APP_NAME)-darwin-arm64      ./cmd/aegisctl
	cd $(SRC_DIR) && GOOS=windows GOARCH=amd64 $(GO) build -o ../$(BIN_DIR)/release/$(APP_NAME)-windows-amd64.exe ./cmd/aegisctl
	@echo "==> Release binaries in $(BIN_DIR)/release/"

## version: Print version metadata from git
.PHONY: version
version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build time: $(BUILD_TIME)"
