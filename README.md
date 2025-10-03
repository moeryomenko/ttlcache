# ttlcache

[![Go Reference](https://pkg.go.dev/badge/github.com/moeryomenko/ttlcache.svg)](https://pkg.go.dev/github.com/moeryomenko/ttlcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/moeryomenko/ttlcache)](https://goreportcard.com/report/github.com/moeryomenko/ttlcache)

A high-performance, thread-safe Go cache library with **Time-To-Live (TTL)** support and configurable **eviction policies** (LRU, LFU, ARC, NOOP). Built with generics for type safety and optimal performance.

## 📋 Table of Contents

- [Features](#-features)
- [Technologies & Stack](#-technologies--stack)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Usage Examples](#-usage-examples)
- [Eviction Policies](#-eviction-policies)
- [API Reference](#-api-reference)
- [Project Structure](#-project-structure)
- [Development](#-development)
- [Benchmarks](#-benchmarks)
- [License](#-license)
- [Author](#-author)

## ✨ Features

- **⏱️ TTL Support** - Automatic expiration of cached items with configurable granularity
- **🔄 Multiple Eviction Policies** - LRU (Least Recently Used), LFU (Least Frequently Used), ARC (Adaptive Replacement Cache), and NOOP (no eviction)
- **🔐 Thread-Safe** - Full concurrency support with mutex-based synchronization
- **🎯 Type-Safe** - Built with Go generics for compile-time type checking
- **⚡ High Performance** - Optimized data structures and efficient memory management
- **🎛️ Configurable** - Flexible capacity limits and TTL epoch granularity
- **🧹 Automatic Cleanup** - Background goroutine handles expired item removal
- **📦 Zero External Dependencies** - Only depends on `github.com/moeryomenko/synx` for synchronization primitives

## 🛠 Technologies & Stack

**Language:** Go 1.25.1

**Core Dependencies:**
- `github.com/moeryomenko/synx` v0.14.0 - Synchronization utilities

**Development Tools:**
- `golangci-lint` v2.5.0 - Comprehensive Go linter
- `gotestsum` v1.13.0 - Enhanced test runner
- `goimports` - Code formatting and import management

**Build System:**
- GNU Make for task automation
- Standard Go toolchain

## 📦 Installation

```bash
go get github.com/moeryomenko/ttlcache
```

**Requirements:** Go 1.25.1 or higher

## 🚀 Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/moeryomenko/ttlcache"
)

func main() {
    ctx := context.Background()

    // Create a new cache with capacity of 100 items and LRU eviction policy
    cache := cache.NewCache[string, string](ctx, 100, cache.WithEvictionPolicy(cache.LRU))

    // Set a value without expiration (will only be evicted by policy)
    cache.Set("key1", "value1")

    // Set a value with 5 second TTL
    cache.SetNX("key2", "value2", 5*time.Second)

    // Get a value
    if value, ok := cache.Get("key1"); ok {
        fmt.Println("Found:", value)
    }
}
```

## 📚 Usage Examples

### Basic Operations

```go
ctx := context.Background()
cache := cache.NewCache[string, int](ctx, 100)

// Set permanent value (evicted only by policy)
cache.Set("counter", 42)

// Set value with TTL
cache.SetNX("session", 12345, 10*time.Minute)

// Get value
if value, ok := cache.Get("counter"); ok {
    fmt.Println("Value:", value)
}

// Remove value
cache.Remove("counter")

// Get cache size
size := cache.Len()
```

### Using Different Eviction Policies

```go
// LRU - Least Recently Used (default)
lruCache := cache.NewCache[string, string](
    ctx,
    100,
    cache.WithEvictionPolicy(cache.LRU),
)

// LFU - Least Frequently Used
lfuCache := cache.NewCache[string, string](
    ctx,
    100,
    cache.WithEvictionPolicy(cache.LFU),
)

// ARC - Adaptive Replacement Cache
arcCache := cache.NewCache[string, string](
    ctx,
    100,
    cache.WithEvictionPolicy(cache.ARC),
)

// NOOP - No eviction (only TTL-based expiration)
noopCache := cache.NewCache[string, string](
    ctx,
    100,
    cache.WithEvictionPolicy(cache.NOOP),
)
```

### Custom TTL Granularity

By default, the TTL cleanup runs every second. You can customize this:

```go
cache := cache.NewCache[string, string](
    ctx,
    100,
    cache.WithEvictionPolicy(cache.LRU),
    cache.WithTTLEpochGranularity(100*time.Millisecond), // Check every 100ms
)
```

### Working with Complex Types

```go
type User struct {
    ID       int
    Username string
    Email    string
}

cache := cache.NewCache[int, *User](ctx, 1000)

user := &User{
    ID:       1,
    Username: "john_doe",
    Email:    "john@example.com",
}

cache.SetNX(user.ID, user, 1*time.Hour)

if foundUser, ok := cache.Get(1); ok {
    fmt.Printf("User: %s (%s)\n", foundUser.Username, foundUser.Email)
}
```

### Session Cache Example

```go
type Session struct {
    UserID    string
    Token     string
    CreatedAt time.Time
}

sessionCache := cache.NewCache[string, *Session](
    ctx,
    10000,
    cache.WithEvictionPolicy(cache.LRU),
    cache.WithTTLEpochGranularity(1*time.Second),
)

// Create new session with 30 minute expiry
session := &Session{
    UserID:    "user123",
    Token:     "abc-def-ghi",
    CreatedAt: time.Now(),
}
sessionCache.SetNX(session.Token, session, 30*time.Minute)

// Validate session
if session, ok := sessionCache.Get("abc-def-ghi"); ok {
    fmt.Println("Valid session for user:", session.UserID)
} else {
    fmt.Println("Session expired or not found")
}
```

## 🔄 Eviction Policies

### LRU (Least Recently Used)
Evicts the items that haven't been accessed for the longest time. Best for general-purpose caching where recent access patterns predict future access.

```go
cache.WithEvictionPolicy(cache.LRU)
```

### LFU (Least Frequently Used)
Evicts items with the lowest access frequency. Best when certain items are accessed much more frequently than others.

```go
cache.WithEvictionPolicy(cache.LFU)
```

### ARC (Adaptive Replacement Cache)
Adaptive policy that balances between recency and frequency. Automatically adjusts to access patterns. Best for workloads with changing patterns.

```go
cache.WithEvictionPolicy(cache.ARC)
```

### NOOP (No Eviction)
No automatic eviction based on access patterns - items are only removed when they expire via TTL or are manually removed. Best when you have strict memory limits and control expiration via TTL.

```go
cache.WithEvictionPolicy(cache.NOOP)
```

## 📖 API Reference

### Types

```go
type Cache[K comparable, V any]
```

Generic cache supporting any comparable key type and any value type.

### Constructor

```go
func NewCache[K comparable, V any](
    ctx context.Context,
    capacity int,
    opts ...Option,
) *Cache[K, V]
```

Creates a new cache instance. The cache runs a background cleanup goroutine that stops when `ctx` is canceled.

### Methods

#### `Set(key K, value V)`
Sets a value that persists until evicted by the replacement policy. Won't be removed by TTL expiration.

#### `SetNX(key K, value V, expiry time.Duration)`
Sets a value with TTL. The item will be automatically removed after `expiry` duration.

#### `Get(key K) (V, bool)`
Retrieves a value by key. Returns the value and `true` if found, zero value and `false` otherwise.

#### `Remove(key K)`
Manually removes an item from the cache.

#### `Len() int`
Returns the current number of items in the cache.

### Options

#### `WithEvictionPolicy(policy evictionPolicy) Option`
Sets the eviction policy. Available policies: `LRU`, `LFU`, `ARC`, `NOOP`. Default is `LRU`.

#### `WithTTLEpochGranularity(period time.Duration) Option`
Sets how often the background cleanup goroutine checks for expired items. Default is `1 * time.Second`.

## 📁 Project Structure

```
ttlcache/
├── cache.go              # Main cache implementation
├── cache_test.go         # Comprehensive test suite
├── config.go             # Configuration structures
├── interfaces.go         # Internal interfaces
├── options.go            # Functional options API
├── policies.go           # Eviction policy constants
├── internal/
│   └── policies/         # Eviction policy implementations
│       ├── lru.go        # Least Recently Used
│       ├── lfu.go        # Least Frequently Used
│       ├── arc.go        # Adaptive Replacement Cache
│       └── noop.go       # No eviction policy
├── go.mod                # Go module definition
├── Makefile              # Build and development tasks
├── LICENSE-MIT           # MIT License
└── LICENSE-APACHE        # Apache 2.0 License
```

## 🔧 Development

### Prerequisites

- Go 1.25.1 or higher
- Make (optional, for convenience commands)

### Available Make Commands

```bash
make lint          # Run linter with auto-fix
make test          # Run tests with coverage
make cover         # View coverage in browser
make fmt           # Format code
make mod           # Update dependencies
make check         # Run lint + test
make clean         # Clean build artifacts
make help          # Show all available commands
```

### Running Tests

```bash
# Run all tests with coverage
make test

# Or using go directly
go test -v -race -coverprofile=coverage.out ./...

# View coverage
make cover
```

### Code Quality

The project uses `golangci-lint` with a comprehensive configuration. Run linting with:

```bash
make lint
```

## ⚡ Benchmarks

The cache is designed for high-performance scenarios. All operations are O(1) average case for get/set operations, with eviction policies implemented using efficient data structures.

Run benchmarks with:

```bash
go test -bench=. -benchmem
```

## 📄 License

This project is dual-licensed under:

- [MIT License](./LICENSE-MIT)
- [Apache License 2.0](./LICENSE-APACHE)

You may choose either license to govern your use of this software.

## 👤 Author

**Maxim Eryomenko** ([@moeryomenko](https://github.com/moeryomenko))

- GitHub: [github.com/moeryomenko/ttlcache](https://github.com/moeryomenko/ttlcache)
- Package Documentation: [pkg.go.dev/github.com/moeryomenko/ttlcache](https://pkg.go.dev/github.com/moeryomenko/ttlcache)

---

⭐ If you find this project useful, please consider giving it a star on GitHub!

## 🙋 FAQ

### Q: What happens when the cache reaches capacity?

When you add a new item to a full cache, the eviction policy determines which existing item to remove. The cache will first try to remove any expired items, then apply the configured eviction policy if still over capacity.

### Q: Is this cache safe for concurrent use?

Yes, the cache is fully thread-safe and can be safely used from multiple goroutines simultaneously.

### Q: Can I use this cache without TTL?

Yes, use the `Set()` method instead of `SetNX()`. Items added with `Set()` will only be evicted by the replacement policy when the cache is full, not by TTL expiration.

### Q: Which eviction policy should I choose?

- Use **LRU** for general-purpose caching (default choice)
- Use **LFU** when some items are accessed much more frequently than others
- Use **ARC** for workloads with changing access patterns
- Use **NOOP** when you want to manage expiration purely via TTL

### Q: How much memory does the cache use?

Memory usage depends on your stored values and the number of items. The cache itself has minimal overhead - mainly the map structures and tracking metadata for the eviction policy.

### Q: Can I update the TTL of an existing item?

Yes, call `SetNX()` again with the same key and the new expiration time.

### Q: What happens when the context is canceled?

The background cleanup goroutine will stop, but the cache remains usable. No new automatic TTL cleanup will occur, but you can still manually use the cache.
