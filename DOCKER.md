# Docker Setup for Integration Tests

This directory contains Docker configurations to run the MCP Language Server integration tests in isolated environments.

## Available Dockerfiles

### Individual Language Test Suites

- **Dockerfile-base** - Base image with Go 1.24 for unit tests
- **Dockerfile-go** - Go integration tests (includes gopls)
- **Dockerfile-python** - Python integration tests (includes Python 3.10 + pyright)
- **Dockerfile-rust** - Rust integration tests (includes Rust stable + rust-analyzer)
- **Dockerfile-typescript** - TypeScript integration tests (includes Node.js 20 + typescript-language-server)
- **Dockerfile-clangd** - Clangd integration tests (includes clang-16 + clangd-16 + bear)

## Quick Start

### Using Docker Compose (Recommended)

Run all test suites in parallel:
```bash
docker-compose up --build
```

Run a specific test suite:
```bash
# Go integration tests
docker-compose up --build go-tests

# Python integration tests
docker-compose up --build python-tests

# Rust integration tests
docker-compose up --build rust-tests

# TypeScript integration tests
docker-compose up --build typescript-tests

# Clangd integration tests
docker-compose up --build clangd-tests

# Unit tests only
docker-compose up --build unit-tests
```

### Using Docker Directly

#### Build and run individual test suites:

**Go Integration Tests:**
```bash
docker build -t mcp-go-tests -f Dockerfile-go .
docker run --rm mcp-go-tests
```

**Python Integration Tests:**
```bash
docker build -t mcp-python-tests -f Dockerfile-python .
docker run --rm mcp-python-tests
```

**Rust Integration Tests:**
```bash
docker build -t mcp-rust-tests -f Dockerfile-rust .
docker run --rm mcp-rust-tests
```

**TypeScript Integration Tests:**
```bash
docker build -t mcp-typescript-tests -f Dockerfile-typescript .
docker run --rm mcp-typescript-tests
```

**Clangd Integration Tests:**
```bash
docker build -t mcp-clangd-tests -f Dockerfile-clangd .
docker run --rm mcp-clangd-tests
```

## Interactive Development

To run tests interactively or debug issues:

```bash
# Start a container with a shell (use any language-specific image you need)
docker run -it --rm -v $(pwd):/workspace mcp-go-tests bash

# Inside the container, you can run specific tests:
go test -v ./integrationtests/tests/go/...

# Or run a single test
go test -v -run TestSpecificTest ./integrationtests/tests/go/...
```

## CI/CD Integration

These Dockerfiles mirror the setup in `.github/workflows/go.yml` and can be used in any CI/CD system that supports Docker.

### Example CI Usage

```yaml
# Example for GitHub Actions
jobs:
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run all integration tests
        run: docker-compose up --build --abort-on-container-exit
```

```yaml
# Example for GitLab CI
test:integration:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker-compose up --build --abort-on-container-exit
```

## Architecture

### Base Image Strategy

All Dockerfiles use `golang:1.24-bookworm` as the base image to ensure consistency and reduce build times through layer caching.

### Language Server Installation

Each Dockerfile installs the specific language server(s) needed:

- **gopls**: `go install golang.org/x/tools/gopls@latest`
- **pyright**: `npm install -g pyright` (requires Node.js)
- **rust-analyzer**: `rustup component add rust-analyzer`
- **typescript-language-server**: `npm install -g typescript typescript-language-server`
- **clangd**: `apt-get install clangd-16`

### Volume Mounting

The docker-compose.yml mounts the current directory into `/workspace` in the container, allowing you to:
- Make changes locally and re-run tests without rebuilding
- Access test output and artifacts on your host machine

## Troubleshooting

### Build Fails

If a build fails, try:
```bash
# Clear Docker cache and rebuild
docker-compose build --no-cache <service-name>
```

### Tests Fail

To debug failing tests:
```bash
# Run interactively (use the appropriate image for your language)
docker run -it --rm -v $(pwd):/workspace mcp-go-tests bash

# Inside the container, check the language server version
gopls version
```

### Clangd Tests Require compile_commands.json

The Clangd tests require `compile_commands.json` to be generated. This is done automatically in the Dockerfile:
```bash
cd integrationtests/workspaces/clangd && bear -- make
```

If you modify the C code, you may need to regenerate this file.

## Performance Tips

1. **Layer Caching**: Docker caches layers. Place frequently changing code (like COPY . .) near the end of the Dockerfile.

2. **Parallel Builds**: Use docker-compose to build all images in parallel:
   ```bash
   docker-compose build --parallel
   ```

3. **Selective Testing**: Only build and run the test suites you need:
   ```bash
   docker-compose up go-tests python-tests
   ```

4. **Multi-stage Builds**: For production, consider multi-stage builds to reduce final image size (not needed for testing).

## Maintenance

When updating language server versions:

1. Update the CI workflow: `.github/workflows/go.yml`
2. Update the corresponding Dockerfile(s)
3. Rebuild and test locally before pushing

### Version References

Current versions (as of the Dockerfile creation):
- Go: 1.24
- Python: 3.10
- Node.js: 20
- Rust: stable (latest)
- Clang/LLVM: 16

## Additional Resources

- [GitHub Actions Workflow](.github/workflows/go.yml) - The CI configuration these Dockerfiles mirror
- [Integration Tests](integrationtests/) - Integration test source code
- [Justfile](justfile) - Additional build and test commands
