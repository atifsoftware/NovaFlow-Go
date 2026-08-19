package core

import (
	"sync"
	"time"
)

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

func (item cacheItem) isExpired() bool {
	if item.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(item.expiresAt)
}

// Cache provides a thread-safe in-memory key-value store with TTL support.
type Cache struct {
	mu     sync.RWMutex
	items  map[string]cacheItem
	stopCh chan struct{}
}

// NewCache creates an in-memory Cache and starts a background cleaner for expired keys.
func NewCache(cleanupInterval time.Duration) *Cache {
	c := &Cache{
		items:  make(map[string]cacheItem),
		stopCh: make(chan struct{}),
	}

	if cleanupInterval > 0 {
		go c.startCleanup(cleanupInterval)
	}

	return c
}

func (c *Cache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.purgeExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache) purgeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, item := range c.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(c.items, k)
		}
	}
}

// Set stores a key-value pair with a specified TTL. Use 0 for no expiration.
func (c *Cache) Set(key string, val interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.items[key] = cacheItem{
		value:     val,
		expiresAt: expiresAt,
	}
}

// Get retrieves a value by key. Returns false if not found or expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || item.isExpired() {
		if ok && item.isExpired() {
			c.Delete(key)
		}
		return nil, false
	}

	return item.value, true
}

// Remember fetches a cached item, or computes it via callback and caches the result.
func (c *Cache) Remember(key string, ttl time.Duration, fallback func() (interface{}, error)) (interface{}, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	val, err := fallback()
	if err != nil {
		return nil, err
	}

	c.Set(key, val, ttl)
	return val, nil
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Flush removes all items from the cache.
func (c *Cache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheItem)
}

// Close stops the background cleanup worker.
func (c *Cache) Close() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}
