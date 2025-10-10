package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockNotificationHandler is a mock implementation of NotificationHandler
type MockNotificationHandler struct {
	mock.Mock
}

func (m *MockNotificationHandler) OnNotification(notification Notification) error {
	args := m.Called(notification)
	return args.Error(0)
}

func (m *MockNotificationHandler) OnServerStatusChange(serverName string, status ServerStatus) error {
	args := m.Called(serverName, status)
	return args.Error(0)
}

func (m *MockNotificationHandler) OnResourceChange(resourceURI string, changeType ResourceChangeType) error {
	args := m.Called(resourceURI, changeType)
	return args.Error(0)
}

func (m *MockNotificationHandler) OnToolListChange(serverName string) error {
	args := m.Called(serverName)
	return args.Error(0)
}

// Test notification types and structures
func TestNotificationTypes(t *testing.T) {
	tests := []struct {
		name               string
		notificationType   NotificationType
		expectedString     string
		validNotification  bool
	}{
		{
			name:               "Resource update notification",
			notificationType:   NotificationTypeResourceUpdate,
			expectedString:     "resource_update",
			validNotification:  true,
		},
		{
			name:               "Tool list changed notification",
			notificationType:   NotificationTypeToolListChanged,
			expectedString:     "tool_list_changed",
			validNotification:  true,
		},
		{
			name:               "Server status notification",
			notificationType:   NotificationTypeServerStatus,
			expectedString:     "server_status",
			validNotification:  true,
		},
		{
			name:               "Progress notification",
			notificationType:   NotificationTypeProgress,
			expectedString:     "progress",
			validNotification:  true,
		},
		{
			name:               "Unknown notification type",
			notificationType:   NotificationType("unknown"),
			expectedString:     "unknown",
			validNotification:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedString, string(tt.notificationType))
			
			// Test validation
			isValid := tt.notificationType.IsValid()
			assert.Equal(t, tt.validNotification, isValid)
		})
	}
}

func TestServerStatusTypes(t *testing.T) {
	tests := []struct {
		name           string
		status         ServerStatus
		expectedString string
		isHealthy      bool
	}{
		{
			name:           "Server connected",
			status:         ServerStatusConnected,
			expectedString: "connected",
			isHealthy:      true,
		},
		{
			name:           "Server disconnected",
			status:         ServerStatusDisconnected,
			expectedString: "disconnected",
			isHealthy:      false,
		},
		{
			name:           "Server error",
			status:         ServerStatusError,
			expectedString: "error",
			isHealthy:      false,
		},
		{
			name:           "Server reconnecting",
			status:         ServerStatusReconnecting,
			expectedString: "reconnecting",
			isHealthy:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedString, string(tt.status))
			assert.Equal(t, tt.isHealthy, tt.status.IsHealthy())
		})
	}
}

func TestResourceChangeTypes(t *testing.T) {
	tests := []struct {
		name           string
		changeType     ResourceChangeType
		expectedString string
	}{
		{
			name:           "Resource created",
			changeType:     ResourceChangeTypeCreated,
			expectedString: "created",
		},
		{
			name:           "Resource updated",
			changeType:     ResourceChangeTypeUpdated,
			expectedString: "updated",
		},
		{
			name:           "Resource deleted",
			changeType:     ResourceChangeTypeDeleted,
			expectedString: "deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedString, string(tt.changeType))
		})
	}
}

func TestNotificationSerialization(t *testing.T) {
	tests := []struct {
		name         string
		notification Notification
	}{
		{
			name: "Resource update notification",
			notification: Notification{
				Type: NotificationTypeResourceUpdate,
				Data: map[string]interface{}{
					"resource_uri": "file:///test.txt",
					"change_type":  "updated",
				},
				Timestamp: time.Unix(1672531200, 0).UTC(),
				ServerName: "test-server",
			},
		},
		{
			name: "Server status notification",
			notification: Notification{
				Type: NotificationTypeServerStatus,
				Data: map[string]interface{}{
					"status": "connected",
					"message": "Server connected successfully",
				},
				Timestamp: time.Unix(1672531200, 0).UTC(),
				ServerName: "test-server",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonBytes, err := json.Marshal(tt.notification)
			require.NoError(t, err)
			
			// Test unmarshaling
			var unmarshaled Notification
			err = json.Unmarshal(jsonBytes, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, tt.notification.Type, unmarshaled.Type)
			assert.Equal(t, tt.notification.Data, unmarshaled.Data)
			assert.Equal(t, tt.notification.ServerName, unmarshaled.ServerName)
			// Compare timestamps with second precision (ignore nanoseconds)
			assert.Equal(t, tt.notification.Timestamp.Unix(), unmarshaled.Timestamp.Unix())
		})
	}
}

