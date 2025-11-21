# Help
help:
  just -l

# Build
build:
  go build -o mcp-language-server

# Install locally
install:
  go install

# Format code
fmt:
  gofmt -w .

# Generate LSP types and methods
generate:
  go run ./cmd/generate

# Run code audit checks
check:
  gofmt -l .
  test -z "$(gofmt -l .)"
  go tool staticcheck ./...
  go tool errcheck ./...
  find . -path "./integrationtests/workspaces" -prune -o \
    -path "./integrationtests/test-output" -prune -o \
    -name "*.go" -print | xargs gopls check
  go tool govulncheck ./...

# Run tests
test:
  go test ./...

# Update snapshot tests
snapshot:
  UPDATE_SNAPSHOTS=true go test ./integrationtests/...

# Build all Docker images
docker-build:
  docker-compose build --parallel

# Run all integration tests in Docker
docker-test:
  docker-compose up --build --abort-on-container-exit

# Run Go integration tests in Docker
docker-test-go:
  docker-compose up --build go-tests

# Run Python integration tests in Docker
docker-test-python:
  docker-compose up --build python-tests

# Run Rust integration tests in Docker
docker-test-rust:
  docker-compose up --build rust-tests

# Run TypeScript integration tests in Docker
docker-test-typescript:
  docker-compose up --build typescript-tests

# Run Clangd integration tests in Docker
docker-test-clangd:
  docker-compose up --build clangd-tests

# Run unit tests in Docker
docker-test-unit:
  docker-compose up --build unit-tests

# Clean up Docker resources
docker-clean:
  docker-compose down -v
  docker system prune -f
