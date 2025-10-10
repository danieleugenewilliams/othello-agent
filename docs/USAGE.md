# User Guide
## Othello AI Agent

**Version:** 1.0  
**Date:** October 10, 2025  

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Installation](#installation)
3. [Basic Usage](#basic-usage)
4. [MCP Server Management](#mcp-server-management)
5. [Configuration](#configuration)
6. [Advanced Features](#advanced-features)
7. [Troubleshooting](#troubleshooting)
8. [Examples](#examples)

---

## Quick Start

### Prerequisites

- **Ollama**: Install Ollama for model management
- **Qwen2.5 Model**: Download a Qwen2.5 model (3B or 7B recommended)
- **Terminal**: Modern terminal with 256-color support

### 5-Minute Setup

```bash
# 1. Install Ollama (if not already installed)
curl -fsSL https://ollama.ai/install.sh | sh

# 2. Download Qwen2.5 model
ollama pull qwen2.5:3b

# 3. Download and install Othello
curl -L https://github.com/danieleugenewilliams/othello-agent/releases/latest/download/othello-linux-amd64 -o othello
chmod +x othello
sudo mv othello /usr/local/bin/

# 4. Start the agent
othello
```

### First Conversation

```
┌─ Othello AI Agent ─────────────────────────────────────────┐
│                                                            │
│ Welcome! I'm Othello, your local AI assistant.            │
│ Type your message below and press Enter to chat.          │
│                                                            │
│ You: Hello! What can you help me with?                    │
│                                                            │
│ Othello: Hello! I'm here to help you with various tasks.  │
│ I can assist with file operations, web searches, data     │
│ analysis, and much more through connected tools. What     │
│ would you like to work on today?                          │
│                                                            │
│ > _                                                        │
└────────────────────────────────────────────────────────────┘
```

---

## Installation

### Binary Installation

**Linux/macOS:**
```bash
# Download for your platform
curl -L https://github.com/danieleugenewilliams/othello-agent/releases/latest/download/othello-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o othello

# Make executable and move to PATH
chmod +x othello
sudo mv othello /usr/local/bin/
```

**Windows:**
```powershell
# Download Windows binary
Invoke-WebRequest -Uri "https://github.com/danieleugenewilliams/othello-agent/releases/latest/download/othello-windows-amd64.exe" -OutFile "othello.exe"

# Move to PATH location
Move-Item othello.exe C:\Windows\System32\
```

### Package Managers

**Homebrew (macOS/Linux):**
```bash
brew install othello
```

**Chocolatey (Windows):**
```powershell
choco install othello
```

**Snap (Linux):**
```bash
sudo snap install othello
```

### Building from Source

```bash
# Clone repository
git clone https://github.com/danieleugenewilliams/othello-agent.git
cd othello

# Build
go build -o othello cmd/othello/main.go

# Install
sudo mv othello /usr/local/bin/
```

### Model Setup

Othello works best with Ollama for model management:

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Download Qwen2.5 models
ollama pull qwen2.5:3b      # Faster, less memory
ollama pull qwen2.5:7b      # Better quality, more memory

# Verify installation
ollama list
```

---

## Basic Usage

### Starting the Agent

```bash
# Start interactive TUI mode
othello

# Start with specific model
othello --model qwen2.5:7b

# Start with configuration file
othello --config ./my-config.yaml

# Non-interactive mode (single query)
othello --query "What files are in my home directory?"
```

### TUI Interface

The Terminal User Interface provides several views:

#### Chat View (Main)
- **Input Field**: Type your messages
- **Conversation**: View AI responses and tool usage
- **Status Bar**: Shows model, connected servers, and shortcuts

#### Server Management View
- **Server List**: All connected MCP servers
- **Server Status**: Connection health and tool count
- **Add/Remove**: Manage server connections

#### Help View
- **Keyboard Shortcuts**: Complete shortcut reference
- **Command Help**: Available commands and usage
- **Tool Documentation**: Available tools and their purposes

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Enter` | Send message |
| `Ctrl+C` | Exit application |
| `Ctrl+L` | Clear conversation |
| `Tab` | Switch between views |
| `Ctrl+S` | Toggle server management |
| `Ctrl+H` | Toggle help |
| `↑/↓` | Navigate history |
| `Ctrl+U` | Clear input |

---

## MCP Server Management

### Adding Servers

#### From Command Line
```bash
# Add filesystem server
othello mcp add filesystem /path/to/filesystem-server

# Add with arguments
othello mcp add weather ./weather-server --args="--api-key=your-key"

# Add with environment variables
othello mcp add database ./db-server --env="DB_URL=postgresql://..."

# Add HTTP server
othello mcp add remote-api http://example.com/mcp --transport=http
```

#### From TUI
1. Press `Ctrl+S` to open server management
2. Press `A` to add new server
3. Fill in server details:
   - **Name**: Unique identifier
   - **Command**: Path to server executable
   - **Arguments**: Command line arguments
   - **Environment**: Environment variables
4. Press `Enter` to save

### Managing Servers

```bash
# List all servers
othello mcp list

# Remove server
othello mcp remove filesystem

# Test server connection
othello mcp test filesystem

# Show server details
othello mcp info filesystem

# Export server configurations
othello mcp export > servers.json

# Import server configurations
othello mcp import servers.json
```

### Popular MCP Servers

#### Filesystem Server
```bash
# Install
npm install -g @modelcontextprotocol/server-filesystem

# Add to Othello
othello mcp add filesystem npx --args="@modelcontextprotocol/server-filesystem /path/to/directory"
```

#### Web Search Server
```bash
# Install
npm install -g @modelcontextprotocol/server-web-search

# Add to Othello
othello mcp add websearch npx --args="@modelcontextprotocol/server-web-search"
```

#### Database Server
```bash
# Install
pip install mcp-server-postgres

# Add to Othello
othello mcp add database python --args="-m mcp_server_postgres --connection-string postgresql://..."
```

---

## Configuration

### Configuration File

Othello looks for configuration in these locations (in order):
1. `./othello.yaml` (current directory)
2. `~/.othello/config.yaml` (user config)
3. `/etc/othello/config.yaml` (system config)

### Sample Configuration

```yaml
# ~/.othello/config.yaml

# Model configuration
model:
  type: "ollama"          # "ollama", "gguf", or "http"
  name: "qwen2.5:7b"      # Model name
  temperature: 0.7        # Response creativity (0.0-1.0)
  max_tokens: 2048        # Maximum response length
  context_length: 8192    # Context window size

# Ollama configuration
ollama:
  url: "http://localhost:11434"
  timeout: "30s"

# TUI configuration
tui:
  theme: "default"        # "default", "dark", "light"
  animations: true        # Enable UI animations
  mouse_support: true     # Enable mouse interaction

# MCP configuration
mcp:
  timeout: "10s"          # Server connection timeout
  max_servers: 20         # Maximum concurrent servers
  auto_reconnect: true    # Automatically reconnect on failure

# Storage configuration
storage:
  history_size: 1000      # Maximum conversation history
  cache_ttl: "1h"         # Tool cache time-to-live

# Logging configuration
logging:
  level: "info"           # "debug", "info", "warn", "error"
  file: "~/.othello/logs/othello.log"
```

### Environment Variables

Override configuration with environment variables:

```bash
export OTHELLO_MODEL_NAME="qwen2.5:3b"
export OTHELLO_MODEL_TEMPERATURE="0.5"
export OTHELLO_OLLAMA_URL="http://192.168.1.100:11434"
export OTHELLO_LOGGING_LEVEL="debug"
```

### CLI Configuration

```bash
# View current configuration
othello config show

# Set configuration values
othello config set model.name qwen2.5:7b
othello config set model.temperature 0.8
othello config set tui.theme dark

# Reset to defaults
othello config reset

# Validate configuration
othello config validate
```

---

## Advanced Features

### Tool Integration

Othello automatically discovers and uses tools from connected MCP servers:

```
You: Can you list the files in my Documents folder?

Othello: I'll check your Documents folder for you.

Using tool: filesystem_list_directory
   Parameters: {"path": "/Users/you/Documents"}

I found several files in your Documents folder:
- project-notes.md (Modified: 2025-10-09)
- budget-2025.xlsx (Modified: 2025-10-08)
- vacation-photos/ (Directory)
- work-presentation.pptx (Modified: 2025-10-07)

Would you like me to show details for any specific file?
```

### Multi-Step Operations

The agent can perform complex multi-step tasks:

```
You: Create a summary of my project files and save it to a new file

Othello: I'll help you create a summary of your project files.

Using tool: filesystem_list_directory
   Parameters: {"path": "./projects"}

Using tool: filesystem_read_file
   Parameters: {"path": "./projects/README.md"}

Using tool: filesystem_read_file
   Parameters: {"path": "./projects/TODO.md"}

Using tool: filesystem_write_file
   Parameters: {
     "path": "./project-summary.md",
     "content": "# Project Summary\n\n..."
   }

I've created a comprehensive summary of your project files and saved it to `project-summary.md`. The summary includes:

1. Overview from README.md
2. Current tasks from TODO.md
3. File structure analysis
4. Recent modifications

The summary file has been saved in your current directory.
```

### Conversation History

```bash
# Search conversation history
You: /history search "database setup"

# Export conversation
You: /export conversation.json

# Clear current conversation
You: /clear

# Load previous conversation
You: /load conversation-id
```

### Server Status Monitoring

```
┌─ Server Status ────────────────────────────────────────────┐
│                                                            │
│ [Success] filesystem      (5 tools)     [Connected]              │
│ [Success] websearch       (3 tools)     [Connected]              │
│ [Warning]  database       (8 tools)     [Reconnecting...]        │
│ [Failed] weather         (2 tools)     [Failed: timeout]        │
│                                                            │
│ Total: 4 servers, 16 tools available                      │
└────────────────────────────────────────────────────────────┘
```

---

## Troubleshooting

### Common Issues

#### "Failed to connect to Ollama"
```bash
# Check if Ollama is running
ollama list

# Start Ollama if not running
ollama serve

# Check configuration
othello config show | grep ollama
```

#### "MCP server failed to start"
```bash
# Test server manually
othello mcp test server-name

# Check server logs
othello mcp logs server-name

# Validate server configuration
othello mcp info server-name
```

#### "Model not found"
```bash
# List available models
ollama list

# Download missing model
ollama pull qwen2.5:3b

# Update configuration
othello config set model.name qwen2.5:3b
```

#### High Memory Usage
```bash
# Use smaller model
othello config set model.name qwen2.5:3b

# Reduce context length
othello config set model.context_length 4096

# Clear conversation history
othello --clear-history
```

### Debug Mode

```bash
# Start with debug logging
othello --log-level debug

# Enable MCP protocol tracing
othello --trace-mcp

# Save debug output
othello --log-file debug.log
```

### Health Check

```bash
# Check agent health
othello health

# Check specific components
othello health --component model
othello health --component mcp
othello health --component storage
```

---

## Examples

### Example 1: File Management

```
You: Help me organize my Downloads folder

Othello: I'll help you organize your Downloads folder. Let me first see what's there.

Using tool: filesystem_list_directory
   Parameters: {"path": "/Users/you/Downloads"}

I found 47 files in your Downloads folder. Here's what I can help organize:

**Documents (12 files):**
- 5 PDF files
- 4 Word documents  
- 3 text files

**Images (23 files):**
- 15 screenshots
- 8 photos

**Software (8 files):**
- 4 installer files
- 4 zip archives

**Other (4 files):**
- Various file types

Would you like me to:
1. Create organized folders and move files
2. Delete old files (older than 30 days)
3. Show detailed analysis first

You: Please create organized folders and move the files

Othello: I'll create organized folders and move your files.

Using tool: filesystem_create_directory
   Parameters: {"path": "/Users/you/Downloads/Documents"}

Using tool: filesystem_create_directory
   Parameters: {"path": "/Users/you/Downloads/Images"}

Using tool: filesystem_create_directory
   Parameters: {"path": "/Users/you/Downloads/Software"}

Using tool: filesystem_move_file
   Parameters: {"source": "/Users/you/Downloads/report.pdf", "destination": "/Users/you/Downloads/Documents/report.pdf"}

[... continuing with all files ...]

[Success] Organization complete! I've:
- Created 4 organizational folders
- Moved 47 files to appropriate categories
- Left 3 recent files in the main Downloads folder

Your Downloads folder is now organized and much cleaner!
```

### Example 2: Research and Analysis

```
You: Research the latest developments in AI and create a summary report

Othello: I'll research the latest AI developments and create a comprehensive summary report for you.

Using tool: web_search
   Parameters: {"query": "latest AI developments 2025", "num_results": 10}

Using tool: web_search
   Parameters: {"query": "artificial intelligence breakthroughs October 2025", "num_results": 10}

Using tool: web_search
   Parameters: {"query": "machine learning research papers 2025", "num_results": 5}

Based on my research, I'll now create a comprehensive summary report.

Using tool: filesystem_write_file
   Parameters: {
     "path": "./ai-developments-report-2025.md",
     "content": "# AI Developments Report - October 2025\n\n## Executive Summary\n\n..."
   }

[Success] Research complete! I've created a comprehensive AI developments report covering:

**Key Topics Covered:**
1. **Breakthrough Models**: Latest large language models and their capabilities
2. **Industry Applications**: Real-world AI implementations
3. **Research Advances**: Recent academic breakthroughs
4. **Ethical Considerations**: Current AI safety and ethics discussions
5. **Future Trends**: Predictions for the next 6 months

**Report Details:**
- **File**: `ai-developments-report-2025.md`
- **Length**: 2,847 words
- **Sources**: 25 recent articles and papers
- **Last Updated**: October 10, 2025

The report is saved in your current directory and ready for review!
```

### Example 3: Development Workflow

```
You: Help me set up a new Go project with proper structure and documentation

Othello: I'll help you create a well-structured Go project with proper documentation and best practices.

Using tool: filesystem_create_directory
   Parameters: {"path": "./my-go-project"}

Using tool: filesystem_create_directory
   Parameters: {"path": "./my-go-project/cmd"}

Using tool: filesystem_create_directory
   Parameters: {"path": "./my-go-project/internal"}

Using tool: filesystem_create_directory
   Parameters: {"path": "./my-go-project/pkg"}

Using tool: filesystem_create_directory
   Parameters: {"path": "./my-go-project/docs"}

Using tool: filesystem_write_file
   Parameters: {
     "path": "./my-go-project/go.mod",
     "content": "module my-go-project\n\ngo 1.21\n"
   }

Using tool: filesystem_write_file
   Parameters: {
     "path": "./my-go-project/README.md",
     "content": "# My Go Project\n\n## Description\n\n..."
   }

Using tool: filesystem_write_file
   Parameters: {
     "path": "./my-go-project/Makefile",
     "content": ".PHONY: build test clean\n\nbuild:\n\tgo build -o bin/ ./cmd/...\n\n..."
   }

[Success] Project setup complete! I've created:

**Directory Structure:**
```
my-go-project/
├── cmd/                    # Application entrypoints
├── internal/               # Private application code
├── pkg/                    # Public library code
├── docs/                   # Documentation
├── go.mod                  # Go module file
├── README.md               # Project documentation
├── Makefile               # Build automation
└── .gitignore             # Git ignore rules
```

**What's included:**
- [Success] Standard Go project layout
- [Success] Module initialization
- [Success] Comprehensive README
- [Success] Makefile with common tasks
- [Success] Git ignore file
- [Success] Basic documentation structure

Your project is ready for development! Would you like me to add any specific components like CI/CD configuration, Docker setup, or testing frameworks?
```

---

## Getting Help

### Built-in Help

```bash
# General help
othello --help

# Command-specific help
othello mcp --help
othello config --help

# In TUI mode
Press Ctrl+H for help
```

### Community Support

- **GitHub Issues**: Report bugs and request features
- **Discussions**: Ask questions and share experiences  
- **Discord**: Real-time community chat
- **Documentation**: Comprehensive guides and tutorials

### Reporting Issues

When reporting issues, include:

1. **Version**: `othello --version`
2. **OS and Architecture**: `uname -a`
3. **Configuration**: `othello config show`
4. **Logs**: Debug output with `--log-level debug`
5. **Steps to Reproduce**: Clear reproduction steps

---

## FAQ

**Q: How much memory does Othello use?**
A: Base agent uses ~50-100MB. With Qwen2.5:3b model, expect 2-4GB total. Qwen2.5:7b requires 6-8GB.

**Q: Can I use Othello without internet?**
A: Yes! Core functionality works completely offline with local models.

**Q: How do I create custom MCP servers?**
A: Follow the MCP specification and use the official SDKs. Examples available in the documentation.

**Q: Can I use other models besides Qwen2.5?**
A: Yes! Any GGUF model supported by Ollama should work. Adjust configuration accordingly.

**Q: Is my data shared with external services?**
A: No. Othello runs completely locally unless you explicitly connect to remote MCP servers.

**Q: How do I update Othello?**
A: Download the latest binary or use your package manager to update to the newest version.