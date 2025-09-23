package main

import (
	"sync"
	"time"
)

// DedupeCache manages recently decoded barcodes to prevent duplicates
type DedupeCache struct {
	cache    map[string]*CacheEntry
	mutex    sync.RWMutex
	cooldown time.Duration
}

// CacheEntry represents a cached decode result
type CacheEntry struct {
	Result    DecodeResult
	Timestamp time.Time
}

// NewDedupeCache creates a new dedupe cache with specified cooldown
func NewDedupeCache(cooldown time.Duration) *DedupeCache {
	if cooldown <= 0 {
		cooldown = 2 * time.Second // Default 2 second cooldown
	}

	return &DedupeCache{
		cache:    make(map[string]*CacheEntry),
		cooldown: cooldown,
	}
}

// IsDuplicate checks if a decode result is a recent duplicate
func (dc *DedupeCache) IsDuplicate(result DecodeResult) bool {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	key := dc.generateKey(result)
	entry, exists := dc.cache[key]

	if !exists {
		return false
	}

	// Check if cooldown period has passed
	return time.Since(entry.Timestamp) < dc.cooldown
}

// Add stores a decode result in the cache
func (dc *DedupeCache) Add(result DecodeResult) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	key := dc.generateKey(result)
	dc.cache[key] = &CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
	}
}

// Cleanup removes expired entries from the cache
func (dc *DedupeCache) Cleanup() {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	now := time.Now()
	for key, entry := range dc.cache {
		if now.Sub(entry.Timestamp) > dc.cooldown*2 { // Keep entries for 2x cooldown
			delete(dc.cache, key)
		}
	}
}

// generateKey creates a unique key for a decode result
func (dc *DedupeCache) generateKey(result DecodeResult) string {
	return result.Text + "|" + result.Format
}

// GetStats returns cache statistics
func (dc *DedupeCache) GetStats() map[string]interface{} {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	return map[string]interface{}{
		"cacheSize":    len(dc.cache),
		"cooldownMs":   dc.cooldown.Milliseconds(),
	}
}

// Clear removes all entries from the cache
func (dc *DedupeCache) Clear() {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	dc.cache = make(map[string]*CacheEntry)
}