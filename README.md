# Othello AI Agent

A Go-based local AI assistant that integrates with Model Context Protocol (MCP) servers to provide intelligent assistance through terminal and CLI interfaces.

## Features

- 🤖 **Local AI Models**: Integrates with Ollama for local language model execution
- 🔧 **MCP Integration**: Seamless connection to Model Context Protocol servers
- 💻 **Terminal UI**: Modern terminal user interface built with Bubbletea
- ⚡ **CLI Commands**: Powerful command-line interface for automation
- 🔐 **Local-First**: All data stays on your machine by default
- 📝 **Conversation History**: Persistent chat history and tool execution logs

## Quick Start

### Prerequisites

- Go 1.21 or later
- [Ollama](https://ollama.ai) (for local models)

### Installation

```bash
# Clone the repository
git clone https://github.com/danieleugenewilliams/othello-agent.git
cd othello-agent

# Build the application
go build -o othello ./cmd/othello

# Run the agent
./othello
```

## Documentation

Comprehensive documentation is available in the [`docs/`](./docs/) directory:

- 📋 [Product Requirements](./docs/PRD.md)
- 📝 [Technical Requirements](./docs/REQUIREMENTS.md)
- 🏗️ [System Architecture](./docs/ARCHITECTURE.md)
- 👤 [User Guide](./docs/USAGE.md)
- ⚙️ [Implementation Guide](./docs/IMPLEMENTATION.md)

## Configuration

Othello stores all configuration and data in `~/.othello/`:

```
~/.othello/
├── config.yaml          # Main configuration
├── history.db           # Conversation history
├── servers.json         # MCP server registry
└── logs/                # Application logs
```

## Development Status

This project is currently in active development. See the [implementation guide](./docs/IMPLEMENTATION.md) for development setup and contribution guidelines.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Links

- **Repository**: [github.com/danieleugenewilliams/othello-agent](https://github.com/danieleugenewilliams/othello-agent)
- **Issues**: [Report bugs and request features](https://github.com/danieleugenewilliams/othello-agent/issues)
- **Discussions**: [Community discussions](https://github.com/danieleugenewilliams/othello-agent/discussions)