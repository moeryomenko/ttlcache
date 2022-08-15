package cache

import (
	"container/list"
	"context"
	"time"

	"github.com/moeryomenko/synx"
)

type LFUCache struct {
	items    map[string]*lfuItem
	freqList *list.List
	lock     synx.Spinlock
	capacity int

	epoch  uint64
	ttlMap map[uint64][]string
}

type lfuItem struct {
	key         string
	value       interface{}
	freqElement *list.Element

	// ttl information.
	epoch uint64
	slot  int
}

type freqEntry struct {
	freq  uint
	items map[*lfuItem]struct{}
}

func newLFUCache(ctx context.Context, capacity int) *LFUCache {
	cache := &LFUCache{
		items:    make(map[string]*lfuItem, capacity),
		ttlMap:   make(map[uint64][]string),
		freqList: list.New(),
		capacity: capacity,
	}

	cache.freqList.PushFront(&freqEntry{
		freq:  0,
		items: make(map[*lfuItem]struct{}),
	})

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

// Set inserts or updates the specified key-value pair with an expiration time.
func (c *LFUCache) Set(key string, value interface{}, expiration time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	it, ok := c.items[key]
	if ok {
		item := it
		c.removeFromTTL(item.epoch, item.slot)
		item.value = value
		item.epoch, item.slot = c.emplaceToTTLBucket(key, expiration)
		return nil
	}

	if len(c.items) == c.capacity {
		c.evict(1)
	}

	item := &lfuItem{
		key:         key,
		value:       value,
		freqElement: nil,
	}
	el := c.freqList.Front()
	fe := el.Value.(*freqEntry)
	fe.items[item] = struct{}{}

	item.freqElement = el
	item.epoch, item.slot = c.emplaceToTTLBucket(key, expiration)
	c.items[key] = item
	return nil
}

// Get returns the value for specified key if it is present in the cache.
func (c *LFUCache) Get(key string) (interface{}, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	it, ok := c.items[key]
	if !ok {
		return nil, ErrNotFound
	}

	return it.value, nil
}
func (c *LFUCache) Remove(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	it, ok := c.items[key]
	if !ok {
		return ErrNotFound
	}

	c.removeItem(it)

	return nil
}

func (c *LFUCache) emplaceToTTLBucket(key string, expiration time.Duration) (epoch uint64, slot int) {
	index := uint64(expiration/step) + c.epoch
	if _, ok := c.ttlMap[index]; ok {
		c.ttlMap[index] = append(c.ttlMap[index], key)
		return index, len(c.ttlMap[index]) - 1
	}

	c.ttlMap[index] = []string{key}
	return index, 0
}

func (c *LFUCache) removeFromTTL(epoch uint64, slot int) {
	slots := c.ttlMap[epoch]
	c.ttlMap[epoch] = append(slots[:slot], slots[slot+1:]...)
}

func (c *LFUCache) collectExpired() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.epoch++

	c.removeExpired()
}

func (c *LFUCache) removeExpired() int {
	removeCount := 0
	counter := 0
	for epochBucket := range c.ttlMap {
		if epochBucket > c.epoch {
			continue
		}

		for _, key := range c.ttlMap[epochBucket] {
			if item, ok := c.items[key]; ok {
				removeCount++
				c.removeItem(item)
			}
		}

		delete(c.ttlMap, epochBucket)
		counter++
	}

	return removeCount
}

func (c *LFUCache) evict(count int) {
	removed := c.removeExpired()
	if count <= removed {
		return
	}

	count -= removed

	entry := c.freqList.Front()
	for i := 0; i < count; {
		if entry == nil {
			return
		}

		for item := range entry.Value.(*freqEntry).items {
			if i >= count {
				return
			}

			c.removeItem(item)
			i++
		}
		entry = entry.Next()
	}
}

func (c *LFUCache) removeItem(item *lfuItem) {
	entry := item.freqElement.Value.(*freqEntry)
	delete(c.items, item.key)
	delete(entry.items, item)
}
