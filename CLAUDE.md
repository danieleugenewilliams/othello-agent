# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Othello is a Go-based AI agent that integrates local language models (via Ollama) with Model Context Protocol (MCP) servers to provide intelligent assistance through both terminal UI and CLI interfaces. The agent enables tool discovery, execution, and conversation management in a local-first architecture.

## Local Memory

Proactively use local-memory MCP to store, retrieve, update, and analyze memories to maintain context and build expertise over time. Store key insights including lessons learned, architectural decisions, development strategies, and project outcomes. Use semantic search and relationship mapping to find relevant memories across all projects and sessions.

### v1.1.0 Unified Tool Architecture

**IMPORTANT**: v1.1.0 introduces 8 unified MCP tools that replace the previous 20+ individual tools. Each unified tool uses operation types to provide multiple functionalities:

1. **`store_memory`** - Store new memories with metadata
2. **`search`** - Unified search with operation types:
   - `search_type: "semantic"` - AI-powered semantic search
   - `search_type: "tags"` - Tag-based filtering
   - `search_type: "date_range"` - Date range filtering
   - `search_type: "hybrid"` - Combined search capabilities
3. **`analysis`** - Unified analysis with operation types:
   - `analysis_type: "question"` - AI-powered Q&A
   - `analysis_type: "summarize"` - Memory summarization
   - `analysis_type: "analyze"` - Pattern analysis
   - `analysis_type: "temporal_patterns"` - Learning progression tracking
4. **`relationships`** - Unified relationships with operation types:
   - `relationship_type: "find_related"` - Find related memories
   - `relationship_type: "discover"` - AI-powered relationship discovery
   - `relationship_type: "create"` - Create memory relationships
   - `relationship_type: "map_graph"` - Generate relationship graphs
5. **`categories`** - Unified categorization with operation types:
   - `categories_type: "list"` - List all categories
   - `categories_type: "create"` - Create new categories
   - `categories_type: "categorize"` - AI-powered memory categorization
6. **`domains`** - Unified domain management with operation types:
   - `domains_type: "list"` - List all domains
   - `domains_type: "create"` - Create new domains
   - `domains_type: "stats"` - Domain statistics
7. **`sessions`** - Unified session management with operation types:
   - `sessions_type: "list"` - List all sessions
   - `sessions_type: "stats"` - Session statistics
8. **`stats`** - Unified statistics with operation types:
   - `stats_type: "session"` - Session statistics
   - `stats_type: "domain"` - Domain statistics
   - `stats_type: "category"` - Category statistics

**Key Benefits**:
- **Token Optimization**: Response format controls (`detailed`, `concise`, `ids_only`, `summary`)
- **Session Filtering**: `session_filter_mode` parameter for cross-session access
- **Simplified Interface**: Fewer tools to manage with more functionality
- **Backwards Compatibility**: Legacy individual tools still supported but deprecated

## Development Commands

### Building and Running
```bash
# Build the application
go build -o ./bin/othello cmd/othello/main.go

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