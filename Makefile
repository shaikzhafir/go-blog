.PHONY: build run dev clean

# Include .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Build the project
build:
	go build -o bin/server cmd/server/main.go

# Run the built binary
run: build
	./bin/server

# Development mode with auto-reload (requires entr)
dev:
	find . \( -name "*.go" -o -name "*.html" \) ! -name "*_test.go" | entr -r go run cmd/server/main.go

# Clean build artifacts
clean:
	rm -rf bin/
