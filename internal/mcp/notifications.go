package mcp

import (
	"encoding/json"
	"sync"
	"time"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	// NotificationTypeResourceUpdate indicates a resource has been updated
	NotificationTypeResourceUpdate NotificationType = "resource_update"
	
	// NotificationTypeToolListChanged indicates the tool list has changed
	NotificationTypeToolListChanged NotificationType = "tool_list_changed"
	
	// NotificationTypeServerStatus indicates server status has changed
	NotificationTypeServerStatus NotificationType = "server_status"
	
	// NotificationTypeProgress indicates progress update
	NotificationTypeProgress NotificationType = "progress"
)

// IsValid checks if the notification type is valid
func (nt NotificationType) IsValid() bool {
	switch nt {
	case NotificationTypeResourceUpdate,
		NotificationTypeToolListChanged,
		NotificationTypeServerStatus,
		NotificationTypeProgress:
		return true
	default:
		return false
	}
}

// ServerStatus represents the status of an MCP server
type ServerStatus string

const (
	// ServerStatusConnected indicates server is connected
	ServerStatusConnected ServerStatus = "connected"
	
	// ServerStatusDisconnected indicates server is disconnected
	ServerStatusDisconnected ServerStatus = "disconnected"
	
	// ServerStatusError indicates server is in error state
	ServerStatusError ServerStatus = "error"
	
	// ServerStatusReconnecting indicates server is reconnecting
	ServerStatusReconnecting ServerStatus = "reconnecting"
)

// IsHealthy returns true if the server status is healthy
func (ss ServerStatus) IsHealthy() bool {
	return ss == ServerStatusConnected
}

// ResourceChangeType represents the type of resource change
type ResourceChangeType string

const (
	// ResourceChangeTypeCreated indicates resource was created
	ResourceChangeTypeCreated ResourceChangeType = "created"
	
	// ResourceChangeTypeUpdated indicates resource was updated
	ResourceChangeTypeUpdated ResourceChangeType = "updated"
	
	// ResourceChangeTypeDeleted indicates resource was deleted
	ResourceChangeTypeDeleted ResourceChangeType = "deleted"
)

