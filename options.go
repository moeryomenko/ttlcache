package cache

import "time"

// Option is an option that can be applied to cache.
type Option func(*config)

// WithEvictionPolicy sets eviction policy for cache.
func WithEvictionPolicy(policy evictionPolicy) Option {
	return func(c *config) {
		c.policy = policy
	}
}

// WithTTLEpochGranularity sets ttl epoch granularity.
func WithTTLEpochGranularity(period time.Duration) Option {
	return func(c *config) {
		c.granularity = period
	}
}
