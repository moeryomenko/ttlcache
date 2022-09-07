package policies

type NoEvictionCache map[string]any

func NewNoEvictionCache(wapmUpCapacity int) NoEvictionCache {
	return make(map[string]any, wapmUpCapacity)
}

func (c NoEvictionCache) Set(key string, value any) {
	c[key] = value
}

func (c NoEvictionCache) Get(key string) (any, bool) {
	value, ok := c[key]
	return value, ok
}

func (c NoEvictionCache) Len() int {
	return len(c)
}

func (c NoEvictionCache) Remove(key string) {
	delete(c, key)
}

func (c NoEvictionCache) Evict(_ int) {}
