package policies

// ARCCache is improved LRU cache, that tracks both recency and frequency of use.
// See: https://ieeexplore.ieee.org/document/1297303.
type ARCCache[K comparable, V any] struct {
	// t1 is lru for recently accessed items.
	t1 *LRUCache[K, V]
	// b1 is lru for eviction from t1.
	b1 *LRUCache[K, V]
	// t2 is lru for frequently accessed times.
	t2 *LRUCache[K, V]
	// b2 is lru for evicted from t2.
	b2 *LRUCache[K, V]

	capacity int
	prefer   int
}

func NewARCCache[K comparable, V any](capacity int) *ARCCache [K, V]{
	return &ARCCache[K, V]{
		capacity: capacity,
		t1:       NewLRUCache[K, V](capacity),
		b1:       NewLRUCache[K, V](capacity),
		t2:       NewLRUCache[K, V](capacity),
		b2:       NewLRUCache[K, V](capacity),
	}
}

func (c *ARCCache[K, V]) Set(key K, value V) {
	if contains(c.t1, key) {
		c.t1.Remove(key)
		c.t2.Set(key, value)
		return
	}

	if contains(c.t2, key) {
		c.t2.Set(key, value)
		return
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
			c.replcae(true)
		}

		c.b2.Remove(key)

		c.t2.Set(key, value)
		return
	}

	if c.t1.Len()+c.t2.Len() >= c.capacity {
		c.replcae(false)
	}

	if c.b1.Len() > c.capacity-c.prefer {
		removeOldest(c.b1)
	}

	if c.b2.Len() > c.prefer {
		removeOldest(c.b2)
	}

	c.t1.Set(key, value)
}

func (c *ARCCache[K, V]) Get(key K) (V, bool) {
	if val, ok := c.t1.Get(key); ok {
		return val, ok
	}

	return c.t2.Get(key)
}

func (c *ARCCache[K, V]) Remove(key K) {
	c.t1.Remove(key)
	c.t2.Remove(key)
	c.b1.Remove(key)
	c.b2.Remove(key)
}

func (c *ARCCache[K, V]) Evict(count int) {
	c.t1.Evict(count)
	c.t2.Evict(count)
}

func (c *ARCCache[K, V]) Len() int {
	return c.t1.Len() + c.t2.Len()
}

func (c *ARCCache[K, V]) replcae(direction bool) {
	var v V
	t1Len := c.t1.Len()
	if t1Len > 0 && (t1Len > c.prefer || (t1Len == c.prefer && direction)) {
		k, ok := removeOldest(c.t1)
		if ok {
			c.b1.Set(k, v)
		}
	} else {
		k, ok := removeOldest(c.t2)
		if ok {
			c.b2.Set(k, v)
		}
	}
}

func removeOldest[K comparable, V any](cache *LRUCache[K, V]) (K, bool) {
	ent := cache.evictList.Back()
	if ent != nil {
		cache.removeElement(ent)
		return ent.Value.(*lruItem[K, V]).key, true
	}
	var k K
	return k, false
}

func contains[K comparable, V any](cache *LRUCache[K, V], key K) bool {
	_, ok := cache.Get(key)
	return ok
}
