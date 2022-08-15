package policies

import (
	"container/list"
)

type LFUCache struct {
	items    map[string]*lfuItem
	freqList *list.List
	capacity int
}

type lfuItem struct {
	key         string
	value       any
	freqElement *list.Element
}

type freqEntry struct {
	freq  uint
	items map[*lfuItem]struct{}
}

func NewLFUCache(capacity int) *LFUCache {
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
func (c *LFUCache) Set(key string, value any) {
	it, ok := c.items[key]
	if ok {
		item := it
		item.value = value
		return
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
	c.items[key] = item
}

// Get returns the value for specified key if it is present in the cache.
func (c *LFUCache) Get(key string) (any, bool) {
	it, ok := c.items[key]
	if !ok {
		return nil, false
	}

	return it.value, true
}

func (c *LFUCache) Remove(key string) {
	if it, ok := c.items[key]; ok {
		c.removeItem(it)
	}
}

func (c *LFUCache) Len() int {
	return len(c.items)
}

func (c *LFUCache) Evict(count int) {
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
