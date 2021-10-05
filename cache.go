package cache

import (
	"container/list"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
)

// InMemCache is sharded cache with TTL.
type InMemCache struct {
	shards []*shard
}

// NewInMemCache returns new sharded cache.
// If opts is nil, also create cache with default configurations.
func NewInMemCache(n, capacity int) *InMemCache {
	cache := &InMemCache{}
	cache.shards = make([]*shard, n)
	for i := range cache.shards {
		cache.shards[i] = newShard(capacity)
	}

	return cache
}

// Set sets pair in cache with TTL.
// If the number of entries in the cache exceeds the bounds,
// it will start garbage collection.
func (c *InMemCache) Set(key string, val interface{}, ttl time.Duration) {
	c.getShard(xxhash.Sum64String(key)).Set(key, val, ttl)
}

// Get search and returns from cache.
func (c *InMemCache) Get(key string) (interface{}, bool) {
	return c.getShard(xxhash.Sum64String(key)).Get(key)
}

// Implementation of the jump consistent hash algorithm by John Lamping and Eric Veach for balancing load between
// shards and reduce contention for access to cache.
// ref: https://arxiv.org/pdf/1406.2294v1.pdf.
func (c *InMemCache) getShard(key uint64) *shard {
	var b, j int64

	shards := int64(len(c.shards))
	if shards <= 0 {
		shards = 1
	}

	for j < shards {
		b = j
		key = key*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(int64(1)<<31) / float64((key>>33)+1)))
	}

	return c.shards[b]
}

type entry struct {
	key    string
	data   interface{}
	timing time.Time
}

type shard struct {
	lock    sync.Mutex
	queue   *list.List
	entries map[string]*list.Element

	// capacity is the number of items in the shard, after which the gc starts.
	capacity int
	// expiredEntries marked as expired entries.
	expiredEntries []*list.Element
}

func (s *shard) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.queue.Len()
}

func (s *shard) Set(key string, value interface{}, ttl time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if element, exists := s.entries[key]; exists {
		s.queue.MoveToFront(element)
		element.Value.(*entry).data = value
		element.Value.(*entry).timing = time.Now().Add(ttl)
		return
	}

	if s.queue.Len() >= s.capacity {
		s.garbageCollect()
	}

	e := &entry{
		key:    key,
		data:   value,
		timing: time.Now().Add(ttl),
	}

	element := s.queue.PushFront(e)
	s.entries[key] = element
}

func (s *shard) Get(key string) (interface{}, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	element, exists := s.entries[key]
	if !exists {
		return nil, false
	}
	e := element.Value.(*entry)
	if e.timing.Before(time.Now()) {
		return nil, false
	}
	s.queue.MoveToFront(element)
	return e.data, true
}

// garbageCollect collects expired entries in shard.
// First, the garbageCollect marks expired entries.
// After, garbageCollect sweep marked entries.
func (s *shard) garbageCollect() {
	s.mark()
	s.sweep()
}

// mark marks expired entries.
func (s *shard) mark() {
	for _, element := range s.entries {
		if element.Value.(*entry).timing.Before(time.Now()) {
			// first, mark to remove expired entries.
			s.expiredEntries = append(s.expiredEntries, element)
		}
	}
}

// sweep removes expired and least recently used entries.
func (s *shard) sweep() {
	for _, element := range s.expiredEntries {
		s.purge(element)
	}
	// NOTE: if after removing expired entries,
	// len more that capacity, remove least recently used entries.
	for s.queue.Len() >= s.capacity {
		if element := s.queue.Back(); element != nil {
			s.purge(element)
		}
	}
}

func (s *shard) purge(element *list.Element) {
	item := s.queue.Remove(element).(*entry)
	delete(s.entries, item.key)
}

func newShard(capacity int) *shard {
	return &shard{
		capacity:       capacity,
		entries:        make(map[string]*list.Element),
		expiredEntries: make([]*list.Element, 0, capacity),
		queue:          list.New(),
	}
}
