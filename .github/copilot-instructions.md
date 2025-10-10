# GitHub Copilot Instructions for Othello AI Agent

This file provides context and instructions for GitHub Copilot when working on the Othello AI Agent project.

## Project Overview

Othello is a Go-based local AI assistant that integrates with Model Context Protocol (MCP) servers. It provides:

- Terminal User Interface (TUI) using Bubbletea
- CLI commands using Cobra
- MCP client implementation for tool integration
- Local model execution via Ollama
- SQLite-based conversation history
- Configuration management with Viper

## Collaboration Guidelines
- Never use emojis in communication, code, code comments, documentation, or commit messages.
- I know I am right. You don't need to remind me.
- Never use em dashes (—) in code comments or documentation.

## Architecture Principles

### Code Organization
- Follow Go standard project layout
- Use dependency injection with interfaces
- Implement clean architecture with layered design
- Separate concerns between UI, business logic, and infrastructure

### Key Directories
```
cmd/othello/           # Main application entry point
internal/              # Private application code
├── agent/            # Core agent logic
├── config/           # Configuration management
├── mcp/              # MCP client implementation
├── model/            # Model interface and implementations
├── storage/          # Database and file storage
├── tui/              # Terminal UI components
└── cli/              # CLI command implementations
pkg/                  # Public library code
docs/                 # Comprehensive documentation
```

## Coding Standards

### Go Best Practices
- Use meaningful variable and function names
- Implement proper error handling with wrapped errors
- Use interfaces for testability and flexibility
- Follow effective Go patterns (channels, goroutines, contexts)
- Write comprehensive unit tests

### Project-Specific Patterns

#### Configuration
- All config and data files go in `~/.othello/` directory
- Use Viper for configuration management
- Support YAML, JSON, and environment variables
- Implement configuration validation

#### Error Handling
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to connect to MCP server %s: %w", serverName, err)
}

// Use custom error types for business logic
type MCPConnectionError struct {
    ServerName string
    Err        error
}
```

#### Logging
```go
// Use structured logging
logger.Info("Starting MCP server connection",
    "server", serverName,
    "transport", transport,
    "timeout", timeout)
```

#### Interfaces
```go
// Define interfaces in consuming packages
type ModelInterface interface {
    Generate(ctx context.Context, prompt string, opts GenerateOptions) (*Response, error)
    IsAvailable(ctx context.Context) bool
}
```

## MCP Integration

### Server Management
- Support both STDIO and HTTP transports
- Implement server discovery and registration
- Handle connection lifecycle (connect, disconnect, reconnect)
- Provide tool discovery and caching

### Tool Execution
- Validate tool parameters against schemas
- Handle synchronous and asynchronous tool calls
- Implement proper timeout and cancellation
- Log all tool executions for debugging

## TUI Development

### Bubbletea Patterns
```go
// Use the standard Bubble Tea model
type Model struct {
    conversation []Message
    input        textinput.Model
    viewport     viewport.Model
    // ... other state
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle key events
    case ResponseMsg:
        // Handle async responses
    }
    return m, nil
}
```

### UI Components
- Use lipgloss for styling consistently
- Implement responsive layouts
- Handle terminal resize events
- Provide keyboard shortcuts and help

## Testing Guidelines

### Unit Testing
- Test all public interfaces
- Mock external dependencies (MCP servers, models)
- Use testify for assertions and mocks
- Aim for >80% code coverage

### Integration Testing
- Test MCP server interactions
- Test configuration loading
- Test database operations
- Use temporary directories for test data

### Example Test Structure
```go
func TestAgentCore_ProcessQuery(t *testing.T) {
    tests := []struct {
        name    string
        query   string
        want    *Response
        wantErr bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Dependencies

### Core Dependencies
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/mattn/go-sqlite3` - SQLite driver
- `github.com/gorilla/rpc` - JSON-RPC implementation

### Development Dependencies
- `github.com/stretchr/testify` - Testing framework
- `github.com/golangci/golangci-lint` - Linting
- `github.com/air-verse/air` - Hot reload for development

## Common Patterns

### Context Usage
- Always pass context.Context as first parameter
- Use context for cancellation and timeouts
- Implement proper context cancellation handling

### Concurrent Operations
```go
// Use errgroup for concurrent operations
g, ctx := errgroup.WithContext(ctx)
for _, server := range servers {
    server := server // capture loop variable
    g.Go(func() error {
        return connectToServer(ctx, server)
    })
}
if err := g.Wait(); err != nil {
    return fmt.Errorf("failed to connect to servers: %w", err)
}
```

### Configuration Loading
```go
// Use consistent configuration patterns
type Config struct {
    Model   ModelConfig   `yaml:"model"`
    Servers []ServerConfig `yaml:"servers"`
    Storage StorageConfig `yaml:"storage"`
    Logging LoggingConfig `yaml:"logging"`
}
```

## File Naming Conventions

- Use snake_case for file names
- Group related functionality in packages
- Use `_test.go` suffix for test files
- Use `_mock.go` suffix for mock implementations

## Documentation

- Document all public APIs with godoc comments
- Include examples in documentation
- Keep README.md updated with latest features
- Maintain comprehensive docs/ directory

## Security Considerations

- Never log sensitive data (API keys, tokens)
- Validate all external inputs
- Use secure defaults for configuration
- Implement proper file permissions for config files

## Performance Guidelines

- Use connection pooling for database operations
- Implement caching for frequently accessed data
- Use efficient data structures for large datasets
- Profile and benchmark performance-critical code

## Deployment

- Build single static binary with `go build`
- Support cross-platform compilation
- Include version information in builds
- Provide installation scripts for different platforms

When implementing features, prioritize:
1. Correctness and reliability
2. User experience and usability
3. Performance and efficiency
4. Maintainability and testability
5. Security and privacy