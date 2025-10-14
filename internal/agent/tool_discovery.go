package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/danieleugenewilliams/othello-agent/internal/mcp"
)

// ToolCapability represents different categories of tool functionality
type ToolCapability int

const (
	CapabilitySearch ToolCapability = iota
	CapabilityCreate
	CapabilityUpdate
	CapabilityDelete
	CapabilityAnalyze
	CapabilityTransform
	CapabilityConnect
	CapabilityUnknown
)

// ToolMetadata contains enhanced information about a tool
type ToolMetadata struct {
	Tool         mcp.Tool
	Capability   ToolCapability
	Complexity   int    // 1-5 scale of parameter complexity
	UsagePattern string // Common usage patterns
	Keywords     []string
}

// ToolDiscovery manages dynamic tool discovery and categorization
type ToolDiscovery struct {
	registry *mcp.ToolRegistry
	cache    map[string][]ToolMetadata
	logger   mcp.Logger
}

// NewToolDiscovery creates a new tool discovery manager
func NewToolDiscovery(registry *mcp.ToolRegistry, logger mcp.Logger) *ToolDiscovery {
	return &ToolDiscovery{
		registry: registry,
		cache:    make(map[string][]ToolMetadata),
		logger:   logger,
	}
}

// DiscoverAllTools discovers and categorizes tools from all registered servers
func (td *ToolDiscovery) DiscoverAllTools(ctx context.Context) ([]ToolMetadata, error) {
	// Check cache first
	cacheKey := "all_tools"
	if cached, exists := td.cache[cacheKey]; exists {
		return cached, nil
	}

	// Get all tools from registry
	tools := td.registry.ListTools()
	metadata := make([]ToolMetadata, len(tools))

	for i, tool := range tools {
		metadata[i] = td.analyzeToolMetadata(tool)
	}

	// Sort by capability and complexity for better prompt organization
	sort.Slice(metadata, func(i, j int) bool {
		if metadata[i].Capability != metadata[j].Capability {
			return metadata[i].Capability < metadata[j].Capability
		}
		return metadata[i].Complexity < metadata[j].Complexity
	})

	// Cache the results
	td.cache[cacheKey] = metadata
	td.logger.Info("Discovered and categorized %d tools from %d servers",
		len(metadata), td.registry.GetServerCount())

	return metadata, nil
}

// DiscoverToolsForServer discovers tools from a specific server
func (td *ToolDiscovery) DiscoverToolsForServer(ctx context.Context, serverName string) ([]ToolMetadata, error) {
	tools := td.registry.ListToolsForServer(serverName)
	metadata := make([]ToolMetadata, len(tools))

	for i, tool := range tools {
		metadata[i] = td.analyzeToolMetadata(tool)
	}

	return metadata, nil
}

// GetToolsByCapability returns tools filtered by capability
func (td *ToolDiscovery) GetToolsByCapability(capability ToolCapability) ([]ToolMetadata, error) {
	allTools, err := td.DiscoverAllTools(context.Background())
	if err != nil {
		return nil, err
	}

	var filtered []ToolMetadata
	for _, tool := range allTools {
		if tool.Capability == capability {
			filtered = append(filtered, tool)
		}
	}

	return filtered, nil
}

// analyzeToolMetadata analyzes a tool to determine its metadata
func (td *ToolDiscovery) analyzeToolMetadata(tool mcp.Tool) ToolMetadata {
	metadata := ToolMetadata{
		Tool:       tool,
		Capability: td.categorizeToolCapability(tool),
		Complexity: td.calculateComplexity(tool),
		Keywords:   td.extractKeywords(tool),
	}

	metadata.UsagePattern = td.generateUsagePattern(metadata)
	return metadata
}

// categorizeToolCapability determines the primary capability of a tool
func (td *ToolDiscovery) categorizeToolCapability(tool mcp.Tool) ToolCapability {
	name := strings.ToLower(tool.Name)
	description := strings.ToLower(tool.Description)
	combined := name + " " + description

	// Search capabilities
	if strings.Contains(combined, "search") || strings.Contains(combined, "find") ||
	   strings.Contains(combined, "query") || strings.Contains(combined, "list") ||
	   strings.Contains(combined, "get") {
		return CapabilitySearch
	}

	// Create capabilities
	if strings.Contains(combined, "create") || strings.Contains(combined, "add") ||
	   strings.Contains(combined, "store") || strings.Contains(combined, "save") ||
	   strings.Contains(combined, "insert") {
		return CapabilityCreate
	}

	// Update capabilities
	if strings.Contains(combined, "update") || strings.Contains(combined, "edit") ||
	   strings.Contains(combined, "modify") || strings.Contains(combined, "change") {
		return CapabilityUpdate
	}

	// Delete capabilities
	if strings.Contains(combined, "delete") || strings.Contains(combined, "remove") ||
	   strings.Contains(combined, "clear") {
		return CapabilityDelete
	}

	// Analysis capabilities
	if strings.Contains(combined, "analyze") || strings.Contains(combined, "stats") ||
	   strings.Contains(combined, "summary") || strings.Contains(combined, "report") {
		return CapabilityAnalyze
	}

	// Transform capabilities
	if strings.Contains(combined, "transform") || strings.Contains(combined, "convert") ||
	   strings.Contains(combined, "format") || strings.Contains(combined, "process") {
		return CapabilityTransform
	}

	// Connection capabilities
	if strings.Contains(combined, "connect") || strings.Contains(combined, "relate") ||
	   strings.Contains(combined, "link") || strings.Contains(combined, "relationship") {
		return CapabilityConnect
	}

	return CapabilityUnknown
}

