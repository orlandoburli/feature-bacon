.PHONY: generate lint lint-proto lint-go test test-cover build clean

# Proto code generation
generate:
	buf generate

# Linting
lint: lint-proto lint-go

lint-proto:
	buf lint

lint-go:
	cd backend && golangci-lint run ./...

# Testing
test:
	cd backend && go test -race ./...

test-cover:
	cd backend && go test -race -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/... ./...
	cd backend && go tool cover -func=coverage.out

# Build all binaries
build:
	@for cmd in backend/cmd/*/; do \
		name=$$(basename "$$cmd"); \
		echo "Building $$name..."; \
		cd backend && go build -o bin/$$name ./cmd/$$name && cd ..; \
	done

clean:
	rm -rf backend/bin backend/coverage.out
