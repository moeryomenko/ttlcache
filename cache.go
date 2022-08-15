package cache

import (
	"context"
	"math"
	"time"

	"github.com/moeryomenko/synx"

	"github.com/moeryomenko/ttlcache/internal/policies"
)

type evictionPolicy int

const (
	// Discards the least recently used items first.
	LRU evictionPolicy = iota
	// Discards the least frequently used items first.
	LFU
	// Adaptive replacement cache policy.
	ARC
)

const defaultEpochGranularity = 1 * time.Second

type Cache struct {
	cache    replacementCacher
	capacity int

	lock        synx.Spinlock
	epoch       uint64
	granularity time.Duration
	ttlMap      map[uint64][]string
}

type config struct {
	policy      evictionPolicy
	granularity time.Duration
}

type Option func(*config)

func WithEvictionPolicy(policy evictionPolicy) Option {
	return func(c *config) {
		c.policy = policy
	}
}

func WithTTLEpochGranularity(period time.Duration) Option {
	return func(c *config) {
		c.granularity = period
	}
}

type entry struct {
	value any

	epoch uint64
	slot  int
}

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

var (
	_ replacementCacher = (*policies.LRUCache)(nil)
	_ replacementCacher = (*policies.LFUCache)(nil)
	_ replacementCacher = (*policies.ARCCache)(nil)
)

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

func (c *Cache) Get(key string) (any, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, ok := c.cache.Get(key)
	if ok {
		return item.(entry).value, ok
	}
	return nil, ok
}

func (c *Cache) Remove(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache.Remove(key)
}

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
	defer c.lock.Unlock()

	c.epoch++

	c.removeExpired()
}

func (c *Cache) removeExpired() int {
	removeCount := 0

	for epochBucket := range c.ttlMap {
		if epochBucket > c.epoch {
			continue
		}

		for _, key := range c.ttlMap[epochBucket] {
			c.cache.Remove(key)
		}

		delete(c.ttlMap, epochBucket)
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
