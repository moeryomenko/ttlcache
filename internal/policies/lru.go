package policies

import "container/list"

type LRUCache[K comparable, V any] struct {
	items     map[K]*list.Element
	evictList *list.List
	capacity  int
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		items:     make(map[K]*list.Element),
		evictList: list.New(),
		capacity:  capacity,
	}
}

type lruItem[K comparable, V any] struct {
	key  K
	value V
}

// Set inserts or updates the specified key-value pair with an expiration time.
func (c *LRUCache[K, V]) Set(key K, value V) {
	// Check for existing item
	var item *lruItem[K, V]
	if it, ok := c.items[key]; ok {
		c.evictList.MoveToFront(it)
		item = it.Value.(*lruItem[K, V])
		item.value = value
		return
	}

	// Verify size not exceeded
	if c.evictList.Len() >= c.capacity {
		c.Evict(1)
	}

	item = &lruItem[K, V]{
		key:   key,
		value: value,
	}
	c.items[key] = c.evictList.PushFront(item)
}

// Get returns the value for specified key if it is present in the cache.
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	item, ok := c.items[key]
	if !ok {
		var v V
		return v, false
	}
	it := item.Value.(*lruItem[K,V])
	c.evictList.MoveToFront(item)

	return it.value, true
}

func (c *LRUCache[K, V]) Len() int {
	return len(c.items)
}

func (c *LRUCache[K, V]) Remove(key K) {
	if item, ok := c.items[key]; ok {
		c.removeElement(item)
	}
}

func (c *LRUCache[K, V]) Evict(count int) {
	for i := 0; i < count; i++ {
		ent := c.evictList.Back()
		if ent == nil {
			return
		}

		c.removeElement(ent)
	}
}

func (c *LRUCache[K, V]) removeElement(e *list.Element) {
	entry := c.evictList.Remove(e).(*lruItem[K,V])
	delete(c.items, entry.key)
}
