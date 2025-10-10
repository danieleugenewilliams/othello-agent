package storage

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *ConversationStore {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	store, err := NewConversationStore(dbPath)
	require.NoError(t, err, "Failed to create test database")
	
	return store
}

func TestNewConversationStore(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "valid database path",
			dbPath:  filepath.Join(t.TempDir(), "test.db"),
			wantErr: false,
		},
		{
			name:    "memory database",
			dbPath:  ":memory:",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewConversationStore(tt.dbPath)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, store)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, store)
				assert.NotNil(t, store.db)
			}
		})
	}
}

func TestCreateConversation(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	tests := []struct {
		name    string
		id      string
		title   string
		wantErr bool
	}{
		{
			name:    "valid conversation",
			id:      "conv-123",
			title:   "Test Conversation",
			wantErr: false,
		},
		{
			name:    "empty title",
			id:      "conv-124",
			title:   "",
			wantErr: false, // Empty title should be allowed
		},
		{
			name:    "duplicate id",
			id:      "conv-123",
			title:   "Duplicate",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := store.CreateConversation(tt.id, tt.title)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, conv)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, conv)
				assert.Equal(t, tt.id, conv.ID)
				assert.Equal(t, tt.title, conv.Title)
				assert.False(t, conv.CreatedAt.IsZero())
				assert.False(t, conv.UpdatedAt.IsZero())
				assert.Equal(t, 0, conv.MessageCount)
				assert.Equal(t, 0, conv.TotalTokens)
			}
		})
	}
}

func TestGetConversation(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create test conversation
	created, err := store.CreateConversation("test-conv", "Test Title")
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      string
		want    *Conversation
		wantErr bool
	}{
		{
			name: "existing conversation",
			id:   "test-conv",
			want: created,
		},
		{
			name: "non-existent conversation",
			id:   "not-found",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := store.GetConversation(tt.id)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.want == nil {
					assert.Nil(t, conv)
				} else {
					assert.NotNil(t, conv)
					assert.Equal(t, tt.want.ID, conv.ID)
					assert.Equal(t, tt.want.Title, conv.Title)
				}
			}
		})
	}
}

func TestListConversations(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create multiple conversations with different timestamps
	convs := []*Conversation{}
	for i := 0; i < 5; i++ {
		conv, err := store.CreateConversation(
			fmt.Sprintf("conv-%d", i),
			fmt.Sprintf("Conversation %d", i),
		)
		require.NoError(t, err)
		convs = append(convs, conv)
		
		// Add small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	tests := []struct {
		name   string
		limit  int
		offset int
		want   int // expected count
	}{
		{
			name:   "get all conversations",
			limit:  10,
			offset: 0,
			want:   5,
		},
		{
			name:   "limit to 3",
			limit:  3,
			offset: 0,
			want:   3,
		},
		{
			name:   "offset by 2",
			limit:  10,
			offset: 2,
			want:   3,
		},
		{
			name:   "beyond available",
			limit:  10,
			offset: 10,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := store.ListConversations(tt.limit, tt.offset)
			assert.NoError(t, err)
			assert.Len(t, result, tt.want)
			
			// Verify ordering (newest first)
			if len(result) > 1 {
				for i := 1; i < len(result); i++ {
					assert.True(t, result[i-1].UpdatedAt.After(result[i].UpdatedAt) || 
						result[i-1].UpdatedAt.Equal(result[i].UpdatedAt))
				}
			}
		})
	}
}

func TestAddMessage(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create conversation
	conv, err := store.CreateConversation("test-conv", "Test")
	require.NoError(t, err)

	tests := []struct {
		name    string
		message *Message
		wantErr bool
	}{
		{
			name: "user message",
			message: &Message{
				ConversationID: conv.ID,
				Role:          "user",
				Content:       "Hello, world!",
				Timestamp:     time.Now(),
				TokenCount:    5,
			},
			wantErr: false,
		},
		{
			name: "assistant message",
			message: &Message{
				ConversationID: conv.ID,
				Role:          "assistant",
				Content:       "Hello! How can I help you?",
				Timestamp:     time.Now(),
				TokenCount:    8,
			},
			wantErr: false,
		},
		{
			name: "tool call message",
			message: &Message{
				ConversationID: conv.ID,
				Role:          "tool",
				Content:       "",
				ToolCall: &ToolCall{
					ID:   "call-123",
					Name: "search",
					Arguments: map[string]interface{}{
						"query": "test search",
					},
				},
				Timestamp:  time.Now(),
				TokenCount: 3,
			},
			wantErr: false,
		},
		{
			name: "tool result message",
			message: &Message{
				ConversationID: conv.ID,
				Role:          "tool",
				Content:       "Search results",
				ToolResult: &ToolResult{
					ID:      "call-123",
					Content: "Found 5 results",
					IsError: false,
				},
				Timestamp:  time.Now(),
				TokenCount: 10,
			},
			wantErr: false,
		},
		{
			name: "invalid role",
			message: &Message{
				ConversationID: conv.ID,
				Role:          "invalid",
				Content:       "Test",
				Timestamp:     time.Now(),
				TokenCount:    1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.AddMessage(tt.message)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.message.ID)
			}
		})
	}
}

