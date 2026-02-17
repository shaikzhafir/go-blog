.PHONY: build run dev clean tailwind tailwind-prod css help deps

# Include .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Tailwind CLI (override if installed elsewhere, e.g. PATH)
TAILWIND_CLI ?= ./tailwindcss
CSS_IN  := input.css
CSS_OUT := static/output.css

# Build for production: Tailwind (minified) + Go binary
build: tailwind-prod
	go build -o bin/server cmd/server/main.go

# Tailwind for production (minified CSS). Pass config so content paths and theme are used.
tailwind-prod:
	$(TAILWIND_CLI) -i $(CSS_IN) -o $(CSS_OUT) -c tailwind.config.js --minify

# Run the built binary
run: build
	./bin/server

# Development: Go server with auto-reload (requires entr). Run `make tailwind` in another terminal for CSS.
dev:
	find . \( -name "*.go" -o -name "*.html" \) ! -path "./.git/*" ! -name "*_test.go" | entr -r go run cmd/server/main.go

# Tailwind: watch and rebuild CSS (run in a separate terminal during dev)
tailwind:
	$(TAILWIND_CLI) -i $(CSS_IN) -o $(CSS_OUT) -c tailwind.config.js --watch

# One-off CSS build, minified (alias for tailwind-prod)
css: tailwind-prod

# Install Go deps
deps:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/

# List available targets
help:
	@echo "Targets:"
	@echo "  make build         - Tailwind (prod minified) + build Go binary to bin/server"
	@echo "  make run           - build and run server"
	@echo "  make dev           - run server with auto-reload (entr); run 'make tailwind' in another terminal for CSS"
	@echo "  make tailwind      - watch and rebuild Tailwind CSS (run alongside 'make dev')"
	@echo "  make tailwind-prod - one-off Tailwind build, minified (for production)"
	@echo "  make css           - same as tailwind-prod"
	@echo "  make deps          - go mod tidy"
	@echo "  make clean         - remove bin/"
