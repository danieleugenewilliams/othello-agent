package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSearchTestDB(t *testing.T) (*ConversationStore, *SearchManager) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "search_test.db")
	
	store, err := NewConversationStore(dbPath)
	require.NoError(t, err, "Failed to create conversation store")
	
	searchManager := NewSearchManager(*store, store.db)
	return store, searchManager
}

func TestNewSearchManager(t *testing.T) {
	store, searchManager := setupSearchTestDB(t)
	defer store.Close()

	assert.NotNil(t, searchManager)
	assert.Equal(t, *store, searchManager.store)
}

func TestSearchManager_FullTextSearch(t *testing.T) {
	store, searchManager := setupSearchTestDB(t)
	defer store.Close()

	// Create test conversation
	conversationID := "test-conv-1"
	_, err := store.CreateConversation(conversationID, "Test Conversation")
	require.NoError(t, err)

	// Add test messages
	messages := []*Message{
		{ConversationID: conversationID, Role: "user", Content: "Hello world", Timestamp: time.Now()},
		{ConversationID: conversationID, Role: "assistant", Content: "Hello there! How can I help?", Timestamp: time.Now().Add(time.Minute)},
		{ConversationID: conversationID, Role: "user", Content: "What is machine learning?", Timestamp: time.Now().Add(2 * time.Minute)},
		{ConversationID: conversationID, Role: "assistant", Content: "Machine learning is a subset of AI", Timestamp: time.Now().Add(3 * time.Minute)},
	}

	for _, msg := range messages {
		err := store.AddMessage(msg)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		query          string
		expectedCount  int
		expectedFirst  string
	}{
		{
			name:          "simple search",
			query:         "hello",
			expectedCount: 2,
			expectedFirst: "Hello there! How can I help?", // Most recent first
		},
		{
			name:          "case insensitive search",
			query:         "MACHINE",
			expectedCount: 2,
			expectedFirst: "Machine learning is a subset of AI",
		},
		{
			name:          "no matches",
			query:         "nonexistent",
			expectedCount: 0,
		},
		{
			name:          "partial word",
			query:         "learn",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := SearchFilter{Query: tt.query}
			results, err := searchManager.SearchMessages(filter)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCount, len(results))
			if tt.expectedCount > 0 {
				assert.Contains(t, results[0].Content, tt.expectedFirst)
			}
		})
	}
}

func TestSearchManager_FilterByDateRange(t *testing.T) {
	store, searchManager := setupSearchTestDB(t)
	defer store.Close()

	// Create test conversation
	conversationID := "test-conv-2"
	_, err := store.CreateConversation(conversationID, "Date Test Conversation")
	require.NoError(t, err)

	// Add messages with different timestamps
	now := time.Now()
	messages := []*Message{
		{ConversationID: conversationID, Role: "user", Content: "Message 1", Timestamp: now.Add(-2 * time.Hour)},
		{ConversationID: conversationID, Role: "user", Content: "Message 2", Timestamp: now.Add(-1 * time.Hour)},
		{ConversationID: conversationID, Role: "user", Content: "Message 3", Timestamp: now},
		{ConversationID: conversationID, Role: "user", Content: "Message 4", Timestamp: now.Add(1 * time.Hour)},
	}

	for _, msg := range messages {
		err := store.AddMessage(msg)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		startDate     *time.Time
		endDate       *time.Time
		expectedCount int
	}{
		{
			name:          "last hour",
			startDate:     timePtr(now.Add(-30 * time.Minute)),
			endDate:       timePtr(now.Add(30 * time.Minute)),
			expectedCount: 1,
		},
		{
			name:          "all messages",
			startDate:     timePtr(now.Add(-3 * time.Hour)),
			endDate:       timePtr(now.Add(2 * time.Hour)),
			expectedCount: 4,
		},
		{
			name:          "future messages",
			startDate:     timePtr(now.Add(30 * time.Minute)),
			endDate:       nil,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := searchManager.FilterByDateRange(tt.startDate, tt.endDate)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(results))
		})
	}
}

func TestSearchManager_FilterByMessageType(t *testing.T) {
	store, searchManager := setupSearchTestDB(t)
	defer store.Close()

	// Create test conversation
	conversationID := "test-conv-3"
	_, err := store.CreateConversation(conversationID, "Type Test Conversation")
	require.NoError(t, err)

	// Add messages with different roles
	messages := []*Message{
		{ConversationID: conversationID, Role: "user", Content: "User message 1", Timestamp: time.Now()},
		{ConversationID: conversationID, Role: "user", Content: "User message 2", Timestamp: time.Now().Add(time.Minute)},
		{ConversationID: conversationID, Role: "assistant", Content: "Assistant message 1", Timestamp: time.Now().Add(2 * time.Minute)},
		{ConversationID: conversationID, Role: "tool", Content: "Tool result", Timestamp: time.Now().Add(3 * time.Minute)},
	}

	for _, msg := range messages {
		err := store.AddMessage(msg)
		require.NoError(t, err)
	}

	tests := []struct {
		messageType   string
		expectedCount int
	}{
		{"user", 2},
		{"assistant", 1},
		{"tool", 1},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.messageType, func(t *testing.T) {
			results, err := searchManager.FilterByMessageType(tt.messageType)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(results))

			for _, result := range results {
				assert.Equal(t, tt.messageType, result.Role)
			}
		})
	}
}

