# simplecache

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
value, ok := cache.Get(key)
cache.Delete(key)
removed := cache.DeleteExpired()
n := cache.Len()
```

`Len` returns the number of stored entries, including expired entries that have not been accessed or removed by `DeleteExpired` yet.

## Requirements

Go 1.22 or newer.

## License

MIT
