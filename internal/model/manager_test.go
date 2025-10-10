package model

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockModel is a mock implementation of the Model interface for testing
type MockModel struct {
	mock.Mock
}

func (m *MockModel) Generate(ctx context.Context, prompt string, options GenerateOptions) (*Response, error) {
	args := m.Called(ctx, prompt, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Response), args.Error(1)
}

func (m *MockModel) Chat(ctx context.Context, messages []Message, options GenerateOptions) (*Response, error) {
	args := m.Called(ctx, messages, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Response), args.Error(1)
}

func (m *MockModel) IsAvailable(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

// Test Model Manager functionality

func TestNewManager(t *testing.T) {
	manager := NewManager()
	assert.NotNil(t, manager)
	assert.Equal(t, "", manager.GetCurrentBackend())
}

func TestManager_RegisterBackend(t *testing.T) {
	manager := NewManager()
	mock1 := new(MockModel)
	mock2 := new(MockModel)

	// Register backends
	err := manager.RegisterBackend("ollama", mock1)
	assert.NoError(t, err)

	err = manager.RegisterBackend("openai", mock2)
	assert.NoError(t, err)

	// Try registering duplicate
	err = manager.RegisterBackend("ollama", mock1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestManager_SwitchBackend(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)
	mockModel.On("IsAvailable", mock.Anything).Return(true)

	manager.RegisterBackend("test", mockModel)

	// Switch to valid backend
	err := manager.SwitchBackend("test")
	assert.NoError(t, err)
	assert.Equal(t, "test", manager.GetCurrentBackend())

	// Try switching to non-existent backend
	err = manager.SwitchBackend("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestManager_SwitchBackend_Unavailable(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)
	mockModel.On("IsAvailable", mock.Anything).Return(false)

	manager.RegisterBackend("unavailable", mockModel)

	err := manager.SwitchBackend("unavailable")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestManager_Generate(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)

	expectedResponse := &Response{
		Content:  "Test response",
		Duration: 100 * time.Millisecond,
	}

	mockModel.On("IsAvailable", mock.Anything).Return(true)
	mockModel.On("Generate", mock.Anything, "Test prompt", mock.Anything).Return(expectedResponse, nil)

	manager.RegisterBackend("test", mockModel)
	manager.SwitchBackend("test")

	ctx := context.Background()
	resp, err := manager.Generate(ctx, "Test prompt", GenerateOptions{})

	assert.NoError(t, err)
	assert.Equal(t, "Test response", resp.Content)
	mockModel.AssertExpectations(t)
}

func TestManager_Generate_NoBackend(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	_, err := manager.Generate(ctx, "Test", GenerateOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backend")
}

func TestManager_Chat(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	expectedResponse := &Response{
		Content:  "Chat response",
		Duration: 150 * time.Millisecond,
	}

	mockModel.On("IsAvailable", mock.Anything).Return(true)
	mockModel.On("Chat", mock.Anything, messages, mock.Anything).Return(expectedResponse, nil)

	manager.RegisterBackend("test", mockModel)
	manager.SwitchBackend("test")

	ctx := context.Background()
	resp, err := manager.Chat(ctx, messages, GenerateOptions{})

	assert.NoError(t, err)
	assert.Equal(t, "Chat response", resp.Content)
	mockModel.AssertExpectations(t)
}

func TestManager_ListBackends(t *testing.T) {
	manager := NewManager()
	mock1 := new(MockModel)
	mock2 := new(MockModel)
	mock3 := new(MockModel)

	mock1.On("IsAvailable", mock.Anything).Return(true)
	mock2.On("IsAvailable", mock.Anything).Return(false)
	mock3.On("IsAvailable", mock.Anything).Return(true)

	manager.RegisterBackend("backend1", mock1)
	manager.RegisterBackend("backend2", mock2)
	manager.RegisterBackend("backend3", mock3)

	backends := manager.ListBackends()
	assert.Len(t, backends, 3)
	assert.Contains(t, backends, BackendInfo{Name: "backend1", Available: true, Current: false})
	assert.Contains(t, backends, BackendInfo{Name: "backend2", Available: false, Current: false})
	assert.Contains(t, backends, BackendInfo{Name: "backend3", Available: true, Current: false})
}

func TestManager_ListBackends_WithCurrent(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)
	mockModel.On("IsAvailable", mock.Anything).Return(true)

	manager.RegisterBackend("test", mockModel)
	manager.SwitchBackend("test")

	backends := manager.ListBackends()
	assert.Len(t, backends, 1)
	assert.Equal(t, BackendInfo{Name: "test", Available: true, Current: true}, backends[0])
}

func TestManager_GetCurrentModel(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)
	mockModel.On("IsAvailable", mock.Anything).Return(true)

	// No current model
	model := manager.GetCurrentModel()
	assert.Nil(t, model)

	// Set current model
	manager.RegisterBackend("test", mockModel)
	manager.SwitchBackend("test")

	model = manager.GetCurrentModel()
	assert.NotNil(t, model)
}

func TestManager_AutoSelectBestBackend(t *testing.T) {
	manager := NewManager()
	mock1 := new(MockModel)
	mock2 := new(MockModel)
	mock3 := new(MockModel)

	// Only mock2 is available
	mock1.On("IsAvailable", mock.Anything).Return(false)
	mock2.On("IsAvailable", mock.Anything).Return(true)
	mock3.On("IsAvailable", mock.Anything).Return(false)

	manager.RegisterBackend("backend1", mock1)
	manager.RegisterBackend("backend2", mock2)
	manager.RegisterBackend("backend3", mock3)

	err := manager.AutoSelectBestBackend()
	assert.NoError(t, err)
	assert.Equal(t, "backend2", manager.GetCurrentBackend())
}

func TestManager_AutoSelectBestBackend_NoneAvailable(t *testing.T) {
	manager := NewManager()
	mock1 := new(MockModel)
	mock2 := new(MockModel)

	mock1.On("IsAvailable", mock.Anything).Return(false)
	mock2.On("IsAvailable", mock.Anything).Return(false)

	manager.RegisterBackend("backend1", mock1)
	manager.RegisterBackend("backend2", mock2)

	err := manager.AutoSelectBestBackend()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available backends")
}

func TestManager_ConcurrentAccess(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)

	mockModel.On("IsAvailable", mock.Anything).Return(true)
	mockModel.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return(&Response{Content: "Response"}, nil)

	manager.RegisterBackend("test", mockModel)
	manager.SwitchBackend("test")

	// Concurrent calls to Generate
	ctx := context.Background()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := manager.Generate(ctx, "Test", GenerateOptions{})
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestManager_ErrorPropagation(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)

	expectedError := errors.New("model error")
	mockModel.On("IsAvailable", mock.Anything).Return(true)
	mockModel.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return(nil, expectedError)

	manager.RegisterBackend("test", mockModel)
	manager.SwitchBackend("test")

	ctx := context.Background()
	_, err := manager.Generate(ctx, "Test", GenerateOptions{})

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestManager_UnregisterBackend(t *testing.T) {
	manager := NewManager()
	mockModel := new(MockModel)
	mockModel.On("IsAvailable", mock.Anything).Return(true)

	manager.RegisterBackend("test", mockModel)
	manager.SwitchBackend("test")

	// Unregister current backend
	err := manager.UnregisterBackend("test")
	assert.NoError(t, err)
	assert.Equal(t, "", manager.GetCurrentBackend())

	// Try unregistering non-existent backend
	err = manager.UnregisterBackend("nonexistent")
	assert.Error(t, err)
}

func TestManager_FallbackBackend(t *testing.T) {
	manager := NewManager()
	primary := new(MockModel)
	fallback := new(MockModel)

	// Primary fails, fallback succeeds
	primary.On("IsAvailable", mock.Anything).Return(true)
	primary.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("primary error"))

	fallback.On("IsAvailable", mock.Anything).Return(true)
	fallback.On("Generate", mock.Anything, mock.Anything, mock.Anything).Return(&Response{Content: "Fallback response"}, nil)

	manager.RegisterBackend("primary", primary)
	manager.RegisterBackend("fallback", fallback)
	manager.SwitchBackend("primary")
	manager.SetFallbackBackend("fallback")

	ctx := context.Background()
	resp, err := manager.Generate(ctx, "Test", GenerateOptions{})

	require.NoError(t, err)
	assert.Equal(t, "Fallback response", resp.Content)
}

func TestBackendInfo_Struct(t *testing.T) {
	info := BackendInfo{
		Name:      "test",
		Available: true,
		Current:   false,
	}

	assert.Equal(t, "test", info.Name)
	assert.True(t, info.Available)
	assert.False(t, info.Current)
}
