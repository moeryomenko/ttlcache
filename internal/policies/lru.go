package policies

import "container/list"

type LRUCache struct {
	items     map[string]*list.Element
	evictList *list.List
	capacity  int
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		items:     make(map[string]*list.Element),
		evictList: list.New(),
		capacity:  capacity,
	}
}

type lruItem struct {
	key   string
	value any
}

// Set inserts or updates the specified key-value pair with an expiration time.
func (c *LRUCache) Set(key string, value any) {
	// Check for existing item
	var item *lruItem
	if it, ok := c.items[key]; ok {
		c.evictList.MoveToFront(it)
		item = it.Value.(*lruItem)
		item.value = value
		return
	}

	// Verify size not exceeded
	if c.evictList.Len() >= c.capacity {
		c.Evict(1)
	}

	item = &lruItem{
		key:   key,
		value: value,
	}
	c.items[key] = c.evictList.PushFront(item)
}

// Get returns the value for specified key if it is present in the cache.
func (c *LRUCache) Get(key string) (any, bool) {
	item, ok := c.items[key]
	if !ok {
		return nil, false
	}
	it := item.Value.(*lruItem)
	c.evictList.MoveToFront(item)

	return it.value, true
}

func (c *LRUCache) Len() int {
	return len(c.items)
}

func (c *LRUCache) Remove(key string) {
	if item, ok := c.items[key]; ok {
		c.removeElement(item)
	}
}

func (c *LRUCache) Evict(count int) {
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