func TestNotificationManager_Subscribe(t *testing.T) {
	manager := NewNotificationManager()
	handler := &MockNotificationHandler{}

	// Test subscription
	unsubscribe := manager.Subscribe(handler)
	assert.NotNil(t, unsubscribe)

	// Verify handler is added
	manager.mu.RLock()
	assert.Len(t, manager.handlers, 1)
	manager.mu.RUnlock()

	// Test unsubscribe
	unsubscribe()
	
	manager.mu.RLock()
	assert.Len(t, manager.handlers, 0)
	manager.mu.RUnlock()
}

func TestNotificationManager_Notify(t *testing.T) {
	manager := NewNotificationManager()
	handler := &MockNotificationHandler{}

	notification := Notification{
		Type: NotificationTypeResourceUpdate,
		Data: map[string]interface{}{
			"resource_uri": "file:///test.txt",
			"change_type":  "updated",
		},
		Timestamp:  time.Now(),
		ServerName: "test-server",
	}

	// Setup expectations
	handler.On("OnNotification", notification).Return(nil)

	// Subscribe handler
	unsubscribe := manager.Subscribe(handler)
	defer unsubscribe()

	// Send notification
	err := manager.Notify(notification)
	require.NoError(t, err)

	// Wait for async notification to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify mock expectations
	handler.AssertExpectations(t)
}

func TestNotificationManager_NotifyMultipleHandlers(t *testing.T) {
	manager := NewNotificationManager()
	handler1 := &MockNotificationHandler{}
	handler2 := &MockNotificationHandler{}

	notification := Notification{
		Type:       NotificationTypeServerStatus,
		Data:       map[string]interface{}{"status": "connected"},
		Timestamp:  time.Now(),
		ServerName: "test-server",
	}

	// Setup expectations for both handlers
	handler1.On("OnNotification", notification).Return(nil)
	handler2.On("OnNotification", notification).Return(nil)

	// Subscribe both handlers
	unsubscribe1 := manager.Subscribe(handler1)
	unsubscribe2 := manager.Subscribe(handler2)
	defer unsubscribe1()
	defer unsubscribe2()

	// Send notification
	err := manager.Notify(notification)
	require.NoError(t, err)

	// Wait for async notifications to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify both handlers were called
	handler1.AssertExpectations(t)
	handler2.AssertExpectations(t)
}

func TestNotificationManager_NotifyWithError(t *testing.T) {
	manager := NewNotificationManager()
	handler := &MockNotificationHandler{}

	notification := Notification{
		Type:       NotificationTypeProgress,
		Data:       map[string]interface{}{"progress": 50},
		Timestamp:  time.Now(),
		ServerName: "test-server",
	}

	// Setup handler to return error
	handler.On("OnNotification", notification).Return(assert.AnError)

	// Subscribe handler
	unsubscribe := manager.Subscribe(handler)
	defer unsubscribe()

	// Send notification - should not return error (errors are logged internally)
	err := manager.Notify(notification)
	require.NoError(t, err)

	// Wait for async notification to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify handler was called
	handler.AssertExpectations(t)
}

func TestNotificationBuffer(t *testing.T) {
	buffer := NewNotificationBuffer(3)

	notifications := []Notification{
		{Type: NotificationTypeResourceUpdate, ServerName: "server1", Timestamp: time.Now()},
		{Type: NotificationTypeServerStatus, ServerName: "server2", Timestamp: time.Now()},
		{Type: NotificationTypeToolListChanged, ServerName: "server3", Timestamp: time.Now()},
		{Type: NotificationTypeProgress, ServerName: "server4", Timestamp: time.Now()},
	}

	// Add notifications
	for _, n := range notifications {
		buffer.Add(n)
	}

	// Should only keep the last 3 (buffer size)
	recent := buffer.GetRecent(10)
	assert.Len(t, recent, 3)
	
	// Should be in reverse chronological order (most recent first)
	assert.Equal(t, notifications[3].Type, recent[0].Type) // Last added should be first
	assert.Equal(t, notifications[2].Type, recent[1].Type)
	assert.Equal(t, notifications[1].Type, recent[2].Type)

	// Test getting limited number
	limited := buffer.GetRecent(2)
	assert.Len(t, limited, 2)
	assert.Equal(t, notifications[3].Type, limited[0].Type)
	assert.Equal(t, notifications[2].Type, limited[1].Type)
}

