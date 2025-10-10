package main

import (
	"fmt"
	"os"

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

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
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