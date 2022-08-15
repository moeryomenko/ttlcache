package cache

const (
	// Discards the least recently used items first.
	LRU evictionPolicy = iota
	// Discards the least frequently used items first.
	LFU
	// Adaptive replacement cache policy.
	ARC
)

// evictionPolicy incapsulated from user.
type evictionPolicy int
