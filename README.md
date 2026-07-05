# simplecache

[![Go Reference](https://pkg.go.dev/badge/github.com/MsN-12/simplecache.svg)](https://pkg.go.dev/github.com/MsN-12/simplecache)

`simplecache` is a small generic in-memory TTL cache for Go applications that do not need an external cache such as Redis.

The package is concurrency-safe and protects cached values from caller mutation by cloning values on both `Set` and `Get`. You can use automatic reflection-based cloning for convenience, or explicit clone functions for maximum control.

## Install

```sh
go get github.com/MsN-12/simplecache
```

Then import it in your application:

```go
import simplecache "github.com/MsN-12/simplecache"
```

## Usage

```go
package main

import (
	"fmt"
	"time"

	simplecache "github.com/MsN-12/simplecache"
)

func main() {
	cache := simplecache.MustNew[string, []int](time.Minute, simplecache.CloneSlice[int])

	items := []int{1, 2, 3}
	cache.Set("items", items)

	items[0] = 99

	cached, ok := cache.Get("items")
	if !ok {
		return
	}

	cached[1] = 88

	stillSafe, _ := cache.Get("items")
	fmt.Println(stillSafe) // [1 2 3]
}
```

Use `GetOrSet` when a value should be calculated only on cache miss:

```go
user, cached, err := cache.GetOrSet("user:1", func() (User, error) {
	return fetchUserFromDatabase("1")
})
if err != nil {
	return err
}

fmt.Println(user, cached)
```

Use `SetWithTTL` when one entry needs a different TTL from the cache default:

```go
err := cache.SetWithTTL("session:token", token, 10*time.Minute)
```

Enable background cleanup only when you want expired entries removed periodically without waiting for access:

```go
err := cache.StartCleanup(time.Minute)
if err != nil {
	return err
}
defer cache.StopCleanup()
```

## Cloning Values

The cache clones values when you call `Set` and `Get`. This prevents callers from mutating cached values through slices, maps, pointers, or nested mutable fields.

There are two ways to configure cloning.

### Automatic Cloning

Use `NewAuto` or `MustNewAuto` for convenient reflection-based deep cloning:

```go
cache := simplecache.MustNewAuto[string, User](time.Minute)
```

`DeepClone` supports common Go values:

- structs
- arrays
- slices
- maps
- pointers
- interfaces
- pointer, map, and slice cycles

Example:

```go
type Notifier interface {
	Notify(message string)
}

type EmailNotifier struct {
	Sent []string
}

func (n *EmailNotifier) Notify(message string) {
	n.Sent = append(n.Sent, message)
}

type Profile struct {
	Bio  string
	Tags []string
}

type User struct {
	Name     string
	Profile  Profile
	Notifier Notifier
}

cache := simplecache.MustNewAuto[string, User](time.Minute)
cache.Set("user:1", user)
```

Automatic cloning has important limitations:

- funcs, channels, and unsafe pointers are copied as-is
- unexported struct fields are shallow-copied
- resource-owning values such as files, sockets, database connections, timers, and mutexes should not rely on automatic cloning
- reflection is slower than a hand-written clone function
- interface fields are cloned based on the concrete runtime value, but resource-like concrete values can still be unsafe

Use automatic cloning for normal application data, DTOs, request/response structs, nested structs, slices, maps, and pointers. Use a custom clone function for critical production data or types with resources/invariants.

You can also use `DeepClone` directly:

```go
copied := simplecache.DeepClone(user)
```

### Explicit Clone Functions

Use a custom clone function when the type needs exact copy rules. This is the safest option for production code where a wrong clone could leak mutable state, duplicate a resource incorrectly, or break type invariants.

You should write a custom cloner when your cached value contains:

- interface fields whose concrete values may contain resources or mutable state
- `sync.Mutex`, `sync.RWMutex`, `sync.Once`, or other synchronization primitives
- channels, funcs, unsafe pointers, file handles, sockets, database connections, timers, contexts, or loggers
- unexported mutable fields that reflection cannot safely copy
- values with invariants that must be rebuilt instead of copied field-by-field
- very large values where reflection cloning is too slow or allocates too much

Worst-case example:

```go
type Notifier interface {
	Notify(message string)
}

type EmailNotifier struct {
	client *smtp.Client // resource, should not be cloned automatically
	buffer []byte       // mutable state
	mu     sync.Mutex   // synchronization primitive
}

type Phone struct {
	Number string
	Labels []string
}

type User struct {
	Name     string
	Age      int
	Notifier Notifier // interface can hide complex concrete values
	Phone    Phone
}
```

For this kind of type, do not rely on `MustNewAuto`. Write the clone rules yourself:

```go
func CloneUser(user User) User {
	user.Phone.Labels = simplecache.CloneSlice(user.Phone.Labels)

	// Decide intentionally what to do with interface/resource fields.
	// Here we keep the notifier shared because it owns external resources.
	user.Notifier = user.Notifier

	return user
}

cache := simplecache.MustNew[string, User](time.Minute, CloneUser)
```

Sometimes the safest clone is not a full deep copy. For resource fields, the correct behavior may be to share the resource, set it to nil, or rebuild it from configuration. That decision belongs in a custom cloner.

Use `Identity` only for immutable values or values that are safe to share by copy:

```go
cache := simplecache.MustNew[string, int](time.Minute, simplecache.Identity[int])
```

Use the provided shallow clone helpers for simple slices, maps, and byte slices:

```go
sliceCache := simplecache.MustNew[string, []string](time.Minute, simplecache.CloneSlice[string])
mapCache := simplecache.MustNew[string, map[string]int](time.Minute, simplecache.CloneMap[string, int])
bytesCache := simplecache.MustNew[string, []byte](time.Minute, simplecache.CloneBytes)
```

Use a custom clone function for structs with nested mutable fields:

```go
type User struct {
	Name     string
	Tags     []string
	Metadata map[string][]int
}

func CloneUser(user User) User {
	user.Tags = simplecache.CloneSlice(user.Tags)
	user.Metadata = simplecache.DeepClone(user.Metadata)
	return user
}

cache := simplecache.MustNew[string, User](time.Minute, CloneUser)
```

Custom clone functions are the safest choice when your type owns resources, contains mutexes, has unexported mutable fields, has interface fields with complex concrete values, or needs special copy rules.

## API

```go
cache, err := simplecache.New[K, V](ttl, clone)
cache := simplecache.MustNew[K, V](ttl, clone)
cache, err := simplecache.NewAuto[K, V](ttl)
cache := simplecache.MustNewAuto[K, V](ttl)
copied := simplecache.DeepClone(value)

cache.Set(key, value)
err := cache.SetWithTTL(key, value, ttl)
value, ok := cache.Get(key)
value, cached, err := cache.GetOrSet(key, loadFunc)
exists := cache.Has(key)
cache.Delete(key)
removed := cache.DeleteExpired()
cache.Clear()
n := cache.Len()
fresh := cache.LenFresh()
err := cache.StartCleanup(interval)
cache.StopCleanup()
```

`Len` returns the number of stored entries, including expired entries that have not been accessed or removed by `DeleteExpired` yet.

`LenFresh` removes expired entries and returns the number of unexpired entries.

Expired entries are removed when accessed by `Get` or `Has`, or when `DeleteExpired` or `LenFresh` is called. The cache does not start a background goroutine by default. Call `StartCleanup` to periodically remove expired entries in the background, and call `StopCleanup` before shutting down if cleanup was started.

`GetOrSet` calls the load function outside the cache lock. If multiple goroutines request the same missing key at the same time, the load function may run more than once.

## Production Notes

This package is intentionally small:

- Data is stored in process memory and is lost when the process exits.
- The cache is not distributed and does not replace Redis when multiple processes need a shared cache.
- There is no maximum size limit yet, so callers should avoid unbounded key growth.
- Clone cost depends on the value size and the clone function used.

## Benchmarks

Run benchmarks with:

```sh
go test -bench=. -benchmem ./...
```

## Requirements

Go 1.22 or newer.

## License

MIT
