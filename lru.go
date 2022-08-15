package cache

import (
	"container/list"
	"context"
	"time"

	"github.com/moeryomenko/synx"
)

const step = 1 * time.Second

type LRUCache struct {
	items     map[string]*list.Element
	evictList *list.List
	capacity  int
	isSafe    bool
	lock      synx.Spinlock

	epoch  uint64
	ttlMap map[uint64][]string
}

func newLRUCache(ctx context.Context, capacity int, isSafe bool) *LRUCache {
	cache := &LRUCache{
		items:     make(map[string]*list.Element),
		ttlMap:    make(map[uint64][]string),
		evictList: list.New(),
		capacity:  capacity,
		isSafe:    isSafe,
	}

	go func() {
		ttlTicker := time.NewTicker(step)
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

type lruItem struct {
	key   string
	value any

	epoch uint64
	slot  int
}

// Set inserts or updates the specified key-value pair with an expiration time.
func (c *LRUCache) Set(key string, value any, expiration time.Duration) error {
	if c.isSafe {
		c.lock.Lock()
		defer c.lock.Unlock()
	}

	// Check for existing item
	var item *lruItem
	if it, ok := c.items[key]; ok {
		c.evictList.MoveToFront(it)
		item = it.Value.(*lruItem)
		item.value = value
		c.removeFromTTL(item.epoch, item.slot)
		item.epoch, item.slot = c.emplaceToTTLBucket(key, expiration)
		return nil
	}

	// Verify size not exceeded
	if c.evictList.Len() >= c.capacity {
		c.evict(1)
	}

	item = &lruItem{
		key:   key,
		value: value,
	}
	c.items[key] = c.evictList.PushFront(item)
	item.epoch, item.slot = c.emplaceToTTLBucket(key, expiration)

	return nil
}

func (c *LRUCache) emplaceToTTLBucket(key string, expiration time.Duration) (epoch uint64, slot int) {
	index := uint64(expiration/step) + c.epoch
	if _, ok := c.ttlMap[index]; ok {
		c.ttlMap[index] = append(c.ttlMap[index], key)
		return index, len(c.ttlMap[index]) - 1
	}

	c.ttlMap[index] = []string{key}
	return index, 0
}

func (c *LRUCache) removeFromTTL(epoch uint64, slot int) {
	slots := c.ttlMap[epoch]
	c.ttlMap[epoch] = append(slots[:slot], slots[slot+1:]...)
}

// Get returns the value for specified key if it is present in the cache.
func (c *LRUCache) Get(key string) (any, error) {
	if c.isSafe {
		c.lock.Lock()
		defer c.lock.Unlock()
	}
	item, ok := c.items[key]
	if !ok {
		return nil, ErrNotFound
	}
	it := item.Value.(*lruItem)
	c.evictList.MoveToFront(item)

	return it.value, nil
}

func (c *LRUCache) Len() int {
	return c.evictList.Len()
}

func (c *LRUCache) Remove(key string) error {
	if c.isSafe {
		c.lock.Lock()
		defer c.lock.Unlock()
	}

	item, ok := c.items[key]
	if !ok {
		return ErrNotFound
	}

	c.removeElement(item)
	return nil
}

func (c *LRUCache) collectExpired() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.epoch++

	c.removeExpired()
}

func (c *LRUCache) removeExpired() int {
	removeCount := 0
	counter := 0
	for epochBucket := range c.ttlMap {
		if epochBucket > c.epoch {
			continue
		}

		for _, key := range c.ttlMap[epochBucket] {
			if item, ok := c.items[key]; ok {
				removeCount++
				c.removeElement(item)
			}
		}

		delete(c.ttlMap, epochBucket)
		counter++
	}

	return removeCount
}

func (c *LRUCache) evict(count int) {
	removed := c.removeExpired()
	if count <= removed {
		return
	}

	count -= removed

	for i := 0; i < count; i++ {
		ent := c.evictList.Back()
		if ent == nil {
			return
		}

		c.removeElement(ent)
	}
}

func (c *LRUCache) removeElement(e *list.Element) {
	entry := c.evictList.Remove(e).(*lruItem)
	delete(c.items, entry.key)
}
