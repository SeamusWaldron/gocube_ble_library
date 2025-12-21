# Contributing to GoCube

Thank you for your interest in contributing to GoCube! This document provides guidelines and information about contributing to this project.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gocube.git
   cd gocube
   ```
3. Create a branch for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites

- Go 1.22 or later
- A GoCube smart cube (for testing BLE functionality)
- macOS (BLE functionality is currently macOS-only)

### Building

```bash
go build ./...
```

### Running Tests

```bash
go test ./...
```

### Running the CLI

```bash
go run ./cmd/gocube
```

## Code Style

- Follow standard Go conventions and formatting (`gofmt`)
- Use `golangci-lint` for linting
- Add godoc comments to all exported types and functions
- Keep functions focused and reasonably sized

## Pull Request Process

1. Ensure your code builds and all tests pass
2. Update documentation if you're changing functionality
3. Add tests for new features
4. Update CHANGELOG.md with your changes
5. Create a pull request with a clear description of the changes

## Commit Messages

Use clear, descriptive commit messages:

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Keep the first line under 50 characters
- Reference issues and pull requests when applicable

## Reporting Issues

When reporting issues, please include:

- Go version (`go version`)
- Operating system and version
- GoCube model (if applicable)
- Steps to reproduce the issue
- Expected vs actual behavior

## Code of Conduct

This project follows a standard code of conduct. Please be respectful and constructive in all interactions.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
