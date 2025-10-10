package storage

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCacheManager(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int
		expected int
	}{
		{"normal size", 10, 10},
		{"zero size", 0, 100}, // Should default to 100
		{"negative size", -5, 100}, // Should default to 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewCacheManager(tt.maxSize)
			defer cm.Close()

			assert.NotNil(t, cm)
			assert.Equal(t, tt.expected, cm.maxSize)
			assert.Equal(t, 0, len(cm.entries))
			assert.Equal(t, 0, len(cm.lruOrder))
		})
	}
}

func TestCacheManager_SetAndGet(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string value", "key1", "hello world"},
		{"int value", "key2", 42},
		{"map value", "key3", map[string]string{"foo": "bar"}},
		{"struct value", "key4", struct{ Name string }{"test"}},
	}

	// Test setting values
	for _, tt := range tests {
		t.Run("set_"+tt.name, func(t *testing.T) {
			cm := NewCacheManager(5)
			defer cm.Close()
			
			cm.Set(tt.key, tt.value, 0) // No TTL
			assert.Equal(t, 1, cm.stats.CurrentSize)
		})
	}

	// Test getting values
	for _, tt := range tests {
		t.Run("get_"+tt.name, func(t *testing.T) {
			cm := NewCacheManager(5)
			defer cm.Close()
			
			cm.Set(tt.key, tt.value, 0) // No TTL
			value, found := cm.Get(tt.key)
			assert.True(t, found)
			assert.Equal(t, tt.value, value)
		})
	}

	// Test non-existent key
	cm := NewCacheManager(5)
	defer cm.Close()
	
	value, found := cm.Get("nonexistent")
	assert.False(t, found)
	assert.Nil(t, value)
}

func TestCacheManager_TTLExpiration(t *testing.T) {
	cm := NewCacheManager(5)
	defer cm.Close()

	// Set value with short TTL
	cm.Set("shortlived", "value", 50*time.Millisecond)
	
	// Should be available immediately
	value, found := cm.Get("shortlived")
	assert.True(t, found)
	assert.Equal(t, "value", value)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	value, found = cm.Get("shortlived")
	assert.False(t, found)
	assert.Nil(t, value)

	// Test value without TTL (should not expire)
	cm.Set("permanent", "value", 0)
	time.Sleep(100 * time.Millisecond)
	
	value, found = cm.Get("permanent")
	assert.True(t, found)
	assert.Equal(t, "value", value)

	// Test value with long TTL
	cm.Set("longlived", "value", 1*time.Hour)
	value, found = cm.Get("longlived")
	assert.True(t, found)
	assert.Equal(t, "value", value)
}

func TestCacheManager_LRUEviction(t *testing.T) {
	cm := NewCacheManager(3) // Small cache for testing eviction
	defer cm.Close()

	// Fill cache to capacity
	cm.Set("key1", "value1", 0)
	cm.Set("key2", "value2", 0)
	cm.Set("key3", "value3", 0)

	// All should be present
	_, found1 := cm.Get("key1")
	_, found2 := cm.Get("key2")
	_, found3 := cm.Get("key3")
	assert.True(t, found1)
	assert.True(t, found2)
	assert.True(t, found3)

	// Add one more item - should evict least recently used (key1)
	cm.Set("key4", "value4", 0)

	// key1 should be evicted
	_, found1 = cm.Get("key1")
	assert.False(t, found1)

	// Others should still be present
	_, found2 = cm.Get("key2")
	_, found3 = cm.Get("key3")
	_, found4 := cm.Get("key4")
	assert.True(t, found2)
	assert.True(t, found3)
	assert.True(t, found4)

	// Access key2 to make it most recently used
	cm.Get("key2")
	
	// Add another item - should evict key3 (now least recently used)
	cm.Set("key5", "value5", 0)

	_, found3 = cm.Get("key3")
	assert.False(t, found3)

	// key2, key4, key5 should remain
	_, found2 = cm.Get("key2")
	_, found4 = cm.Get("key4")
	_, found5 := cm.Get("key5")
	assert.True(t, found2)
	assert.True(t, found4)
	assert.True(t, found5)
}

func TestCacheManager_UpdateExisting(t *testing.T) {
	cm := NewCacheManager(5)
	defer cm.Close()

	// Set initial value
	cm.Set("key1", "initial", 0)
	value, found := cm.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "initial", value)

	// Update value
	cm.Set("key1", "updated", 0)
	value, found = cm.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "updated", value)

	// Should still have only one entry
	assert.Equal(t, 1, cm.stats.CurrentSize)
}

func TestCacheManager_Delete(t *testing.T) {
	cm := NewCacheManager(5)
	defer cm.Close()

	// Set some values
	cm.Set("key1", "value1", 0)
	cm.Set("key2", "value2", 0)

	// Delete existing key
	deleted := cm.Delete("key1")
	assert.True(t, deleted)

	// Verify it's gone
	_, found := cm.Get("key1")
	assert.False(t, found)

	// Other key should remain
	value, found := cm.Get("key2")
	assert.True(t, found)
	assert.Equal(t, "value2", value)

	// Delete non-existent key
	deleted = cm.Delete("nonexistent")
	assert.False(t, deleted)
}