func TestSearchManager_FilterByConversation(t *testing.T) {
	store, searchManager := setupSearchTestDB(t)
	defer store.Close()

	// Create multiple test conversations
	conv1ID := "test-conv-4a"
	conv2ID := "test-conv-4b"
	
	_, err := store.CreateConversation(conv1ID, "Conversation 1")
	require.NoError(t, err)
	_, err = store.CreateConversation(conv2ID, "Conversation 2")
	require.NoError(t, err)

	// Add messages to different conversations
	messages := []*Message{
		{ConversationID: conv1ID, Role: "user", Content: "Message in conv 1", Timestamp: time.Now()},
		{ConversationID: conv1ID, Role: "assistant", Content: "Response in conv 1", Timestamp: time.Now().Add(time.Minute)},
		{ConversationID: conv2ID, Role: "user", Content: "Message in conv 2", Timestamp: time.Now().Add(2 * time.Minute)},
	}

	for _, msg := range messages {
		err := store.AddMessage(msg)
		require.NoError(t, err)
	}

	// Test filtering by conversation
	results1, err := searchManager.FilterByConversation(conv1ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(results1))
	for _, result := range results1 {
		assert.Equal(t, conv1ID, result.ConversationID)
	}

	results2, err := searchManager.FilterByConversation(conv2ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results2))
	assert.Equal(t, conv2ID, results2[0].ConversationID)

	// Test non-existent conversation
	results3, err := searchManager.FilterByConversation("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, 0, len(results3))
}

func TestSearchManager_CombinedFilters(t *testing.T) {
	store, searchManager := setupSearchTestDB(t)
	defer store.Close()

	// Create test conversation
	conversationID := "test-conv-5"
	_, err := store.CreateConversation(conversationID, "Combined Filter Test")
	require.NoError(t, err)

	// Add test messages
	now := time.Now()
	messages := []*Message{
		{ConversationID: conversationID, Role: "user", Content: "Hello world", Timestamp: now.Add(-2 * time.Hour)},
		{ConversationID: conversationID, Role: "assistant", Content: "Hello there", Timestamp: now.Add(-1 * time.Hour)},
		{ConversationID: conversationID, Role: "user", Content: "Goodbye world", Timestamp: now},
		{ConversationID: conversationID, Role: "tool", Content: "Tool result", Timestamp: now.Add(time.Hour)},
	}

	for _, msg := range messages {
		err := store.AddMessage(msg)
		require.NoError(t, err)
	}

	// Test combined filters
	filter := SearchFilter{
		Query:          "world",
		MessageType:    "user",
		StartDate:      timePtr(now.Add(-3 * time.Hour)),
		EndDate:        timePtr(now.Add(30 * time.Minute)),
		ConversationID: conversationID,
	}

	results, err := searchManager.SearchMessages(filter)
	require.NoError(t, err)
	
	// Should find both user messages containing "world" within the time range
	assert.Equal(t, 2, len(results))
	for _, result := range results {
		assert.Equal(t, "user", result.Role)
		assert.Contains(t, result.Content, "world")
		assert.Equal(t, conversationID, result.ConversationID)
	}
}

func TestSearchManager_SearchConversations(t *testing.T) {
	store, searchManager := setupSearchTestDB(t)
	defer store.Close()

	// Create test conversations with different titles
	conversations := []struct {
		id    string
		title string
	}{
		{"conv-1", "Machine Learning Discussion"},
		{"conv-2", "Python Programming Help"},
		{"conv-3", "Learning About AI"},
		{"conv-4", "Random Chat"},
	}

	for _, conv := range conversations {
		_, err := store.CreateConversation(conv.id, conv.title)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedTitles []string
	}{
		{
			name:          "search by learning",
			query:         "learning",
			expectedCount: 2,
			expectedTitles: []string{"Machine Learning Discussion", "Learning About AI"},
		},
		{
			name:          "case insensitive search",
			query:         "PYTHON",
			expectedCount: 1,
			expectedTitles: []string{"Python Programming Help"},
		},
		{
			name:          "no matches",
			query:         "nonexistent",
			expectedCount: 0,
		},
		{
			name:          "empty query returns all",
			query:         "",
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := searchManager.SearchConversations(tt.query, 0)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCount, len(results))
			
			if len(tt.expectedTitles) > 0 {
				var resultTitles []string
				for _, result := range results {
					resultTitles = append(resultTitles, result.Title)
				}
				
				for _, expectedTitle := range tt.expectedTitles {
					assert.Contains(t, resultTitles, expectedTitle)
				}
			}
		})
	}
}

func TestSearchManager_GetSearchStatistics(t *testing.T) {
	store, searchManager := setupSearchTestDB(t)
	defer store.Close()

	// Initially should have default statistics
	stats := searchManager.GetSearchStatistics()
	assert.Equal(t, 0, stats.TotalQueries)
	assert.Equal(t, 0, stats.CacheHits)
	assert.Equal(t, 0, stats.CacheMisses)
	assert.Equal(t, time.Duration(0), stats.AverageQueryTime)
	assert.WithinDuration(t, time.Now(), stats.LastUpdated, time.Second)

	// Perform some searches to update statistics
	filter := SearchFilter{Query: "test"}
	_, err := searchManager.SearchMessages(filter)
	require.NoError(t, err)

	_, err = searchManager.SearchConversations("test", 0)
	require.NoError(t, err)

	// Check updated statistics
	stats = searchManager.GetSearchStatistics()
	assert.Equal(t, 2, stats.TotalQueries)
	assert.Greater(t, stats.AverageQueryTime, time.Duration(0))
	assert.WithinDuration(t, time.Now(), stats.LastUpdated, time.Second)
}

// Helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}