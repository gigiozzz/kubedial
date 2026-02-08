# Contributing to Kubedial

Thank you for your interest in contributing to Kubedial! This document provides guidelines and information about contributing.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

## How to Contribute

### Reporting Bugs

1. Check the [existing issues](https://github.com/gigiozzz/kubedial/issues) to see if the bug has already been reported
2. If not, [create a new issue](https://github.com/gigiozzz/kubedial/issues/new?template=bug_report.md) using the bug report template
3. Include as much detail as possible: steps to reproduce, expected behavior, actual behavior, logs, environment details

### Suggesting Features

1. Check the [existing issues](https://github.com/gigiozzz/kubedial/issues) to see if the feature has already been suggested
2. If not, [create a new issue](https://github.com/gigiozzz/kubedial/issues/new?template=feature_request.md) using the feature request template
3. Describe the feature, its use case, and potential implementation approach

### Pull Requests

1. Fork the repository
2. Create a new branch for your changes: `git checkout -b feature/your-feature-name`
3. Make your changes following the coding guidelines below
4. Write or update tests as needed
5. Run tests: `make test`
6. Run linter: `make lint`
7. Commit your changes with a descriptive message
8. Push to your fork and create a pull request

## Development Setup

### Prerequisites

- Go 1.24 or later
- Docker (for building images)
- kubectl (for testing with Kubernetes)
- golangci-lint (for linting)

### Building

```bash
# Build all binaries
make build

# Build specific component
make build-commander
make build-dialer
```

### Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-short

# Run integration tests only
make test-integration
```

### Linting

```bash
make lint
```

## Coding Guidelines

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (enforced by linter)
- Write meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused

### Project Structure

- Common code goes in `common/`
- Use interfaces for external dependencies
- Place implementation in `internal/` packages
- Integration tests should be in `*_integration_test.go` files

### Testing

- Write unit tests for complex logic
- Skip simple DTOs and struct tests
- Use `testing.Short()` to skip integration tests
- Use client-go fake for repository unit tests
- Use envtest for repository integration tests
- Use httpexpect for endpoint integration tests

### Commit Messages

- Use clear, descriptive commit messages
- Start with a verb in present tense: "Add", "Fix", "Update", "Remove"
- Keep the first line under 72 characters
- Add more details in the body if needed

## Questions?

Feel free to open an issue for any questions about contributing.
