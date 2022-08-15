package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLFU_TTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := newLFUCache(ctx, 1)

	cache.Set(`test`, `string`, 2*time.Second)
	<-time.After(time.Second)
	value, err := cache.Get(`test`)
	if err != nil {
		fail(t, `expected key not expired`)
	}
	if v, ok := value.(string); !ok || v != `string` {
		fail(t, `unexpected value %v`, value)
	}
	<-time.After(2 * time.Second)
	_, err = cache.Get(`test`)
	if !errors.Is(err, ErrNotFound) {
		fail(t, `expected key expired`)
	}
}

func TestLFU_TTLUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := newLFUCache(ctx, 10)

	cache.Set(`test`, `string`, 2*time.Second)
	<-time.After(time.Second)
	cache.Set(`test`, `new string`, 2*time.Second)

	value, err := cache.Get(`test`)
	if err != nil {
		fail(t, `expected key not expired`)
	}
	if v, ok := value.(string); !ok || v != `new string` {
		fail(t, `unexpected value %v`, value)
	}
	<-time.After(time.Second)
	value, err = cache.Get(`test`)
	if err != nil {
		fail(t, `expected key not expired`)
	}
	if v, ok := value.(string); !ok || v != `new string` {
		fail(t, `unexpected value %v`, value)
	}
}

func TestLFU_evictionWithTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := newLFUCache(ctx, 2)
	cache.Set(`k1`, `v1`, time.Second)
	cache.Set(`k2`, `v2`, time.Second)
	cache.Set(`k3`, `v3`, 2*time.Second)

	_, err := cache.Get(`k1`)
	if !errors.Is(err, ErrNotFound) {
		fail(t, `expected key evicted by lru policy`)
	}
	<-time.After(time.Second)
	cache.Set(`k4`, `v4`, time.Second)
	value3, err := cache.Get(`k3`)
	if err != nil {
		fail(t, `expected key not expired`)
	}
	if v, ok := value3.(string); !ok || v != `v3` {
		fail(t, `unexpected value %v`, value3)
	}
	value4, err := cache.Get(`k4`)
	if err != nil {
		fail(t, `expected key not expired`)
	}
	if v, ok := value4.(string); !ok || v != `v4` {
		fail(t, `unexpected value %v`, value4)
	}
}