func TestNotificationBuffer_Concurrent(t *testing.T) {
	buffer := NewNotificationBuffer(10)
	
	// Test concurrent access
	done := make(chan bool, 2)
	
	// Writer goroutine
	go func() {
		for i := 0; i < 50; i++ {
			buffer.Add(Notification{
				Type:       NotificationTypeProgress,
				ServerName: "test",
				Timestamp:  time.Now(),
				Data:       map[string]interface{}{"i": i},
			})
		}
		done <- true
	}()
	
	// Reader goroutine
	go func() {
		for i := 0; i < 50; i++ {
			buffer.GetRecent(5)
		}
		done <- true
	}()
	
	// Wait for both goroutines
	<-done
	<-done
	
	// Buffer should have at most 10 items
	recent := buffer.GetRecent(20)
	assert.LessOrEqual(t, len(recent), 10)
}

func TestNotificationFilter(t *testing.T) {
	filter := NewNotificationFilter()

	// Add filters
	filter.AddTypeFilter(NotificationTypeResourceUpdate)
	filter.AddServerFilter("server1")

	tests := []struct {
		name         string
		notification Notification
		shouldPass   bool
	}{
		{
			name: "Matching type and server",
			notification: Notification{
				Type:       NotificationTypeResourceUpdate,
				ServerName: "server1",
			},
			shouldPass: true,
		},
		{
			name: "Matching type, different server",
			notification: Notification{
				Type:       NotificationTypeResourceUpdate,
				ServerName: "server2",
			},
			shouldPass: false,
		},
		{
			name: "Different type, matching server",
			notification: Notification{
				Type:       NotificationTypeServerStatus,
				ServerName: "server1",
			},
			shouldPass: false,
		},
		{
			name: "Different type and server",
			notification: Notification{
				Type:       NotificationTypeProgress,
				ServerName: "server2",
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.ShouldProcess(tt.notification)
			assert.Equal(t, tt.shouldPass, result)
		})
	}
}

func TestNotificationFilter_NoFilters(t *testing.T) {
	filter := NewNotificationFilter()

	notification := Notification{
		Type:       NotificationTypeProgress,
		ServerName: "any-server",
	}

	// With no filters, should pass all notifications
	assert.True(t, filter.ShouldProcess(notification))
}

func TestNotificationManagerIntegration(t *testing.T) {
	manager := NewNotificationManager()
	
	// Setup notification buffer and filter
	buffer := NewNotificationBuffer(5)
	filter := NewNotificationFilter()
	filter.AddTypeFilter(NotificationTypeResourceUpdate)

	receivedNotifications := make([]Notification, 0)
	handler := &MockNotificationHandler{}

	// Mock handler to collect notifications
	handler.On("OnNotification", mock.AnythingOfType("mcp.Notification")).Run(func(args mock.Arguments) {
		notification := args.Get(0).(Notification)
		if filter.ShouldProcess(notification) {
			buffer.Add(notification)
			receivedNotifications = append(receivedNotifications, notification)
		}
	}).Return(nil)

	// Subscribe handler
	unsubscribe := manager.Subscribe(handler)
	defer unsubscribe()

	// Send various notifications
	notifications := []Notification{
		{Type: NotificationTypeResourceUpdate, ServerName: "server1", Timestamp: time.Now()},
		{Type: NotificationTypeServerStatus, ServerName: "server1", Timestamp: time.Now()},
		{Type: NotificationTypeResourceUpdate, ServerName: "server2", Timestamp: time.Now()},
		{Type: NotificationTypeProgress, ServerName: "server1", Timestamp: time.Now()},
	}

	for _, n := range notifications {
		err := manager.Notify(n)
		require.NoError(t, err)
	}

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify only filtered notifications were processed
	assert.Len(t, receivedNotifications, 2) // Only resource_update notifications
	
	// Verify buffer contains the filtered notifications
	buffered := buffer.GetRecent(10)
	assert.Len(t, buffered, 2)
	
	for _, n := range buffered {
		assert.Equal(t, NotificationTypeResourceUpdate, n.Type)
	}

	handler.AssertExpectations(t)
}