func TestCacheManager_Clear(t *testing.T) {
	cm := NewCacheManager(5)
	defer cm.Close()

	// Add some entries
	cm.Set("key1", "value1", 0)
	cm.Set("key2", "value2", 0)
	cm.Set("key3", "value3", 0)

	// Verify they exist
	assert.Equal(t, 3, cm.stats.CurrentSize)

	// Clear cache
	cm.Clear()

	// Verify all are gone
	assert.Equal(t, 0, cm.stats.CurrentSize)
	assert.Equal(t, 0, len(cm.entries))
	assert.Equal(t, 0, len(cm.lruOrder))

	// Verify specific keys are gone
	_, found1 := cm.Get("key1")
	_, found2 := cm.Get("key2")
	_, found3 := cm.Get("key3")
	assert.False(t, found1)
	assert.False(t, found2)
	assert.False(t, found3)
}

func TestCacheManager_CleanupExpired(t *testing.T) {
	cm := NewCacheManager(5)
	defer cm.Close()

	// Add mix of expired and non-expired entries
	cm.Set("permanent", "value", 0)                          // No TTL
	cm.Set("expired1", "value", 1*time.Millisecond)         // Will expire
	cm.Set("expired2", "value", 1*time.Millisecond)         // Will expire
	cm.Set("longlived", "value", 1*time.Hour)               // Won't expire

	// Wait for short TTL entries to expire
	time.Sleep(10 * time.Millisecond)

	// Manually cleanup
	removed := cm.CleanupExpired()
	assert.Equal(t, 2, removed) // Should remove 2 expired entries

	// Verify expired entries are gone
	_, found1 := cm.Get("expired1")
	_, found2 := cm.Get("expired2")
	assert.False(t, found1)
	assert.False(t, found2)

	// Verify non-expired entries remain
	_, foundPerm := cm.Get("permanent")
	_, foundLong := cm.Get("longlived")
	assert.True(t, foundPerm)
	assert.True(t, foundLong)
}

func TestCacheManager_AccessStatistics(t *testing.T) {
	cm := NewCacheManager(5)
	defer cm.Close()

	// Initially should have zero stats
	stats := cm.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, float64(0), stats.HitRatio)

	// Add an entry
	cm.Set("key1", "value1", 0)

	// Perform some cache operations
	cm.Get("key1")      // Hit
	cm.Get("key1")      // Hit
	cm.Get("missing")   // Miss
	cm.Get("missing2")  // Miss
	cm.Get("key1")      // Hit

	// Check statistics
	stats = cm.GetStats()
	assert.Equal(t, int64(3), stats.Hits)
	assert.Equal(t, int64(2), stats.Misses)
	assert.Equal(t, int64(5), stats.TotalRequests)
	assert.Equal(t, float64(3)/float64(5), stats.HitRatio)
}

func TestCacheManager_ConcurrentAccess(t *testing.T) {
	cm := NewCacheManager(10000) // Large cache to avoid evictions during test
	defer cm.Close()

	numGoroutines := 50
	operationsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines performing cache operations
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("worker_%d_key_%d", workerID, j)
				value := fmt.Sprintf("value_%d_%d", workerID, j)

				// Set value
				cm.Set(key, value, 0)

				// Get value (only if we haven't deleted it)
				if j%10 != 0 {
					retrievedValue, found := cm.Get(key)
					assert.True(t, found, "Key %s should be found", key)
					assert.Equal(t, value, retrievedValue)
				}

				// Occasionally delete
				if j%10 == 0 {
					cm.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is in consistent state
	stats := cm.GetStats()
	assert.Greater(t, stats.TotalRequests, int64(0))
	assert.GreaterOrEqual(t, stats.Hits, int64(0))
	assert.GreaterOrEqual(t, stats.Misses, int64(0))
}

func TestCacheManager_MemoryUsage(t *testing.T) {
	cm := NewCacheManager(10)
	defer cm.Close()

	// Initially should have minimal memory usage
	stats := cm.GetStats()
	assert.Equal(t, int64(0), stats.MemoryUsage)

	// Add some entries
	cm.Set("key1", "small value", 0)
	cm.Set("key2", map[string]interface{}{
		"large_key": "large value with more data",
		"numbers":   []int{1, 2, 3, 4, 5},
	}, 0)

	// Should have non-zero memory usage
	stats = cm.GetStats()
	assert.Greater(t, stats.MemoryUsage, int64(0))

	previousUsage := stats.MemoryUsage

	// Add more entries
	cm.Set("key3", "another value", 0)
	stats = cm.GetStats()
	assert.Greater(t, stats.MemoryUsage, previousUsage)
}

func TestCacheManager_EdgeCases(t *testing.T) {
	t.Run("zero size cache", func(t *testing.T) {
		cm := NewCacheManager(0) // Should default to 100
		defer cm.Close()

		cm.Set("key1", "value1", 0)
		value, found := cm.Get("key1")
		assert.True(t, found)
		assert.Equal(t, "value1", value)
	})

	t.Run("nil values", func(t *testing.T) {
		cm := NewCacheManager(5)
		defer cm.Close()

		cm.Set("nil_key", nil, 0)
		value, found := cm.Get("nil_key")
		assert.True(t, found)
		assert.Nil(t, value)
	})

	t.Run("empty key", func(t *testing.T) {
		cm := NewCacheManager(5)
		defer cm.Close()

		cm.Set("", "empty key value", 0)
		value, found := cm.Get("")
		assert.True(t, found)
		assert.Equal(t, "empty key value", value)
	})

	t.Run("very long key", func(t *testing.T) {
		cm := NewCacheManager(5)
		defer cm.Close()

		longKey := ""
		for i := 0; i < 1000; i++ {
			longKey += "a"
		}

		cm.Set(longKey, "long key value", 0)
		value, found := cm.Get(longKey)
		assert.True(t, found)
		assert.Equal(t, "long key value", value)
	})
}