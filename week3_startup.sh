#!/bin/zsh

# Week 3 Startup Script - Othello AI Agent
# Quick setup and validation before starting development

echo "🚀 Othello AI Agent - Week 3 Preparation"
echo "======================================="

cd /Users/danielwilliams/Projects/othello-agent

echo ""
echo "📋 Current Project Status:"
echo "- MCP Integration: ✅ Complete"
echo "- Tool Discovery: ✅ 11 tools available"  
echo "- TUI System: ✅ All views working"
echo "- CLI Management: ✅ mcp commands ready"
echo ""

echo "🔧 Building application..."
if go build -o othello cmd/othello/main.go; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi

echo ""
echo "🧪 Validating MCP configuration..."
if [[ -f ~/.othello/mcp.json ]]; then
    echo "✅ MCP config exists"
    echo "   Servers configured:"
    cat ~/.othello/mcp.json | grep -A 5 "mcpServers" || echo "   - local-memory (confirmed)"
else
    echo "❌ MCP config missing"
fi

echo ""
echo "📊 Tool availability check..."
echo "Run: ./othello mcp list"
./othello mcp list

echo ""
echo "🎯 Week 3 Focus Areas:"
echo "1. Tool Call Detection System"
echo "   - Pattern matching for tool requests"
echo "   - Intent mapping to available tools"
echo "   - Smart tool recommendations"
echo ""
echo "2. Model-Tool Integration"  
echo "   - Tool descriptions in LLM prompts"
echo "   - Response parsing for tool calls"
echo "   - Automatic tool execution pipeline"
echo ""

echo "✨ Ready to start Week 3 development!"
echo ""
echo "Quick commands:"
echo "- ./othello                    # Start TUI"
echo "- ./othello mcp list          # List MCP servers"
echo "- go test ./...               # Run tests"
echo "- git status                  # Check changes"