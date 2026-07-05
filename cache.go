// Package simplecache provides a small, generic, in-memory TTL cache.
package simplecache

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrInvalidTTL is returned when a cache is created with a non-positive TTL.
	ErrInvalidTTL = errors.New("simplecache: ttl must be greater than zero")

	// ErrNilCloneFunc is returned when a cache is created without a clone function.
	ErrNilCloneFunc = errors.New("simplecache: clone function must not be nil")
)

// CloneFunc returns an independent copy of a cached value.
//
// The cache calls this function on Set and Get. For immutable values, use
// Identity. For mutable values, provide a clone function that copies every
// mutable field that must be protected from caller mutation. Clone functions
// must not mutate their input and must be safe to call concurrently.
type CloneFunc[V any] func(V) V

type entry[V any] struct {
	id        uint64
	value     V
	expiresAt time.Time
}

// Cache is a concurrency-safe in-memory TTL cache.
//
// Values are copied on Set and Get using the configured CloneFunc. This avoids
// callers mutating the cached value through slices, maps, pointers, or structs
// containing mutable fields, as long as the CloneFunc performs the needed copy.
type Cache[K comparable, V any] struct {
	mu     sync.RWMutex
	items  map[K]entry[V]
	ttl    time.Duration
	clone  CloneFunc[V]
	nextID uint64
}

// New creates a cache whose entries expire after ttl.
func New[K comparable, V any](ttl time.Duration, clone CloneFunc[V]) (*Cache[K, V], error) {
	if ttl <= 0 {
		return nil, ErrInvalidTTL
	}

	if clone == nil {
		return nil, ErrNilCloneFunc
	}

	return &Cache[K, V]{
		items: make(map[K]entry[V]),
		ttl:   ttl,
		clone: clone,
	}, nil
}

// MustNew creates a cache and panics if the configuration is invalid.
func MustNew[K comparable, V any](ttl time.Duration, clone CloneFunc[V]) *Cache[K, V] {
	cache, err := New[K, V](ttl, clone)
	if err != nil {
		panic(err)
	}

	return cache
}

// Set stores value under key until the cache TTL expires.
func (c *Cache[K, V]) Set(key K, value V) {
	cloned := c.clone(value)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	c.items[key] = entry[V]{
		id:        c.nextID,
		value:     cloned,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Get returns a cloned cached value if key exists and has not expired.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	var zero V

	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return zero, false
	}

	now := time.Now()
	if !now.Before(item.expiresAt) {
		c.mu.Lock()
		current, exists := c.items[key]
		if exists && current.id == item.id {
			delete(c.items, key)
		}
		c.mu.Unlock()

		return zero, false
	}

	return c.clone(item.value), true
}

// Delete removes key from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// DeleteExpired removes all expired entries and returns the number removed.
func (c *Cache[K, V]) DeleteExpired() int {
	now := time.Now()
	removed := 0

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if !now.Before(item.expiresAt) {
			delete(c.items, key)
			removed++
		}
	}

	return removed
}

// Len returns the number of stored entries, including entries that may have
// expired but have not been accessed or removed by DeleteExpired yet.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}
