package cache

import (
	"container/list"
	"time"

	"github.com/moeryomenko/synx"
)

type LRUCache struct {
	items     *synx.Map
	evictList *list.List
	capacity  int
	lock      synx.Spinlock
}

func newLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		items:     synx.New(capacity),
		evictList: list.New(),
		capacity:  capacity,
	}
}

type lruItem struct {
	key        string
	value      interface{}
	expiration time.Time
}

// Set inserts or updates the specified key-value pair with an expiration time.
func (c *LRUCache) Set(key string, value interface{}, expiration time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	// Check for existing item
	var item *lruItem
	if it, err := c.items.Get(key); err == nil {
		element, _ := it.(*list.Element)
		c.evictList.MoveToFront(element)
		item = element.Value.(*lruItem)
		item.value = value
		item.expiration = time.Now().Add(expiration)
		return nil
	}

	// Verify size not exceeded
	if c.evictList.Len() >= c.capacity {
		c.evict(1)
	}

	item = &lruItem{
		key:        key,
		value:      value,
		expiration: time.Now().Add(expiration),
	}
	_ = c.items.Set(key, c.evictList.PushFront(item))

	return nil
}

// Get returns the value for specified key if it is present in the cache.
func (c *LRUCache) Get(key string) (interface{}, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	ent, err := c.items.Get(key)
	if err != nil {
		return nil, ErrNotFound
	}
	item := ent.(*list.Element)
	it := item.Value.(*lruItem)
	if it.expiration.Before(time.Now()) {
		c.removeElement(item)
		return nil, ErrNotFound
	}
	c.evictList.MoveToFront(item)

	return it.value, nil
}

func (c *LRUCache) evict(count int) {
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
	c.items.Del(entry.key)
}
