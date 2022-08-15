package cache

import (
	"context"
	"time"

	"github.com/moeryomenko/synx"
)

// ARCCache is improved LRU cache, that tracks both recency and frequency of use.
// See: https://ieeexplore.ieee.org/document/1297303.
type ARCCache struct {
	// t1 is lru for recently accessed items.
	t1 *LRUCache
	// b1 is lru for eviction from t1.
	b1 *LRUCache
	// t2 is lru for frequently accessed times.
	t2 *LRUCache
	// b2 is lru for evicted from t2.
	b2 *LRUCache

	capacity int
	prefer   int
	lock     synx.Spinlock
}

func newARCCache(ctx context.Context, capacity int) *ARCCache {
	return &ARCCache{
		capacity: capacity,
		t1:       newLRUCache(ctx, capacity, false),
		b1:       newLRUCache(ctx, capacity, false),
		t2:       newLRUCache(ctx, capacity, false),
		b2:       newLRUCache(ctx, capacity, false),
	}
}

func (c *ARCCache) Set(key string, value any, expiration time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if contains(c.t1, key) {
		_ = c.t1.Remove(key)
		return c.t2.Set(key, value, expiration)
	}

	if contains(c.t2, key) {
		return c.t2.Set(key, value, expiration)
	}

	if contains(c.b1, key) {
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()

		if b2Len > b1Len {
			delta = b2Len / b1Len
		}

		if c.prefer+delta >= c.capacity {
			c.prefer = c.capacity
		} else {
			c.prefer += delta
		}

		if c.t1.Len()+c.t2.Len() >= c.capacity {
			c.replcae(true, expiration)
		}

		c.b2.Remove(key)

		return c.t2.Set(key, value, expiration)
	}

	if c.t1.Len()+c.t2.Len() >= c.capacity {
		c.replcae(false, expiration)
	}

	if c.b1.Len() > c.capacity-c.prefer {
		removeOldest(c.b1)
	}

	if c.b2.Len() > c.prefer {
		removeOldest(c.b2)
	}

	return c.t1.Set(key, value, expiration)
}

func (c *ARCCache) Get(key string) (any, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if val, err := c.t1.Get(key); err != ErrNotFound {
		return val, nil
	}

	return c.t2.Get(key)
}

func (c *ARCCache) Remove(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// currently ignore errors.
	c.t1.Remove(key)
	c.t2.Remove(key)
	c.b1.Remove(key)
	c.b2.Remove(key)

	return nil
}

func (c *ARCCache) replcae(direction bool, expiration time.Duration) {
	t1Len := c.t1.Len()
	if t1Len > 0 && (t1Len > c.prefer || (t1Len == c.prefer && direction)) {
		k, ok := removeOldest(c.t1)
		if ok {
			c.b1.Set(k, nil, expiration)
		}
	} else {
		k, ok := removeOldest(c.t2)
		if ok {
			c.b2.Set(k, nil, expiration)
		}
	}
}

func removeOldest(cache *LRUCache) (string, bool) {
	ent := cache.evictList.Back()
	if ent != nil {
		cache.removeElement(ent)
		return ent.Value.(*lruItem).key, true
	}
	return "", false
}

func contains(cache Cache, key string) bool {
	_, err := cache.Get(key)
	return err != ErrNotFound
}