// Notification represents a notification from an MCP server
type Notification struct {
	Type       NotificationType       `json:"type"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
	ServerName string                 `json:"server_name"`
}

// NotificationHandler is an interface for handling notifications
type NotificationHandler interface {
	OnNotification(notification Notification) error
	OnServerStatusChange(serverName string, status ServerStatus) error
	OnResourceChange(resourceURI string, changeType ResourceChangeType) error
	OnToolListChange(serverName string) error
}

// NotificationManager manages notification subscriptions and distribution
type NotificationManager struct {
	handlers []NotificationHandler
	mu       sync.RWMutex
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		handlers: make([]NotificationHandler, 0),
	}
}

// Subscribe adds a notification handler and returns an unsubscribe function
func (nm *NotificationManager) Subscribe(handler NotificationHandler) func() {
	nm.mu.Lock()
	nm.handlers = append(nm.handlers, handler)
	nm.mu.Unlock()

	return func() {
		nm.mu.Lock()
		defer nm.mu.Unlock()
		for i, h := range nm.handlers {
			if h == handler {
				nm.handlers = append(nm.handlers[:i], nm.handlers[i+1:]...)
				break
			}
		}
	}
}

// Notify sends a notification to all subscribed handlers
func (nm *NotificationManager) Notify(notification Notification) error {
	nm.mu.RLock()
	handlers := make([]NotificationHandler, len(nm.handlers))
	copy(handlers, nm.handlers)
	nm.mu.RUnlock()

	for _, handler := range handlers {
		// Call handler in goroutine to avoid blocking
		go func(h NotificationHandler) {
			if err := h.OnNotification(notification); err != nil {
				// Log error but don't fail the notification
				// In a real application, you'd use a proper logger here
			}
		}(handler)
	}

	return nil
}

// NotifyServerStatus is a convenience method for server status notifications
func (nm *NotificationManager) NotifyServerStatus(serverName string, status ServerStatus) error {
	notification := Notification{
		Type:       NotificationTypeServerStatus,
		Data:       map[string]interface{}{"status": string(status)},
		Timestamp:  time.Now(),
		ServerName: serverName,
	}
	return nm.Notify(notification)
}

// NotifyResourceChange is a convenience method for resource change notifications
func (nm *NotificationManager) NotifyResourceChange(serverName, resourceURI string, changeType ResourceChangeType) error {
	notification := Notification{
		Type: NotificationTypeResourceUpdate,
		Data: map[string]interface{}{
			"resource_uri": resourceURI,
			"change_type":  string(changeType),
		},
		Timestamp:  time.Now(),
		ServerName: serverName,
	}
	return nm.Notify(notification)
}

// NotifyToolListChange is a convenience method for tool list change notifications
func (nm *NotificationManager) NotifyToolListChange(serverName string) error {
	notification := Notification{
		Type:       NotificationTypeToolListChanged,
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
		ServerName: serverName,
	}
	return nm.Notify(notification)
}

// NotificationBuffer maintains a circular buffer of recent notifications
type NotificationBuffer struct {
	notifications []Notification
	maxSize       int
	mu            sync.RWMutex
}

// NewNotificationBuffer creates a new notification buffer with the specified maximum size
func NewNotificationBuffer(maxSize int) *NotificationBuffer {
	return &NotificationBuffer{
		notifications: make([]Notification, 0, maxSize),
		maxSize:       maxSize,
	}
}

// Add adds a notification to the buffer
func (nb *NotificationBuffer) Add(notification Notification) {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	// Add notification
	nb.notifications = append(nb.notifications, notification)

	// Keep only the last maxSize notifications
	if len(nb.notifications) > nb.maxSize {
		nb.notifications = nb.notifications[len(nb.notifications)-nb.maxSize:]
	}
}

// GetRecent returns the most recent n notifications
func (nb *NotificationBuffer) GetRecent(n int) []Notification {
	nb.mu.RLock()
	defer nb.mu.RUnlock()

	// Get the last n notifications (or all if less than n)
	count := n
	if count > len(nb.notifications) {
		count = len(nb.notifications)
	}

	// Return in reverse order (most recent first)
	result := make([]Notification, count)
	for i := 0; i < count; i++ {
		result[i] = nb.notifications[len(nb.notifications)-1-i]
	}

	return result
}

// Clear removes all notifications from the buffer
func (nb *NotificationBuffer) Clear() {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	nb.notifications = nb.notifications[:0]
}

// NotificationFilter filters notifications based on criteria
type NotificationFilter struct {
	types       map[NotificationType]bool
	servers     map[string]bool
	mu          sync.RWMutex
}

// NewNotificationFilter creates a new notification filter
func NewNotificationFilter() *NotificationFilter {
	return &NotificationFilter{
		types:   make(map[NotificationType]bool),
		servers: make(map[string]bool),
	}
}

// AddTypeFilter adds a notification type to filter
func (nf *NotificationFilter) AddTypeFilter(notificationType NotificationType) {
	nf.mu.Lock()
	defer nf.mu.Unlock()
	nf.types[notificationType] = true
}

// RemoveTypeFilter removes a notification type from filter
func (nf *NotificationFilter) RemoveTypeFilter(notificationType NotificationType) {
	nf.mu.Lock()
	defer nf.mu.Unlock()
	delete(nf.types, notificationType)
}

// AddServerFilter adds a server to filter
func (nf *NotificationFilter) AddServerFilter(serverName string) {
	nf.mu.Lock()
	defer nf.mu.Unlock()
	nf.servers[serverName] = true
}

// RemoveServerFilter removes a server from filter
func (nf *NotificationFilter) RemoveServerFilter(serverName string) {
	nf.mu.Lock()
	defer nf.mu.Unlock()
	delete(nf.servers, serverName)
}

// ShouldProcess determines if a notification should be processed based on filters
func (nf *NotificationFilter) ShouldProcess(notification Notification) bool {
	nf.mu.RLock()
	defer nf.mu.RUnlock()

	// If no filters are set, process all notifications
	if len(nf.types) == 0 && len(nf.servers) == 0 {
		return true
	}

	// Check type filter
	typeMatch := len(nf.types) == 0 || nf.types[notification.Type]
	
	// Check server filter
	serverMatch := len(nf.servers) == 0 || nf.servers[notification.ServerName]

	// Both must match if filters are set
	return typeMatch && serverMatch
}

// Clear removes all filters
func (nf *NotificationFilter) Clear() {
	nf.mu.Lock()
	defer nf.mu.Unlock()
	nf.types = make(map[NotificationType]bool)
	nf.servers = make(map[string]bool)
}

// MarshalJSON implements json.Marshaler for custom serialization
func (n Notification) MarshalJSON() ([]byte, error) {
	type Alias Notification
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(&n),
		Timestamp: n.Timestamp.Format(time.RFC3339),
	})
}

// UnmarshalJSON implements json.Unmarshaler for custom deserialization
func (n *Notification) UnmarshalJSON(data []byte) error {
	type Alias Notification
	aux := &struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias: (*Alias)(n),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Timestamp != "" {
		t, err := time.Parse(time.RFC3339, aux.Timestamp)
		if err != nil {
			return err
		}
		n.Timestamp = t
	}

	return nil
}
