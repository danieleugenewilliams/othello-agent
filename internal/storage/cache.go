package storage

import (
	"encoding/json"
	"sync"
	"time"
)

// CacheEntry represents a single cache entry with TTL support
type CacheEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	ExpiresAt  *time.Time  `json:"expires_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	CreatedAt  time.Time   `json:"created_at"`
}

// CacheStats provides cache performance statistics
type CacheStats struct {
	Hits          int64         `json:"hits"`
	Misses        int64         `json:"misses"`
	Evictions     int64         `json:"evictions"`
	CurrentSize   int           `json:"current_size"`
	MaxSize       int           `json:"max_size"`
	HitRatio      float64       `json:"hit_ratio"`
	MemoryUsage   int64         `json:"memory_usage"`
	LastCleanup   time.Time     `json:"last_cleanup"`
	TotalRequests int64         `json:"total_requests"`
}

// CacheManager implements an LRU cache with TTL support and thread safety
type CacheManager struct {
	mu         sync.RWMutex
	entries    map[string]*CacheEntry
	lruOrder   []*CacheEntry    // Most recently used at the end
	maxSize    int
	stats      CacheStats
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewCacheManager creates a new cache manager with specified maximum size
func NewCacheManager(maxSize int) *CacheManager {
	if maxSize <= 0 {
		maxSize = 100 // Default size
	}

	cm := &CacheManager{
		entries:       make(map[string]*CacheEntry),
		lruOrder:      make([]*CacheEntry, 0),
		maxSize:       maxSize,
		stopCleanup:   make(chan bool),
		stats: CacheStats{
			MaxSize:     maxSize,
			LastCleanup: time.Now(),
		},
	}

	// Start background cleanup routine for expired entries
	cm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go cm.cleanupRoutine()

	return cm
}

// Set stores a value in the cache with optional TTL
func (cm *CacheManager) Set(key string, value interface{}, ttl time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	var expiresAt *time.Time
	if ttl > 0 {
		expTime := now.Add(ttl)
		expiresAt = &expTime
	}

	// Check if key already exists
	if existingEntry, exists := cm.entries[key]; exists {
		// Update existing entry
		existingEntry.Value = value
		existingEntry.ExpiresAt = expiresAt
		existingEntry.AccessedAt = now
		cm.moveToEnd(existingEntry)
		return
	}

	// Create new entry
	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		ExpiresAt:  expiresAt,
		AccessedAt: now,
		CreatedAt:  now,
	}

	// Add to maps and LRU order
	cm.entries[key] = entry
	cm.lruOrder = append(cm.lruOrder, entry)
	cm.stats.CurrentSize++

	// Evict least recently used entries if over capacity
	cm.evictIfNecessary()
}

// Get retrieves a value from the cache
func (cm *CacheManager) Get(key string) (interface{}, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats.TotalRequests++

	entry, exists := cm.entries[key]
	if !exists {
		cm.stats.Misses++
		cm.updateHitRatio()
		return nil, false
	}

	// Check if entry has expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		cm.deleteEntry(key)
		cm.stats.Misses++
		cm.updateHitRatio()
		return nil, false
	}

	// Update access time and move to end (most recently used)
	entry.AccessedAt = time.Now()
	cm.moveToEnd(entry)
	
	cm.stats.Hits++
	cm.updateHitRatio()
	return entry.Value, true
}

// Delete removes a key from the cache
func (cm *CacheManager) Delete(key string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.entries[key]; exists {
		cm.deleteEntry(key)
		return true
	}
	return false
}

// Clear removes all entries from the cache
func (cm *CacheManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.entries = make(map[string]*CacheEntry)
	cm.lruOrder = make([]*CacheEntry, 0)
	cm.stats.CurrentSize = 0
	cm.stats.Evictions = 0
	cm.stats.Hits = 0
	cm.stats.Misses = 0
	cm.stats.TotalRequests = 0
	cm.updateHitRatio()
}

// CleanupExpired manually removes all expired entries
func (cm *CacheManager) CleanupExpired() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.cleanupExpiredEntries()
}

// GetStats returns current cache statistics
func (cm *CacheManager) GetStats() CacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Update memory usage estimate
	cm.stats.MemoryUsage = cm.estimateMemoryUsage()
	return cm.stats
}

// Close stops the background cleanup routine
func (cm *CacheManager) Close() {
	close(cm.stopCleanup)
	if cm.cleanupTicker != nil {
		cm.cleanupTicker.Stop()
	}
}

// Internal helper methods

// evictIfNecessary removes least recently used entries if cache is over capacity
func (cm *CacheManager) evictIfNecessary() {
	for len(cm.lruOrder) > cm.maxSize {
		// Remove least recently used (first in slice)
		lru := cm.lruOrder[0]
		cm.deleteEntry(lru.Key)
		cm.stats.Evictions++
	}
}

// deleteEntry removes an entry from both maps and LRU order
func (cm *CacheManager) deleteEntry(key string) {
	entry, exists := cm.entries[key]
	if !exists {
		return
	}

	// Remove from entries map
	delete(cm.entries, key)
	cm.stats.CurrentSize--

	// Remove from LRU order
	for i, e := range cm.lruOrder {
		if e == entry {
			cm.lruOrder = append(cm.lruOrder[:i], cm.lruOrder[i+1:]...)
			break
		}
	}
}

// moveToEnd moves an entry to the end of LRU order (most recently used)
func (cm *CacheManager) moveToEnd(entry *CacheEntry) {
	// Find and remove entry from current position
	for i, e := range cm.lruOrder {
		if e == entry {
			cm.lruOrder = append(cm.lruOrder[:i], cm.lruOrder[i+1:]...)
			break
		}
	}
	
	// Add to end
	cm.lruOrder = append(cm.lruOrder, entry)
}

// cleanupExpiredEntries removes all expired entries
func (cm *CacheManager) cleanupExpiredEntries() int {
	now := time.Now()
	expiredKeys := make([]string, 0)

	// Find expired entries
	for key, entry := range cm.entries {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Remove expired entries
	for _, key := range expiredKeys {
		cm.deleteEntry(key)
	}

	cm.stats.LastCleanup = now
	return len(expiredKeys)
}

// updateHitRatio calculates and updates the cache hit ratio
func (cm *CacheManager) updateHitRatio() {
	if cm.stats.TotalRequests > 0 {
		cm.stats.HitRatio = float64(cm.stats.Hits) / float64(cm.stats.TotalRequests)
	} else {
		cm.stats.HitRatio = 0.0
	}
}

// estimateMemoryUsage provides a rough estimate of memory usage
func (cm *CacheManager) estimateMemoryUsage() int64 {
	var totalSize int64
	
	for _, entry := range cm.entries {
		// Estimate size of key
		totalSize += int64(len(entry.Key))
		
		// Estimate size of value using JSON marshaling
		if valueBytes, err := json.Marshal(entry.Value); err == nil {
			totalSize += int64(len(valueBytes))
		}
		
		// Add overhead for entry metadata (approximate)
		totalSize += 100 // Time fields, pointers, etc.
	}
	
	return totalSize
}

// cleanupRoutine runs in background to periodically clean up expired entries
func (cm *CacheManager) cleanupRoutine() {
	for {
		select {
		case <-cm.cleanupTicker.C:
			cm.mu.Lock()
			cm.cleanupExpiredEntries()
			cm.mu.Unlock()
		case <-cm.stopCleanup:
			return
		}
	}
}