package cache

import "time"

type config struct {
	policy      evictionPolicy
	granularity time.Duration
}

const defaultEpochGranularity = 1 * time.Second
