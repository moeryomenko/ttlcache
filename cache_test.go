package cache

import (
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

func genTestEntry() gopter.Gen {
	notEmptyString := func(s string) bool {
		return s != ""
	}
	return gen.Struct(reflect.TypeOf(&testEntry{}), map[string]gopter.Gen{
		"Key":   gen.AnyString().SuchThat(notEmptyString),
		"Value": gen.AnyString().SuchThat(notEmptyString),
		"TTL":   gen.Int64Range(400, 500),
	})
}

func Test_Shard(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	// parameters.Rng.Seed(1633694940285315084) // for generate reproducible results.
	parameters.MaxSize = 5
	properties := gopter.NewProperties(parameters)

	testShard := newShard(100)

	properties.Property("get recently set entry", prop.ForAll(
		func(e testEntry) bool {
			testShard.Set(e.Key, e.Value, time.Duration(e.TTL)*time.Millisecond)
			_, exists := testShard.Get(e.Key)
			return exists
		},
		genTestEntry(),
	))

	properties.Property("get false on expired entry", prop.ForAll(
		func(e testEntry) bool {
			dur := time.Duration(e.TTL) * time.Microsecond
			testShard.Set(e.Key, e.Value, dur)
			time.Sleep(dur)
			_, exists := testShard.Get(e.Key)
			return !exists
		},
		genTestEntry(),
	))

	properties.Property("cache capacity doesn't exceed the specified", prop.ForAll(
		func(entries []testEntry) bool {
			capacity := len(entries)
			if capacity == 0 {
				capacity = 1
			}
			testShard := newShard(capacity)
			for _, e := range entries {
				testShard.Set(e.Key, e.Value, time.Duration(e.TTL)*time.Millisecond)
			}

			testShard.Set("test", "test", 300*time.Millisecond)

			return testShard.Len() == capacity
		},
		gen.SliceOf(genTestEntry()),
	))

	properties.TestingRun(t)
}
