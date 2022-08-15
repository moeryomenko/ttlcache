package cache

import "github.com/moeryomenko/ttlcache/internal/policies"

// replacementCacher is internal common interface of cache.
type replacementCacher interface {
	// Set inserts or updates the specified key-value pair.
	Set(key string, value any)
	// Get returns the value for specified key if it is present in the cache.
	Get(key string) (any, bool)
	// Remove removes item from cache by given key.
	Remove(key string)
	// Evict evicts given numbers of key from cache by given policy.
	Evict(count int)
	// Len returns current size of cache.
	Len() int
}

// dummy test for policies.
var (
	_ replacementCacher = (*policies.LRUCache)(nil)
	_ replacementCacher = (*policies.LFUCache)(nil)
	_ replacementCacher = (*policies.ARCCache)(nil)
)