func TestGetMessages(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create conversation
	conv, err := store.CreateConversation("test-conv", "Test")
	require.NoError(t, err)

	// Add test messages
	messages := []*Message{
		{
			ConversationID: conv.ID,
			Role:          "user",
			Content:       "First message",
			Timestamp:     time.Now().Add(-2 * time.Minute),
			TokenCount:    3,
		},
		{
			ConversationID: conv.ID,
			Role:          "assistant",
			Content:       "Second message",
			Timestamp:     time.Now().Add(-1 * time.Minute),
			TokenCount:    3,
		},
		{
			ConversationID: conv.ID,
			Role:          "user",
			Content:       "Third message",
			Timestamp:     time.Now(),
			TokenCount:    3,
		},
	}

	for _, msg := range messages {
		err := store.AddMessage(msg)
		require.NoError(t, err)
	}

	tests := []struct {
		name   string
		convID string
		limit  int
		offset int
		want   int
	}{
		{
			name:   "get all messages",
			convID: conv.ID,
			limit:  10,
			offset: 0,
			want:   3,
		},
		{
			name:   "limit to 2",
			convID: conv.ID,
			limit:  2,
			offset: 0,
			want:   2,
		},
		{
			name:   "non-existent conversation",
			convID: "not-found",
			limit:  10,
			offset: 0,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := store.GetMessages(tt.convID, tt.limit, tt.offset)
			assert.NoError(t, err)
			assert.Len(t, result, tt.want)
			
			// Verify ordering (oldest first)
			if len(result) > 1 {
				for i := 1; i < len(result); i++ {
					assert.True(t, result[i-1].Timestamp.Before(result[i].Timestamp) || 
						result[i-1].Timestamp.Equal(result[i].Timestamp))
				}
			}
		})
	}
}

func TestUpdateConversationStats(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create conversation
	conv, err := store.CreateConversation("test-conv", "Test")
	require.NoError(t, err)

	// Add messages
	messages := []*Message{
		{
			ConversationID: conv.ID,
			Role:          "user",
			Content:       "Hello",
			Timestamp:     time.Now(),
			TokenCount:    5,
		},
		{
			ConversationID: conv.ID,
			Role:          "assistant",
			Content:       "Hi there!",
			Timestamp:     time.Now(),
			TokenCount:    3,
		},
	}

	for _, msg := range messages {
		err := store.AddMessage(msg)
		require.NoError(t, err)
	}

	// Update stats
	err = store.UpdateConversationStats(conv.ID)
	assert.NoError(t, err)

	// Verify updated conversation
	updated, err := store.GetConversation(conv.ID)
	require.NoError(t, err)
	
	assert.Equal(t, 2, updated.MessageCount)
	assert.Equal(t, 8, updated.TotalTokens) // 5 + 3
	assert.True(t, updated.UpdatedAt.After(conv.UpdatedAt))
}

func TestDeleteConversation(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create conversation with messages
	conv, err := store.CreateConversation("test-conv", "Test")
	require.NoError(t, err)

	msg := &Message{
		ConversationID: conv.ID,
		Role:          "user",
		Content:       "Test message",
		Timestamp:     time.Now(),
		TokenCount:    2,
	}
	err = store.AddMessage(msg)
	require.NoError(t, err)

	// Delete conversation
	err = store.DeleteConversation(conv.ID)
	assert.NoError(t, err)

	// Verify deletion
	retrieved, err := store.GetConversation(conv.ID)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)

	// Verify messages are also deleted (CASCADE)
	messages, err := store.GetMessages(conv.ID, 10, 0)
	assert.NoError(t, err)
	assert.Empty(t, messages)
}

func TestClose(t *testing.T) {
	store := setupTestDB(t)
	
	// Close should not error
	err := store.Close()
	assert.NoError(t, err)
	
	// Operations after close should fail
	_, err = store.CreateConversation("test", "test")
	assert.Error(t, err)
}

func TestConcurrentAccess(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Create conversation
	conv, err := store.CreateConversation("concurrent-test", "Concurrent Test")
	require.NoError(t, err)

	// Concurrent message adding
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- true }()
			
			msg := &Message{
				ConversationID: conv.ID,
				Role:          "user",
				Content:       fmt.Sprintf("Message %d", n),
				Timestamp:     time.Now(),
				TokenCount:    2,
			}
			
			err := store.AddMessage(msg)
			assert.NoError(t, err)
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify all messages were added
	messages, err := store.GetMessages(conv.ID, 20, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 10)
}