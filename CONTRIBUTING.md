# Contributing to browser-use-go

Thank you for your interest in contributing to browser-use-go! We appreciate your time and effort in helping improve this project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Pull Request Process](#pull-request-process)
- [Code Style](#code-style)
- [Testing](#testing)
- [Reporting Issues](#reporting-issues)
- [Feature Requests](#feature-requests)

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report any unacceptable behavior.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/browser-use-go.git
   cd browser-use-go
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Install Playwright browsers:
   ```bash
   go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5101.0 install --with-deps
   ```

## Development Workflow

1. Create a new branch for your feature or bugfix:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b bugfix/issue-number-description
   ```
2. Make your changes
3. Run tests to ensure everything works
4. Commit your changes with a clear and descriptive commit message
5. Push your branch to your fork
6. Open a pull request against the main branch

## Pull Request Process

1. Ensure any install or build dependencies are removed before the end of the layer when doing a build.
2. Update the README.md with details of changes to the interface, this includes new environment variables, exposed ports, useful file locations, and container parameters.
3. Increase the version numbers in any examples files and the README.md to the new version that this Pull Request would represent. The versioning scheme we use is [SemVer](http://semver.org/).
4. Your pull request will be reviewed by the maintainers.

## Code Style

- Follow the standard Go formatting rules (`go fmt`)
- Write clear and concise commit messages
- Document exported functions and types
- Add tests for new functionality
- Keep the code clean and simple

## Testing

Before submitting a pull request, please ensure:

1. All tests pass:
   ```bash
   go test ./...
   ```
2. The code passes all linters:
   ```bash
   golangci-lint run
   ```

## Reporting Issues

When reporting issues, please include:

- A clear title and description
- Steps to reproduce the issue
- Expected vs. actual behavior
- Any relevant logs or error messages
- Your environment details (OS, Go version, etc.)

## Feature Requests

We welcome feature requests! Please open an issue to discuss your idea before implementing it. Include:

- A clear description of the feature
- The problem it solves
- Any alternative solutions you've considered

## License

By contributing, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
