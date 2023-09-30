package cache

import (
	"context"
	"math"
	"time"

	"github.com/moeryomenko/synx"

	"github.com/moeryomenko/ttlcache/internal/policies"
)

// Cache is cache with TTL and eviction over capacity.
type Cache[K comparable, V any] struct {
	cache    replacementCacher[K, entry[V]]
	capacity int

	lock        synx.Spinlock
	epoch       uint64
	granularity time.Duration
	ttlMap      map[uint64][]K
}

// NewCache returns cache with selected eviction policy.
func NewCache[K comparable, V any](ctx context.Context, capacity int, opts ...Option) *Cache[K, V] {
	cfg := config{
		policy:      LRU,
		granularity: defaultEpochGranularity,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	cache := &Cache[K, V]{
		capacity:    capacity,
		granularity: cfg.granularity,
		ttlMap:      make(map[uint64][]K),
	}
	switch cfg.policy {
	case LRU:
		cache.cache = policies.NewLRUCache[K, entry[V]](capacity)
	case LFU:
		cache.cache = policies.NewLFUCache[K, entry[V]](capacity)
	case ARC:
		cache.cache = policies.NewARCCache[K, entry[V]](capacity)
	case NOOP:
		cache.cache = policies.NewNoEvictionCache[K, entry[V]](capacity)
	default:
		panic("Unknown eviction policy")
	}

	go func() {
		ttlTicker := time.NewTicker(cache.granularity)
		defer ttlTicker.Stop()

		for {
			select {
			case <-ttlTicker.C:
				cache.collectExpired()
			case <-ctx.Done():
				return
			}
		}
	}()

	return cache
}

// Set sets new or updates key-value pair to cache, which can be evicted only by policy.
func (c *Cache[K, V]) Set(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// NOTE: set max epoch value, prevent eviction by ttl, but can be
	// evicted by replacement policy.
	c.cache.Set(key, entry[V]{value: value, epoch: math.MaxUint64})

	if c.cache.Len() > c.capacity {
		c.evict(1)
	}
}

// SetNX sets new or updates key-value pair with given expiration time.
func (c *Cache[K, V]) SetNX(key K, value V, expiry time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()

    if item, ok := c.cache.Get(key); ok {
		c.removeFromTTL(item.epoch, item.slot)
	}

    epoch, slot := c.emplaceToTTLBucket(key, expiry)
	c.cache.Set(key, entry[V]{value: value, epoch: epoch, slot: slot})

	if c.cache.Len() > c.capacity {
		c.evict(1)
	}
}

// Get returns value by given key.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, ok := c.cache.Get(key)
	if ok {
		return item.value, ok
	}
        var v V
	return v, ok
}

// Remove removes cache entry by given key.
func (c *Cache[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache.Remove(key)
}

// Len returns current size of cache.
func (c *Cache[K, V]) Len() int {
	return c.cache.Len()
}

func (c *Cache[K, V]) emplaceToTTLBucket(key K, expiration time.Duration) (epoch uint64, slot int) {
	index := uint64(expiration/c.granularity) + c.epoch
	if _, ok := c.ttlMap[index]; ok {
		c.ttlMap[index] = append(c.ttlMap[index], key)
		return index, len(c.ttlMap[index]) - 1
	}

	c.ttlMap[index] = []K{key}
	return index, 0
}

func (c *Cache[K, V]) removeFromTTL(epoch uint64, slot int) {
	slots := c.ttlMap[epoch]
	c.ttlMap[epoch] = append(slots[:slot], slots[slot+1:]...)
}

func (c *Cache[K, V]) collectExpired() {
	c.lock.Lock()
	defer func() {
		c.epoch++
		c.lock.Unlock()
	}()

	c.removeExpired()
}

func (c *Cache[K, V]) removeExpired() int {
	removeCount := 0

	for epochCounter := c.epoch; epochCounter >= 0; epochCounter-- {
		epochBucket, ok := c.ttlMap[epochCounter]
		if !ok {
			return removeCount
		}
		for _, key := range epochBucket {
			c.cache.Remove(key)
		}

		delete(c.ttlMap, epochCounter)
	}

	return removeCount
}

func (c *Cache[K, V]) evict(count int) {
	removed := c.removeExpired()
	if count <= removed {
		return
	}

	count -= removed

	c.cache.Evict(count)
}

type entry[V any] struct {
	value V

	epoch uint64
	slot  int
}
