package policies

import (
	"container/list"
)

type LFUCache[K comparable, V any] struct {
	items    map[K]*lfuItem[K, V]
	freqList *list.List
	capacity int
}

type lfuItem[K comparable, V any] struct {
	key         K
	value       V
	freqElement *list.Element
}

type freqEntry[K comparable, V any] struct {
	freq  uint
	items map[*lfuItem[K, V]]struct{}
}

func NewLFUCache[K comparable, V any](capacity int) *LFUCache[K, V] {
	cache := &LFUCache[K, V]{
		items:    make(map[K]*lfuItem[K, V], capacity),
		freqList: list.New(),
		capacity: capacity,
	}

	cache.freqList.PushFront(&freqEntry[K, V]{
		freq:  0,
		items: make(map[*lfuItem[K, V]]struct{}),
	})

	return cache
}

// Set inserts or updates the specified key-value pair with an expiration time.
func (c *LFUCache[K, V]) Set(key K, value V) {
	it, ok := c.items[key]
	if ok {
		item := it
		item.value = value
		return
	}

	item := &lfuItem[K, V]{
		key:         key,
		value:       value,
		freqElement: nil,
	}
	el := c.freqList.Front()
	fe := el.Value.(*freqEntry[K, V])
	fe.items[item] = struct{}{}

	item.freqElement = el
	c.items[key] = item
}

// Get returns the value for specified key if it is present in the cache.
func (c *LFUCache[K, V]) Get(key K) (V, bool) {
	it, ok := c.items[key]
	if !ok {
		var v V
		return v, false
	}

	return it.value, true
}

func (c *LFUCache[K, V]) Remove(key K) {
	if it, ok := c.items[key]; ok {
		c.removeItem(it)
	}
}

func (c *LFUCache[K, V]) Len() int {
	return len(c.items)
}

func (c *LFUCache[K, V]) Evict(count int) {
	entry := c.freqList.Front()
	for i := 0; i < count; {
		if entry == nil {
			return
		}

		for item := range entry.Value.(*freqEntry[K, V]).items {
			if i >= count {
				return
			}

			c.removeItem(item)
			i++
		}
		entry = entry.Next()
	}
}

func (c *LFUCache[K, V]) removeItem(item *lfuItem[K, V]) {
	entry := item.freqElement.Value.(*freqEntry[K, V])
	delete(c.items, item.key)
	delete(entry.items, item)
}
