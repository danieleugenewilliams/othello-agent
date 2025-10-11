package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danieleugenewilliams/othello-agent/internal/agent"
	"github.com/danieleugenewilliams/othello-agent/internal/config"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "othello",
	Short: "Othello AI Agent - Local AI assistant with MCP tool integration",
	Long: `Othello is a high-performance AI agent built in Go that integrates with
local language models through Ollama and provides tool capabilities
via Model Context Protocol (MCP) servers.

Features:
- Local AI model execution via Ollama
- MCP tool server integration
- Terminal user interface
- Conversation history
- Configuration management`,
	RunE: runInteractive,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Othello AI Agent\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		fmt.Printf("Configuration loaded from: %s\n", cfg.ConfigFile())
		fmt.Printf("\nModel Configuration:\n")
		fmt.Printf("  Type: %s\n", cfg.Model.Type)
		fmt.Printf("  Name: %s\n", cfg.Model.Name)
		fmt.Printf("  Temperature: %.2f\n", cfg.Model.Temperature)
		fmt.Printf("  Max Tokens: %d\n", cfg.Model.MaxTokens)

		fmt.Printf("\nOllama Configuration:\n")
		fmt.Printf("  Host: %s\n", cfg.Ollama.Host)
		fmt.Printf("  Timeout: %s\n", cfg.Ollama.Timeout)

		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.CreateDefaultConfig()
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server management commands",
	Long:  "Manage Model Context Protocol (MCP) servers - add, remove, list, and view server configurations",
}

var mcpAddCmd = &cobra.Command{
	Use:   "add <name> <command> [args...]",
	Short: "Add a new MCP server",
	Long: `Add a new MCP server to the configuration.

Examples:
  # Add filesystem server
  othello mcp add filesystem npx @modelcontextprotocol/server-filesystem /tmp

  # Add local-memory server  
  othello mcp add memory npx @danieleugenewilliams/local-memory-server

  # Add custom server with environment variables
  othello mcp add custom /usr/bin/python3 -m myserver --port 8080`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		name := args[0]
		command := args[1]
		serverArgs := args[2:]

		// Get flags
		transport, _ := cmd.Flags().GetString("transport")
		timeout, _ := cmd.Flags().GetString("timeout")
		envVars, _ := cmd.Flags().GetStringToString("env")

		// Parse timeout
		var timeoutDuration time.Duration
		if timeout != "" {
			timeoutDuration, err = time.ParseDuration(timeout)
			if err != nil {
				return fmt.Errorf("invalid timeout format: %w", err)
			}
		}

		server := config.ServerConfig{
			Name:      name,
			Command:   command,
			Args:      serverArgs,
			Env:       envVars,
			Transport: transport,
			Timeout:   timeoutDuration,
		}

		if err := cfg.AddMCPServer(server); err != nil {
			return fmt.Errorf("failed to add MCP server: %w", err)
		}

		fmt.Printf("✅ Successfully added MCP server '%s'\n", name)
		fmt.Printf("   Command: %s %s\n", command, strings.Join(serverArgs, " "))
		fmt.Printf("   Transport: %s\n", transport)
		if timeout != "" {
			fmt.Printf("   Timeout: %s\n", timeout)
		}
		
		return nil
	},
}

var mcpRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an MCP server",
	Long:  "Remove an MCP server from the configuration by name.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		name := args[0]

		if err := cfg.RemoveMCPServer(name); err != nil {
			return fmt.Errorf("failed to remove MCP server: %w", err)
		}

		fmt.Printf("✅ Successfully removed MCP server '%s'\n", name)
		return nil
	},
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured MCP servers",
	Long:  "Display all configured MCP servers with their details.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		servers := cfg.ListMCPServers()
		
		if len(servers) == 0 {
			fmt.Println("No MCP servers configured.")
			fmt.Println("\nTo add a server, use:")
			fmt.Println("  othello mcp add <name> <command> [args...]")
			return nil
		}

		fmt.Printf("Configured MCP Servers (%d):\n\n", len(servers))
		
		for i, server := range servers {
			fmt.Printf("%d. %s\n", i+1, server.Name)
			fmt.Printf("   Command: %s", server.Command)
			if len(server.Args) > 0 {
				fmt.Printf(" %s", strings.Join(server.Args, " "))
			}
			fmt.Printf("\n")
			fmt.Printf("   Transport: %s\n", server.Transport)
			if server.Timeout > 0 {
				fmt.Printf("   Timeout: %s\n", server.Timeout)
			}
			if len(server.Env) > 0 {
				fmt.Printf("   Environment:\n")
				for k, v := range server.Env {
					fmt.Printf("     %s=%s\n", k, v)
				}
			}
			if i < len(servers)-1 {
				fmt.Println()
			}
		}
		
		return nil
	},
}

var mcpShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a specific MCP server",
	Long:  "Display detailed information about a specific MCP server configuration.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		name := args[0]
		server, err := cfg.GetMCPServer(name)
		if err != nil {
			return err
		}

		fmt.Printf("MCP Server: %s\n\n", server.Name)
		fmt.Printf("Command: %s", server.Command)
		if len(server.Args) > 0 {
			fmt.Printf(" %s", strings.Join(server.Args, " "))
		}
		fmt.Printf("\n")
		fmt.Printf("Transport: %s\n", server.Transport)
		
		if server.Timeout > 0 {
			fmt.Printf("Timeout: %s\n", server.Timeout)
		}
		
		if len(server.Env) > 0 {
			fmt.Printf("Environment Variables:\n")
			for k, v := range server.Env {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	
	// Add MCP command and subcommands
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpAddCmd)
	mcpCmd.AddCommand(mcpRemoveCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpShowCmd)
	
	// Add flags for mcp add command
	mcpAddCmd.Flags().StringP("transport", "t", "stdio", "Transport type (stdio or http)")
	mcpAddCmd.Flags().String("timeout", "", "Timeout duration (e.g., 30s, 1m)")
	mcpAddCmd.Flags().StringToStringP("env", "e", nil, "Environment variables (key=value)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runInteractive(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting Othello AI Agent...")
	
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create agent instance
	agentInstance, err := agent.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Start TUI mode
	return agentInstance.StartTUI()
}