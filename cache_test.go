package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func Test_TTL(t *testing.T) {
	testcaces := map[string]evictionPolicy{
		`LRU`: LRU,
		`LFU`: LFU,
		`ARC`: ARC,
	}

	for name, policy := range testcaces {
		policy := policy
		t.Run(fmt.Sprintf(`cache(%s) eviction expired items`, name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cache := NewCache(ctx, 1, WithEvictionPolicy(policy))

			cache.SetNX(`test`, `string`, 2*time.Second)
			<-time.After(time.Second)
			value, ok := cache.Get(`test`)
			if !ok {
				fail(t, `expected key not expired`)
			}
			if v, ok := value.(string); !ok || v != `string` {
				fail(t, `unexpected value %v`, value)
			}
			<-time.After(2 * time.Second)
			_, ok = cache.Get(`test`)
			if ok {
				fail(t, `expected key expired`)
			}
		})

		t.Run(fmt.Sprintf(`cache(%s) update expiration`, name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cache := NewCache(ctx, 10, WithEvictionPolicy(policy))

			cache.SetNX(`test`, `string`, 2*time.Second)
			<-time.After(time.Second)
			cache.SetNX(`test`, `new string`, 2*time.Second)

			value, ok := cache.Get(`test`)
			if !ok {
				fail(t, `expected key not expired`)
			}
			if v, ok := value.(string); !ok || v != `new string` {
				fail(t, `unexpected value %v`, value)
			}
			<-time.After(time.Second)
			value, ok = cache.Get(`test`)
			if !ok {
				fail(t, `expected key not expired`)
			}
			if v, ok := value.(string); !ok || v != `new string` {
				fail(t, `unexpected value %v`, value)
			}
		})

		t.Run(fmt.Sprintf(`cache(%s) eviction policy and expiration`, name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cache := NewCache(ctx, 2, WithEvictionPolicy(policy))
			cache.SetNX(`k1`, `v1`, time.Second)
			cache.SetNX(`k2`, `v2`, time.Second)
			cache.SetNX(`k3`, `v3`, 2*time.Second)

			_, ok := cache.Get(`k1`)
			if ok {
				fail(t, `expected key evicted by lru policy`)
			}
			<-time.After(time.Second)
			cache.SetNX(`k4`, `v4`, time.Second)
			value3, ok := cache.Get(`k3`)
			if !ok {
				fail(t, `expected key not expired`)
			}
			if v, ok := value3.(string); !ok || v != `v3` {
				fail(t, `unexpected value %v`, value3)
			}
			value4, ok := cache.Get(`k4`)
			if !ok {
				fail(t, `expected key not expired`)
			}
			if v, ok := value4.(string); !ok || v != `v4` {
				fail(t, `unexpected value %v`, value4)
			}
		})
	}
}

func fail(t *testing.T, msg string, args ...any) {
	t.Logf(msg, args...)
	t.FailNow()
}
