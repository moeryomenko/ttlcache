package cache

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

type testEntry struct {
	Key   string
	Value string
	TTL   int64
}

var dedup = make(map[string]struct{})

func genTestEntry() gopter.Gen {
	notEmptyUniqueString := func(s string) bool {
		if s == "" {
			return false
		}

		_, ok := dedup[s]
		if !ok {
			dedup[s] = struct{}{}
		}

		return !ok
	}
	return gen.Struct(reflect.TypeOf(&testEntry{}), map[string]gopter.Gen{
		"Key":   gen.AnyString().SuchThat(notEmptyUniqueString),
		"Value": gen.AnyString().SuchThat(notEmptyUniqueString),
		"TTL":   gen.Int64Range(400, 500),
	})
}

func Test_LRUCache(t *testing.T) {
	testcases := map[string]EvictionPolicy{
		"LRU": LRU,
		"LFU": LFU,
		"ARC": ARC,
	}

	for name, testcase := range testcases {
		name := name
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			parameters := gopter.DefaultTestParameters()
			properties := gopter.NewProperties(parameters)

			properties.Property(fmt.Sprintf("cache(%s) capacity doesn't exceed the specified", name), prop.ForAll(
				func(capacity int, entries []testEntry) bool {
					cache := NewCache(capacity, testcase)

					for _, entry := range entries {
						cache.Set(entry.Key, entry.Value, time.Duration(entry.TTL)*time.Millisecond)
					}

					counter := 0
					for _, entry := range entries {
						_, err := cache.Get(entry.Key)
						if err == nil {
							counter++
						}
					}

					return counter <= capacity // less if keys were duplicated.
				},
				gen.IntRange(10, 20),
				gen.SliceOf(genTestEntry()),
			))

			properties.TestingRun(t)
		})
	}
}
