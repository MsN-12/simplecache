# simple-cache

[![Go Reference](https://pkg.go.dev/badge/github.com/MsN-12/simple-cache.svg)](https://pkg.go.dev/github.com/MsN-12/simple-cache)

`simple-cache` is a small generic in-memory TTL cache for Go applications that do not need an external cache such as Redis.

The package is concurrency-safe and protects cached values from caller mutation by cloning values on both `Set` and `Get`.

## Install

```sh
go get github.com/MsN-12/simple-cache
```

Then import it in your application:

```go
import simplecache "github.com/MsN-12/simple-cache"
```

## Usage

```go
package main

import (
	"fmt"
	"time"

	simplecache "github.com/MsN-12/simple-cache"
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

## Clone Functions

Go cannot automatically deep-copy every possible generic value safely. Values may contain slices, maps, pointers, interfaces, cycles, mutexes, file handles, channels, or other resources.

For that reason, this package requires an explicit clone function when creating a cache.

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
	Name string
	Tags []string
}

func CloneUser(user User) User {
	user.Tags = simplecache.CloneSlice(user.Tags)
	return user
}

cache := simplecache.MustNew[string, User](time.Minute, CloneUser)
```

## API

```go
cache, err := simplecache.New[K, V](ttl, clone)
cache := simplecache.MustNew[K, V](ttl, clone)

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
```

`Len` returns the number of stored entries, including expired entries that have not been accessed or removed by `DeleteExpired` yet.

`LenFresh` removes expired entries and returns the number of unexpired entries.

Expired entries are removed when accessed by `Get` or `Has`, or when `DeleteExpired` or `LenFresh` is called. There is no background cleanup goroutine.

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
