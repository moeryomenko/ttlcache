package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func Test_TTL(t *testing.T) {
	testcaces := map[string]struct {
		policy     evictionPolicy
		evictedKey string
	}{
		`LRU`: {policy: LRU, evictedKey: `k2`},
		`LFU`: {policy: LFU, evictedKey: `k1`},
		`ARC`: {policy: ARC, evictedKey: `k2`},
	}

	for name, tc := range testcaces {
		tc := tc
		t.Run(fmt.Sprintf(`cache(%s) eviction expired items`, name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cache := NewCache(ctx, 1, WithEvictionPolicy(tc.policy), WithTTLEpochGranularity(10*time.Millisecond))

			cache.SetNX(`test`, `string`, 10*time.Millisecond)
			<-time.After(5 * time.Millisecond)
			value, ok := cache.Get(`test`)
			if !ok {
				fail(t, `expected key not expired`)
			}
			if v, ok := value.(string); !ok || v != `string` {
				fail(t, `unexpected value %v`, value)
			}
			<-time.After(20 * time.Millisecond)
			_, ok = cache.Get(`test`)
			if ok {
				fail(t, `expected key expired`)
			}
		})

		t.Run(fmt.Sprintf(`cache(%s) update expiration`, name), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cache := NewCache(ctx, 10, WithEvictionPolicy(tc.policy), WithTTLEpochGranularity(10*time.Millisecond))

			cache.SetNX(`test`, `string`, 20*time.Millisecond)
			<-time.After(10 * time.Millisecond)
			cache.SetNX(`test`, `new string`, 20*time.Millisecond)

			value, ok := cache.Get(`test`)
			if !ok {
				fail(t, `expected key not expired`)
			}
			if v, ok := value.(string); !ok || v != `new string` {
				fail(t, `unexpected value %v`, value)
			}
			<-time.After(10 * time.Millisecond)
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

			cache := NewCache(ctx, 2, WithEvictionPolicy(tc.policy), WithTTLEpochGranularity(10*time.Millisecond))
			cache.SetNX(`k1`, `v1`, 10*time.Millisecond)
			cache.SetNX(`k2`, `v2`, 10*time.Millisecond)
			_, ok := cache.Get(`k1`)
			if !ok {
				fail(t, `expected key dont evicted by policy`)
			}
			cache.SetNX(`k3`, `v3`, 30*time.Millisecond)
			_, ok = cache.Get(`k3`)
			if !ok {
				fail(t, `expected key dont evicted by policy`)
			}

			_, ok = cache.Get(tc.evictedKey)
			if ok {
				fail(t, `expected key evicted by policy`)
			}
			<-time.After(10 * time.Millisecond)
			cache.SetNX(`k4`, `v4`, 30*time.Millisecond)
			value3, ok := cache.Get(`k3`)
			if !ok {
				fail(t, `expected key(k3) not expired`)
			}
			if v, ok := value3.(string); !ok || v != `v3` {
				fail(t, `unexpected value %v`, value3)
			}
			value4, ok := cache.Get(`k4`)
			if !ok {
				fail(t, `expected key(k4) not expired`)
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
