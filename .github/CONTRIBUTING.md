# Contributing to Othello AI Agent

Thank you for your interest in contributing to Othello! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Guidelines](#contributing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)
- [Development Workflow](#development-workflow)

## Code of Conduct

This project adheres to a code of conduct that we expect all contributors to follow. Please be respectful and constructive in all interactions.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/othello-agent.git
   cd othello-agent
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/danieleugenewilliams/othello-agent.git
   ```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- [Ollama](https://ollama.ai) (for testing with local models)

### Setup Steps

1. **Install development tools**:
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install github.com/air-verse/air@latest  # Hot reload
   ```

2. **Initialize the project**:
   ```bash
   go mod tidy
   ```

3. **Run tests**:
   ```bash
   go test ./...
   ```

4. **Run linting**:
   ```bash
   golangci-lint run
   ```

## Contributing Guidelines

### Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` to format your code
- Run `golangci-lint` before submitting
- Write clear, self-documenting code
- Add comments for public APIs and complex logic

### Commit Messages

Use conventional commit format:
```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Maintenance tasks

Example:
```
feat(mcp): add support for HTTP transport

Implement HTTP transport for MCP server connections
in addition to the existing STDIO transport.

Closes #123
```

### Testing

- Write unit tests for new functionality
- Ensure all tests pass before submitting
- Aim for good test coverage (>80%)
- Use table-driven tests where appropriate
- Mock external dependencies

### Documentation

- Update documentation for new features
- Add godoc comments for public APIs
- Update README.md if needed
- Keep documentation in sync with code changes

## Pull Request Process

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the guidelines above

3. **Test thoroughly**:
   ```bash
   go test ./...
   golangci-lint run
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: your descriptive commit message"
   ```

5. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request** on GitHub

7. **Address review feedback** if any

### PR Requirements

- [ ] Tests pass
- [ ] Code is properly formatted
- [ ] Documentation is updated
- [ ] PR description is clear and complete
- [ ] Related issues are referenced

## Issue Reporting

### Bug Reports

Use the bug report template and include:
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details
- Relevant logs or error messages

### Feature Requests

Use the feature request template and include:
- Clear description of the desired feature
- Use case and motivation
- Proposed implementation ideas
- Acceptance criteria

## Development Workflow

### Project Structure

```
othello-agent/
├── cmd/othello/           # Main application entry point
├── internal/              # Private application code
│   ├── agent/            # Core agent logic
│   ├── config/           # Configuration management
│   ├── mcp/              # MCP client implementation
│   ├── model/            # Model interface
│   ├── storage/          # Database operations
│   ├── tui/              # Terminal UI
│   └── cli/              # CLI commands
├── pkg/                  # Public library code
├── docs/                 # Documentation
└── .github/              # GitHub templates and workflows
```

### Architecture Principles

- **Clean Architecture**: Separate concerns into layers
- **Dependency Injection**: Use interfaces for testability
- **Error Handling**: Wrap errors with context
- **Concurrency**: Use goroutines and channels effectively
- **Configuration**: Centralized configuration management

### Key Patterns

#### Error Handling
```go
if err != nil {
    return fmt.Errorf("failed to connect to server %s: %w", serverName, err)
}
```

#### Context Usage
```go
func (s *Service) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
    // Always pass context through the call chain
    return s.client.Call(ctx, req)
}
```

#### Interface Design
```go
type ModelInterface interface {
    Generate(ctx context.Context, prompt string, opts GenerateOptions) (*Response, error)
    IsAvailable(ctx context.Context) bool
}
```

## Getting Help

- Check the [documentation](../docs/) first
- Search existing [issues](https://github.com/danieleugenewilliams/othello-agent/issues)
- Join [discussions](https://github.com/danieleugenewilliams/othello-agent/discussions)
- Ask questions in issues or discussions

## Recognition

Contributors will be recognized in the project documentation and release notes. Thank you for helping make Othello better!

---

By contributing to this project, you agree that your contributions will be licensed under the same license as the project (MIT License).