package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SearchFilter represents search and filter criteria
type SearchFilter struct {
	Query           string     `json:"query"`
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	MessageType     string     `json:"message_type"`     // "user", "assistant", "tool"
	ConversationID  string     `json:"conversation_id"`
	Limit           int        `json:"limit"`
	Offset          int        `json:"offset"`
}

// SearchStatistics provides search performance and cache metrics
type SearchStatistics struct {
	TotalQueries     int           `json:"total_queries"`
	CacheHits        int           `json:"cache_hits"`
	CacheMisses      int           `json:"cache_misses"`
	AverageQueryTime time.Duration `json:"average_query_time"`
	LastUpdated      time.Time     `json:"last_updated"`
}

// SearchManager handles conversation and message search operations
type SearchManager struct {
	store      ConversationStore
	db         *sql.DB
	statistics SearchStatistics
}

// NewSearchManager creates a new search manager
func NewSearchManager(store ConversationStore, db *sql.DB) *SearchManager {
	return &SearchManager{
		store: store,
		db:    db,
		statistics: SearchStatistics{
			LastUpdated: time.Now(),
		},
	}
}

// SearchMessages performs full-text search on message content with filtering
func (sm *SearchManager) SearchMessages(filter SearchFilter) ([]*Message, error) {
	start := time.Now()
	defer func() {
		sm.updateQueryStats(time.Since(start))
	}()

	// Build the SQL query
	query := `
		SELECT m.id, m.conversation_id, m.role, m.content, m.timestamp
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIndex := 1

	// Add search conditions
	if filter.Query != "" {
		query += fmt.Sprintf(" AND LOWER(m.content) LIKE LOWER($%d)", argIndex)
		args = append(args, "%"+filter.Query+"%")
		argIndex++
	}

	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND m.timestamp >= $%d", argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND m.timestamp <= $%d", argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}

	if filter.MessageType != "" {
		query += fmt.Sprintf(" AND m.role = $%d", argIndex)
		args = append(args, filter.MessageType)
		argIndex++
	}

	if filter.ConversationID != "" {
		query += fmt.Sprintf(" AND m.conversation_id = $%d", argIndex)
		args = append(args, filter.ConversationID)
		argIndex++
	}

	// Add ordering and pagination
	query += " ORDER BY m.timestamp DESC"
	
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	// Execute query
	rows, err := sm.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{}
		err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.Role,
			&message.Content,
			&message.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over messages: %w", err)
	}

	return messages, nil
}

// SearchConversations searches conversation titles and returns matching conversations
func (sm *SearchManager) SearchConversations(query string, limit int) ([]*Conversation, error) {
	start := time.Now()
	defer func() {
		sm.updateQueryStats(time.Since(start))
	}()

	if query == "" {
		return sm.getAllConversations(limit)
	}

	sqlQuery := `
		SELECT id, title, created_at, updated_at
		FROM conversations
		WHERE LOWER(title) LIKE LOWER($1)
		ORDER BY updated_at DESC
	`
	args := []interface{}{"%" + query + "%"}

	if limit > 0 {
		sqlQuery += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := sm.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*Conversation
	for rows.Next() {
		conv := &Conversation{}
		err := rows.Scan(&conv.ID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// getAllConversations returns all conversations when no search query is provided
func (sm *SearchManager) getAllConversations(limit int) ([]*Conversation, error) {
	sqlQuery := `
		SELECT id, title, created_at, updated_at
		FROM conversations
		ORDER BY updated_at DESC
	`
	args := make([]interface{}, 0)

	if limit > 0 {
		sqlQuery += " LIMIT $1"
		args = append(args, limit)
	}

	rows, err := sm.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get all conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*Conversation
	for rows.Next() {
		conv := &Conversation{}
		err := rows.Scan(&conv.ID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// FilterByDateRange filters messages within a date range
func (sm *SearchManager) FilterByDateRange(startDate, endDate *time.Time) ([]*Message, error) {
	filter := SearchFilter{
		StartDate: startDate,
		EndDate:   endDate,
	}
	return sm.SearchMessages(filter)
}

// FilterByMessageType filters messages by role (user, assistant, tool)
func (sm *SearchManager) FilterByMessageType(messageType string) ([]*Message, error) {
	filter := SearchFilter{
		MessageType: messageType,
	}
	return sm.SearchMessages(filter)
}

// FilterByConversation filters messages within a specific conversation
func (sm *SearchManager) FilterByConversation(conversationID string) ([]*Message, error) {
	filter := SearchFilter{
		ConversationID: conversationID,
	}
	return sm.SearchMessages(filter)
}

// GetSearchStatistics returns current search statistics
func (sm *SearchManager) GetSearchStatistics() SearchStatistics {
	sm.statistics.LastUpdated = time.Now()
	return sm.statistics
}

// updateQueryStats updates search statistics after each query
func (sm *SearchManager) updateQueryStats(duration time.Duration) {
	sm.statistics.TotalQueries++
	
	// Calculate running average
	if sm.statistics.TotalQueries == 1 {
		sm.statistics.AverageQueryTime = duration
	} else {
		// Weighted average: (old_avg * (n-1) + new_time) / n
		oldTotal := sm.statistics.AverageQueryTime * time.Duration(sm.statistics.TotalQueries-1)
		sm.statistics.AverageQueryTime = (oldTotal + duration) / time.Duration(sm.statistics.TotalQueries)
	}
}

// Helper functions for case-insensitive search operations

// containsIgnoreCase checks if the content contains the query (case-insensitive)
func containsIgnoreCase(content, query string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(query))
}

// findIgnoreCase finds the first occurrence of query in content (case-insensitive)
func findIgnoreCase(content, query string) int {
	return strings.Index(strings.ToLower(content), strings.ToLower(query))
}

// toLower converts string to lowercase
func toLower(s string) string {
	return strings.ToLower(s)
}