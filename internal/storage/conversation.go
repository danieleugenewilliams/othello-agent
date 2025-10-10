package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Message represents a conversation message
type Message struct {
	ID            int64     `json:"id" db:"id"`
	ConversationID string   `json:"conversation_id" db:"conversation_id"`
	Role          string    `json:"role" db:"role"` // "user", "assistant", "tool"
	Content       string    `json:"content" db:"content"`
	ToolCall      *ToolCall `json:"tool_call,omitempty" db:"tool_call"`
	ToolResult    *ToolResult `json:"tool_result,omitempty" db:"tool_result"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
	TokenCount    int       `json:"token_count" db:"token_count"`
}

// ToolCall represents a tool call request
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents a tool call result
type ToolResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// Conversation represents a conversation thread
type Conversation struct {
	ID          string    `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	MessageCount int      `json:"message_count" db:"message_count"`
	TotalTokens  int      `json:"total_tokens" db:"total_tokens"`
}

// ConversationStore manages conversation storage
type ConversationStore struct {
	db *sql.DB
}

// NewConversationStore creates a new conversation store
func NewConversationStore(dbPath string) (*ConversationStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	
	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	
	store := &ConversationStore{db: db}
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("initialize schema: %w", err)
	}
	
	return store, nil
}

// initSchema creates the database tables
func (s *ConversationStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		message_count INTEGER NOT NULL DEFAULT 0,
		total_tokens INTEGER NOT NULL DEFAULT 0
	);
	
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id TEXT NOT NULL,
		role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
		content TEXT NOT NULL,
		tool_call TEXT, -- JSON blob for tool calls
		tool_result TEXT, -- JSON blob for tool results
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		token_count INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
	);
	
	CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
	CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at);
	`
	
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	
	return nil
}

// CreateConversation creates a new conversation
func (s *ConversationStore) CreateConversation(id, title string) (*Conversation, error) {
	now := time.Now()
	conv := &Conversation{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	query := `
		INSERT INTO conversations (id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`
	
	if _, err := s.db.Exec(query, conv.ID, conv.Title, conv.CreatedAt, conv.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert conversation: %w", err)
	}
	
	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *ConversationStore) GetConversation(id string) (*Conversation, error) {
	query := `
		SELECT id, title, created_at, updated_at, message_count, total_tokens
		FROM conversations
		WHERE id = ?
	`
	
	var conv Conversation
	if err := s.db.QueryRow(query, id).Scan(
		&conv.ID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt,
		&conv.MessageCount, &conv.TotalTokens,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query conversation: %w", err)
	}
	
	return &conv, nil
}

// ListConversations returns all conversations ordered by updated time
func (s *ConversationStore) ListConversations(limit, offset int) ([]*Conversation, error) {
	query := `
		SELECT id, title, created_at, updated_at, message_count, total_tokens
		FROM conversations
		ORDER BY updated_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query conversations: %w", err)
	}
	defer rows.Close()
	
	var conversations []*Conversation
	for rows.Next() {
		var conv Conversation
		if err := rows.Scan(
			&conv.ID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt,
			&conv.MessageCount, &conv.TotalTokens,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		conversations = append(conversations, &conv)
	}
	
	return conversations, nil
}