// calculateComplexity estimates the complexity of using a tool (1-5 scale)
func (td *ToolDiscovery) calculateComplexity(tool mcp.Tool) int {
	if tool.InputSchema == nil {
		return 1 // No parameters = simple
	}

	properties, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		return 1
	}

	complexity := 1
	paramCount := len(properties)

	// Base complexity on parameter count
	if paramCount > 5 {
		complexity = 5
	} else if paramCount > 3 {
		complexity = 4
	} else if paramCount > 1 {
		complexity = 3
	} else if paramCount == 1 {
		complexity = 2
	}

	// Adjust for required parameters
	if required, ok := tool.InputSchema["required"].([]interface{}); ok {
		if len(required) > 3 {
			complexity++
		}
	}

	// Adjust for complex parameter types
	for _, param := range properties {
		if paramMap, ok := param.(map[string]interface{}); ok {
			if paramType, ok := paramMap["type"].(string); ok {
				if paramType == "object" || paramType == "array" {
					complexity++
				}
			}
		}
	}

	// Cap at 5
	if complexity > 5 {
		complexity = 5
	}

	return complexity
}

// extractKeywords extracts relevant keywords from tool name and description
func (td *ToolDiscovery) extractKeywords(tool mcp.Tool) []string {
	keywords := []string{}

	// Split name and description into words
	words := strings.Fields(strings.ToLower(tool.Name + " " + tool.Description))

	// Filter for meaningful keywords (longer than 2 chars, not common words)
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "from": true,
		"this": true, "that": true, "will": true, "can": true, "are": true,
		"tool": true, "function": true, "method": true,
	}

	for _, word := range words {
		// Clean word
		word = strings.Trim(word, ".,!?();:")
		if len(word) > 2 && !commonWords[word] {
			keywords = append(keywords, word)
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := []string{}
	for _, keyword := range keywords {
		if !seen[keyword] {
			seen[keyword] = true
			unique = append(unique, keyword)
		}
	}

	return unique
}

// generateUsagePattern creates a usage pattern description for the tool
func (td *ToolDiscovery) generateUsagePattern(metadata ToolMetadata) string {
	capability := metadata.Capability
	toolName := metadata.Tool.Name

	switch capability {
	case CapabilitySearch:
		return fmt.Sprintf("Use %s when you need to find or retrieve information", toolName)
	case CapabilityCreate:
		return fmt.Sprintf("Use %s when you need to create or store new data", toolName)
	case CapabilityUpdate:
		return fmt.Sprintf("Use %s when you need to modify existing data", toolName)
	case CapabilityDelete:
		return fmt.Sprintf("Use %s when you need to remove data", toolName)
	case CapabilityAnalyze:
		return fmt.Sprintf("Use %s when you need insights or analysis", toolName)
	case CapabilityTransform:
		return fmt.Sprintf("Use %s when you need to transform or process data", toolName)
	case CapabilityConnect:
		return fmt.Sprintf("Use %s when you need to create relationships or connections", toolName)
	default:
		return fmt.Sprintf("Use %s for specialized operations", toolName)
	}
}

// GetCapabilityName returns a human-readable name for a capability
func GetCapabilityName(capability ToolCapability) string {
	switch capability {
	case CapabilitySearch:
		return "Search & Retrieval"
	case CapabilityCreate:
		return "Creation & Storage"
	case CapabilityUpdate:
		return "Updates & Modifications"
	case CapabilityDelete:
		return "Deletion & Cleanup"
	case CapabilityAnalyze:
		return "Analysis & Insights"
	case CapabilityTransform:
		return "Transformation & Processing"
	case CapabilityConnect:
		return "Relationships & Connections"
	default:
		return "Specialized Operations"
	}
}

// InvalidateCache clears the tool discovery cache
func (td *ToolDiscovery) InvalidateCache() {
	td.cache = make(map[string][]ToolMetadata)
	td.logger.Info("Tool discovery cache invalidated")
}