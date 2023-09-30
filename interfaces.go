package cache

import "github.com/moeryomenko/ttlcache/internal/policies"

// replacementCacher is internal common interface of cache.
type replacementCacher[K comparable, V any] interface {
	// Set inserts or updates the specified key-value pair.
	Set(key K, value V)
	// Get returns the value for specified key if it is present in the cache.
	Get(key K) (V, bool)
	// Remove removes item from cache by given key.
	Remove(key K)
	// Evict evicts given numbers of key from cache by given policy.
	Evict(count int)
	// Len returns current size of cache.
	Len() int
}

// dummy test for policies.
var (
	_ replacementCacher[int, any] = (*policies.LRUCache[int, any])(nil)
	_ replacementCacher[int, any] = (*policies.LFUCache[int, any])(nil)
	_ replacementCacher[int, any] = (*policies.ARCCache[int, any])(nil)
	_ replacementCacher[int, any] = (policies.NoEvictionCache[int, any])(nil)
)