// AddMessage adds a message to a conversation
func (s *ConversationStore) AddMessage(msg *Message) error {
	// Serialize tool call and result to JSON
	var toolCallJSON, toolResultJSON sql.NullString
	
	if msg.ToolCall != nil {
		data, err := json.Marshal(msg.ToolCall)
		if err != nil {
			return fmt.Errorf("marshal tool call: %w", err)
		}
		toolCallJSON = sql.NullString{String: string(data), Valid: true}
	}
	
	if msg.ToolResult != nil {
		data, err := json.Marshal(msg.ToolResult)
		if err != nil {
			return fmt.Errorf("marshal tool result: %w", err)
		}
		toolResultJSON = sql.NullString{String: string(data), Valid: true}
	}
	
	// Insert message
	query := `
		INSERT INTO messages (conversation_id, role, content, tool_call, tool_result, timestamp, token_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	result, err := s.db.Exec(query,
		msg.ConversationID, msg.Role, msg.Content,
		toolCallJSON, toolResultJSON, msg.Timestamp, msg.TokenCount,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	
	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	msg.ID = id
	
	// Update conversation stats
	if err := s.updateConversationStats(msg.ConversationID); err != nil {
		return fmt.Errorf("update conversation stats: %w", err)
	}
	
	return nil
}

// GetMessages retrieves messages for a conversation
func (s *ConversationStore) GetMessages(conversationID string, limit, offset int) ([]*Message, error) {
	query := `
		SELECT id, conversation_id, role, content, tool_call, tool_result, timestamp, token_count
		FROM messages
		WHERE conversation_id = ?
		ORDER BY timestamp ASC
		LIMIT ? OFFSET ?
	`
	
	rows, err := s.db.Query(query, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()
	
	var messages []*Message
	for rows.Next() {
		var msg Message
		var toolCallJSON, toolResultJSON sql.NullString
		
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content,
			&toolCallJSON, &toolResultJSON, &msg.Timestamp, &msg.TokenCount,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		
		// Deserialize tool call and result
		if toolCallJSON.Valid {
			var toolCall ToolCall
			if err := json.Unmarshal([]byte(toolCallJSON.String), &toolCall); err != nil {
				return nil, fmt.Errorf("unmarshal tool call: %w", err)
			}
			msg.ToolCall = &toolCall
		}
		
		if toolResultJSON.Valid {
			var toolResult ToolResult
			if err := json.Unmarshal([]byte(toolResultJSON.String), &toolResult); err != nil {
				return nil, fmt.Errorf("unmarshal tool result: %w", err)
			}
			msg.ToolResult = &toolResult
		}
		
		messages = append(messages, &msg)
	}
	
	return messages, nil
}

// DeleteConversation deletes a conversation and all its messages
func (s *ConversationStore) DeleteConversation(id string) error {
	query := "DELETE FROM conversations WHERE id = ?"
	if _, err := s.db.Exec(query, id); err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	return nil
}

// UpdateConversationTitle updates the title of a conversation
func (s *ConversationStore) UpdateConversationTitle(id, title string) error {
	query := "UPDATE conversations SET title = ?, updated_at = ? WHERE id = ?"
	if _, err := s.db.Exec(query, title, time.Now(), id); err != nil {
		return fmt.Errorf("update conversation title: %w", err)
	}
	return nil
}

// SearchMessages searches for messages containing the given text
func (s *ConversationStore) SearchMessages(query string, limit int) ([]*Message, error) {
	sqlQuery := `
		SELECT id, conversation_id, role, content, tool_call, tool_result, timestamp, token_count
		FROM messages
		WHERE content LIKE ?
		ORDER BY timestamp DESC
		LIMIT ?
	`
	
	rows, err := s.db.Query(sqlQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()
	
	var messages []*Message
	for rows.Next() {
		var msg Message
		var toolCallJSON, toolResultJSON sql.NullString
		
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content,
			&toolCallJSON, &toolResultJSON, &msg.Timestamp, &msg.TokenCount,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		
		// Deserialize tool call and result
		if toolCallJSON.Valid {
			var toolCall ToolCall
			if err := json.Unmarshal([]byte(toolCallJSON.String), &toolCall); err != nil {
				return nil, fmt.Errorf("unmarshal tool call: %w", err)
			}
			msg.ToolCall = &toolCall
		}
		
		if toolResultJSON.Valid {
			var toolResult ToolResult
			if err := json.Unmarshal([]byte(toolResultJSON.String), &toolResult); err != nil {
				return nil, fmt.Errorf("unmarshal tool result: %w", err)
			}
			msg.ToolResult = &toolResult
		}
		
		messages = append(messages, &msg)
	}
	
	return messages, nil
}

// updateConversationStats updates message count and token count for a conversation
func (s *ConversationStore) updateConversationStats(conversationID string) error {
	query := `
		UPDATE conversations
		SET message_count = (
			SELECT COUNT(*) FROM messages WHERE conversation_id = ?
		),
		total_tokens = (
			SELECT COALESCE(SUM(token_count), 0) FROM messages WHERE conversation_id = ?
		),
		updated_at = ?
		WHERE id = ?
	`
	
	_, err := s.db.Exec(query, conversationID, conversationID, time.Now(), conversationID)
	if err != nil {
		return fmt.Errorf("update conversation stats: %w", err)
	}
	
	return nil
}

// UpdateConversationStats is a public wrapper for updateConversationStats
func (s *ConversationStore) UpdateConversationStats(conversationID string) error {
	return s.updateConversationStats(conversationID)
}

// Close closes the database connection
func (s *ConversationStore) Close() error {
	return s.db.Close()
}