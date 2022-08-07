package cache

import (
	"container/list"
	"time"

	"github.com/moeryomenko/synx"
)

type LRUCache struct {
	items     map[string]any
	evictList *list.List
	capacity  int
	lock      synx.Spinlock
}

func newLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		items:     make(map[string]any),
		evictList: list.New(),
		capacity:  capacity,
	}
}

type lruItem struct {
	key        string
	value      any
	expiration time.Time
}

// Set inserts or updates the specified key-value pair with an expiration time.
func (c *LRUCache) Set(key string, value any, expiration time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	// Check for existing item
	var item *lruItem
	if it, ok := c.items[key]; ok {
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
	c.items[key] = c.evictList.PushFront(item)

	return nil
}

// Get returns the value for specified key if it is present in the cache.
func (c *LRUCache) Get(key string) (any, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	ent, ok := c.items[key]
	if !ok {
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

func (c *LRUCache) Len() int {
	return c.evictList.Len()
}

func (c *LRUCache) Remove(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	ent, ok := c.items[key]
	if !ok {
		return ErrNotFound
	}

	item := ent.(*list.Element)
	c.removeElement(item)
	return nil
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
	delete(c.items, entry.key)
}
