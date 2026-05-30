# Makefile for warproot
# Default goal prints available commands so running `make` shows help

BINARY := warproot
PREFIX ?= /usr/local
INSTALL_DIR ?= $(PREFIX)/bin
MODE ?= info

# Default target
.DEFAULT_GOAL := help

.PHONY: help build test test-null test-info test-err test-debug install clean bin log

help:
	@echo "Usage: make [target]"
	@echo
	@echo "Available targets:"
	@echo "  build            Build the binary (output: $(BINARY))"
	@echo "  test             Build the binary, run tests, then (optionally) create latest.log"
	@echo "  test-null        Run tests and ensure no latest.log is present"
	@echo "  test-info        Run tests and create latest.log with info level"
	@echo "  test-err         Run tests and create latest.log with error-only level"
	@echo "  test-debug       Run tests and create latest.log with debug level"
	@echo "  clean            Remove built binary and logs"
	@echo "  bin              Remove built binary"
	@echo "  log              Remove logs (latest.log)"
	@echo "  install          Install binary to $(INSTALL_DIR) (run with sudo if required)"
	@echo
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test MODE=info"
	@echo "  make test-info"
	@echo "  sudo make install"

build:
	@echo "Building $(BINARY)..."
	@go build -o $(BINARY) .

# Build first, run tests, and then create (or not) the latest.log according to MODE.
# MODE may be: null, info, err, debug
test: build
	@echo "Running tests..."
	@sh -c 'if ! go test ./...; then rv=$$?; $(MAKE) bin; exit $$rv; fi; \
	case "$(MODE)" in \
		null) echo "Ensuring no latest.log"; rm -f latest.log ;; \
		info) echo "Creating latest.log (info)"; ./$(BINARY) --help --loglevel=info >/dev/null 2>&1 || true ;; \
		err) echo "Creating latest.log (err)"; ./$(BINARY) --help --loglevel=err >/dev/null 2>&1 || true ;; \
		debug) echo "Creating latest.log (debug)"; ./$(BINARY) --help --loglevel=debug >/dev/null 2>&1 || true ;; \
		*) echo "Unknown MODE $(MODE), creating info log"; ./$(BINARY) --help --loglevel=info >/dev/null 2>&1 || true ;; \
	esac; $(MAKE) bin'

# Convenience wrappers
test-null:
	@$(MAKE) test MODE=null

test-info:
	@$(MAKE) test MODE=info

test-err:
	@$(MAKE) test MODE=err

test-debug:
	@$(MAKE) test MODE=debug

install: build
	@echo "Installing $(BINARY) to $(INSTALL_DIR)/"
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY) $(INSTALL_DIR)/
	@chmod 0755 $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

# Remove binary
bin:
	@echo "Removing binary $(BINARY)"
	@rm -f $(BINARY)

# Remove logs
log:
	@echo "Removing logs"
	@rm -f latest.log || true
	@rm -rf logs || true

# Convenience target to remove both binary and logs
clean: bin log
	@true
