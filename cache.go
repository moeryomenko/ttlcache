package cache

import (
	"context"
	"errors"
	"time"
)

type EvictionPolicy int

var ErrNotFound = errors.New("key not found")

const (
	// Discards the least recently used items first.
	LRU EvictionPolicy = iota
	// Discards the least frequently used items first.
	LFU
	// Adaptive replacement cache policy.
	ARC
)

// Cache is common interface of cache.
type Cache interface {
	// Set inserts or updates the specified key-value pair with an expiration time.
	Set(key string, value interface{}, expiry time.Duration) error
	// Get returns the value for specified key if it is present in the cache.
	Get(key string) (interface{}, error)
	// Remove removes item from cache by given key.
	Remove(key string) error
}

// NewCache returns cache with selected eviction policy.
func NewCache(ctx context.Context, capacity int, policy EvictionPolicy) Cache {
	switch policy {
	case LRU:
		return newLRUCache(ctx, capacity, true)
	case LFU:
		return newLFUCache(ctx, capacity)
	case ARC:
		return newARCCache(ctx, capacity)
	default:
		panic("Unknown eviction policy")
	}
}
