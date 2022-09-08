package cache

import (
	"context"
	"math"
	"time"

	"github.com/moeryomenko/synx"

	"github.com/moeryomenko/ttlcache/internal/policies"
)

// Cache is cache with TTL and eviction over capacity.
type Cache struct {
	cache    replacementCacher
	capacity int

	lock        synx.Spinlock
	epoch       uint64
	granularity time.Duration
	ttlMap      map[uint64][]string
}

// NewCache returns cache with selected eviction policy.
func NewCache(ctx context.Context, capacity int, opts ...Option) *Cache {
	cfg := config{
		policy:      LRU,
		granularity: defaultEpochGranularity,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	cache := &Cache{
		capacity:    capacity,
		granularity: cfg.granularity,
		ttlMap:      make(map[uint64][]string),
	}
	switch cfg.policy {
	case LRU:
		cache.cache = policies.NewLRUCache(capacity)
	case LFU:
		cache.cache = policies.NewLFUCache(capacity)
	case ARC:
		cache.cache = policies.NewARCCache(capacity)
	case NOOP:
		cache.cache = policies.NewNoEvictionCache(capacity)
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
func (c *Cache) Set(key string, value any) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// NOTE: set max epoch value, prevent eviction by ttl, but can be
	// evicted by replacement policy.
	c.cache.Set(key, entry{value: value, epoch: math.MaxUint64})

	if c.cache.Len() > c.capacity {
		c.evict(1)
	}
}

// SetNX sets new or updates key-value pair with given expiration time.
func (c *Cache) SetNX(key string, value any, expiry time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()

	ent := entry{value: value}

	item, ok := c.cache.Get(key)
	if ok {
		ent := item.(entry)
		c.removeFromTTL(ent.epoch, ent.slot)
	}

	ent.epoch, ent.slot = c.emplaceToTTLBucket(key, expiry)
	c.cache.Set(key, ent)

	if c.cache.Len() > c.capacity {
		c.evict(1)
	}
}

// Get returns value by given key.
func (c *Cache) Get(key string) (any, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, ok := c.cache.Get(key)
	if ok {
		return item.(entry).value, ok
	}
	return nil, ok
}

// Remove removes cache entry by given key.
func (c *Cache) Remove(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache.Remove(key)
}

// Len returns current size of cache.
func (c *Cache) Len() int {
	return c.cache.Len()
}

func (c *Cache) emplaceToTTLBucket(key string, expiration time.Duration) (epoch uint64, slot int) {
	index := uint64(expiration/c.granularity) + c.epoch
	if _, ok := c.ttlMap[index]; ok {
		c.ttlMap[index] = append(c.ttlMap[index], key)
		return index, len(c.ttlMap[index]) - 1
	}

	c.ttlMap[index] = []string{key}
	return index, 0
}

func (c *Cache) removeFromTTL(epoch uint64, slot int) {
	slots := c.ttlMap[epoch]
	c.ttlMap[epoch] = append(slots[:slot], slots[slot+1:]...)
}

func (c *Cache) collectExpired() {
	c.lock.Lock()
	defer func() {
		c.epoch++
		c.lock.Unlock()
	}()

	c.removeExpired()
}

func (c *Cache) removeExpired() int {
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

func (c *Cache) evict(count int) {
	removed := c.removeExpired()
	if count <= removed {
		return
	}

	count -= removed

	c.cache.Evict(count)
}

type entry struct {
	value any

	epoch uint64
	slot  int
}
