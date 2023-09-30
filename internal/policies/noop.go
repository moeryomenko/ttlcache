package policies

type NoEvictionCache[K comparable, V any] map[K]V

func NewNoEvictionCache[K comparable, V any](wapmUpCapacity int) NoEvictionCache[K, V] {
	return make(map[K]V, wapmUpCapacity)
}

func (c NoEvictionCache[K, V]) Set(key K, value V) {
	c[key] = value
}

func (c NoEvictionCache[K, V]) Get(key K) (V, bool) {
	value, ok := c[key]
	return value, ok
}

func (c NoEvictionCache[K, V]) Len() int {
	return len(c)
}

func (c NoEvictionCache[K, V]) Remove(key K) {
	delete(c, key)
}

func (c NoEvictionCache[K, V]) Evict(_ int) {}
