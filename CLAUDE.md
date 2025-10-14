# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Othello is a Go-based AI agent that integrates local language models (via Ollama) with Model Context Protocol (MCP) servers to provide intelligent assistance through both terminal UI and CLI interfaces. The agent enables tool discovery, execution, and conversation management in a local-first architecture.

## Development Commands

### Building and Running
```bash
# Build the application
go build -o othello cmd/othello/main.go

# Run the interactive TUI
./othello

# Show version information
./othello version

# Show current configuration
./othello config show

# Create default configuration
./othello config init
```

### MCP Server Management
```bash
# List configured MCP servers
./othello mcp list

# Add a new MCP server
./othello mcp add <name> <command> [args...]

# Remove an MCP server
./othello mcp remove <name>

# Show details of specific server
./othello mcp show <name>
```

### Testing
```bash
# Run all tests
go test ./...

# Run internal package tests only
go test ./internal/...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...

# Run integration tests (has build issues - fix disconnect calls)
go test -v ./integration_test.go
```

### Development Scripts
```bash
# Quick build and test script
./test_mcp.sh

# Week 3 development prep script
./week3_startup.sh
```

## Architecture Overview

The codebase follows a clean layered architecture with clear separation of concerns:

### Core Packages
- **`cmd/othello/`** - Main CLI application entry point with Cobra commands
- **`internal/agent/`** - Core agent orchestration and conversation management
- **`internal/mcp/`** - MCP client implementation and server management
- **`internal/model/`** - Model interface abstractions (Ollama integration)
- **`internal/tui/`** - Terminal UI implementation using Bubbletea
- **`internal/config/`** - Configuration management with Viper
- **`internal/storage/`** - Data persistence layer (SQLite)

### Key Components

**Agent Core** (`internal/agent/agent.go`)
- Main orchestrator managing agent lifecycle
- Coordinates between MCP clients, models, and TUI
- Handles conversation state and tool execution

**MCP Manager** (`internal/mcp/`)
- Manages connections to multiple MCP servers
- Handles tool discovery, validation, and execution
- Implements JSON-RPC 2.0 over STDIO transport

**TUI Application** (`internal/tui/`)
- Multi-view terminal interface (chat, server management, help)
- Real-time status updates and keyboard navigation
- Built with Charmbracelet Bubbletea framework

## Configuration

The application uses a hierarchical configuration system:

1. **Main config**: `~/.othello/config.yaml` (model, Ollama, TUI settings)
2. **MCP servers**: `~/.othello/mcp.json` (server configurations)
3. **History**: `~/.othello/history.db` (SQLite conversation storage)

## Development Patterns

### Error Handling
- Consistent error wrapping with context: `fmt.Errorf("operation failed: %w", err)`
- Custom error types for MCP protocol errors
- Graceful degradation when servers are unavailable

### Concurrency
- Go routines for parallel MCP server connections
- Channel-based communication for real-time updates
- Context cancellation for timeout handling

### Testing
- Interface-based mocking for unit tests
- Separate integration tests with real MCP servers
- Table-driven tests for comprehensive coverage

## Current Development Status

The project is actively developed with recent focus on:
- **Week 5**: Tool result processing and conversation fixes
- **TDD approach**: RED-GREEN-REFACTOR cycles implemented
- **Tool validation**: Enhanced parameter validation and error handling
- **MCP integration**: Stable connection management and tool discovery

## Important Implementation Notes

### MCP Protocol
- Uses JSON-RPC 2.0 over STDIO transport
- Implements proper initialization handshake
- Supports tool discovery and execution
- Handles real-time notifications

### Model Integration
- Primary backend: Ollama HTTP API
- Configurable model parameters (temperature, max tokens)
- Streaming and non-streaming response support

### Data Storage
- SQLite for conversation history
- JSON files for configuration
- No external database dependencies

## Known Issues

1. **Integration test failures**: `client.Disconnect()` calls missing context parameter
2. **Test command**: `go test` with `-dry-run` flag not supported (use standard test commands)

## Dependencies

Key external dependencies:
- **Cobra**: CLI framework
- **Viper**: Configuration management
- **Bubbletea/Lipgloss**: Terminal UI
- **SQLite**: Embedded database
- **Testify**: Testing framework

The project follows Go 1.25 standards and maintains compatibility with the latest Go toolchain.