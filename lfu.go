package cache

import (
	"container/list"
	"time"

	"github.com/moeryomenko/synx"
)

type LFUCache struct {
	items    map[string]*lfuItem
	freqList *list.List
	lock     synx.Spinlock
	capacity int
}

type lfuItem struct {
	key         string
	value       interface{}
	freqElement *list.Element
	expiration  time.Time
}

type freqEntry struct {
	freq  uint
	items map[*lfuItem]struct{}
}

func newLFUCache(capacity int) *LFUCache {
	cache := &LFUCache{
		items:    make(map[string]*lfuItem, capacity),
		freqList: list.New(),
		capacity: capacity,
	}

	cache.freqList.PushFront(&freqEntry{
		freq:  0,
		items: make(map[*lfuItem]struct{}),
	})

	return cache
}

// Set inserts or updates the specified key-value pair with an expiration time.
func (c *LFUCache) Set(key string, value interface{}, expiration time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	it, ok := c.items[key]
	if ok {
		item := it
		item.value = value
		item.expiration = time.Now().Add(expiration)
		return nil
	}

	if len(c.items) == c.capacity {
		c.evict(1)
	}

	item := &lfuItem{
		key:         key,
		value:       value,
		expiration:  time.Now().Add(expiration),
		freqElement: nil,
	}
	el := c.freqList.Front()
	fe := el.Value.(*freqEntry)
	fe.items[item] = struct{}{}

	item.freqElement = el
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

	if it.expiration.Before(time.Now()) {
		c.removeItem(it)
		return nil, ErrNotFound
	}

	return it.value, nil
}

func (c *LFUCache) evict(count int) {
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
