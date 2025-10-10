package model

import (
	"context"
	"fmt"
	"sync"
)

// Manager manages multiple model backends
type Manager struct {
	backends        map[string]Model
	currentBackend  string
	fallbackBackend string
	mu              sync.RWMutex
}

// BackendInfo provides information about a model backend
type BackendInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Current   bool   `json:"current"`
}

// NewManager creates a new model manager
func NewManager() *Manager {
	return &Manager{
		backends: make(map[string]Model),
	}
}

// RegisterBackend registers a new model backend
func (m *Manager) RegisterBackend(name string, model Model) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.backends[name]; exists {
		return fmt.Errorf("backend %s already registered", name)
	}

	m.backends[name] = model
	return nil
}

// UnregisterBackend removes a backend
func (m *Manager) UnregisterBackend(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.backends[name]; !exists {
		return fmt.Errorf("backend %s not registered", name)
	}

	delete(m.backends, name)

	// Clear current backend if it was unregistered
	if m.currentBackend == name {
		m.currentBackend = ""
	}

	// Clear fallback backend if it was unregistered
	if m.fallbackBackend == name {
		m.fallbackBackend = ""
	}

	return nil
}

// SwitchBackend switches to a different backend
func (m *Manager) SwitchBackend(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	model, exists := m.backends[name]
	if !exists {
		return fmt.Errorf("backend %s not registered", name)
	}

	// Check if backend is available
	ctx := context.Background()
	if !model.IsAvailable(ctx) {
		return fmt.Errorf("backend %s not available", name)
	}

	m.currentBackend = name
	return nil
}

// SetFallbackBackend sets the fallback backend to use if the current backend fails
func (m *Manager) SetFallbackBackend(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.backends[name]; !exists {
		return fmt.Errorf("backend %s not registered", name)
	}

	m.fallbackBackend = name
	return nil
}

// GetCurrentBackend returns the name of the current backend
func (m *Manager) GetCurrentBackend() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentBackend
}

// GetCurrentModel returns the current model instance
func (m *Manager) GetCurrentModel() Model {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentBackend == "" {
		return nil
	}

	return m.backends[m.currentBackend]
}

// ListBackends returns information about all registered backends
func (m *Manager) ListBackends() []BackendInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx := context.Background()
	backends := make([]BackendInfo, 0, len(m.backends))

	for name, model := range m.backends {
		backends = append(backends, BackendInfo{
			Name:      name,
			Available: model.IsAvailable(ctx),
			Current:   name == m.currentBackend,
		})
	}

	return backends
}

// AutoSelectBestBackend automatically selects the first available backend
func (m *Manager) AutoSelectBestBackend() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx := context.Background()

	// Try to find an available backend
	for name, model := range m.backends {
		if model.IsAvailable(ctx) {
			m.currentBackend = name
			return nil
		}
	}

	return fmt.Errorf("no available backends found")
}

// Generate generates text using the current backend
func (m *Manager) Generate(ctx context.Context, prompt string, options GenerateOptions) (*Response, error) {
	m.mu.RLock()
	currentModel := m.backends[m.currentBackend]
	fallbackModel := m.backends[m.fallbackBackend]
	m.mu.RUnlock()

	if currentModel == nil {
		return nil, fmt.Errorf("no backend selected")
	}

	// Try current backend
	resp, err := currentModel.Generate(ctx, prompt, options)
	if err == nil {
		return resp, nil
	}

	// Try fallback if configured
	if fallbackModel != nil {
		return fallbackModel.Generate(ctx, prompt, options)
	}

	return nil, err
}

// Chat performs a chat completion using the current backend
func (m *Manager) Chat(ctx context.Context, messages []Message, options GenerateOptions) (*Response, error) {
	m.mu.RLock()
	currentModel := m.backends[m.currentBackend]
	fallbackModel := m.backends[m.fallbackBackend]
	m.mu.RUnlock()

	if currentModel == nil {
		return nil, fmt.Errorf("no backend selected")
	}

	// Try current backend
	resp, err := currentModel.Chat(ctx, messages, options)
	if err == nil {
		return resp, nil
	}

	// Try fallback if configured
	if fallbackModel != nil {
		return fallbackModel.Chat(ctx, messages, options)
	}

	return nil, err
}

// IsAvailable checks if the current backend is available
func (m *Manager) IsAvailable(ctx context.Context) bool {
	m.mu.RLock()
	currentModel := m.backends[m.currentBackend]
	m.mu.RUnlock()

	if currentModel == nil {
		return false
	}

	return currentModel.IsAvailable(ctx)
}
