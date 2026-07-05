package simplecache_test

import (
	"fmt"
	"time"

	simplecache "github.com/MsN-12/simple-cache"
)

func Example() {
	cache := simplecache.MustNew[string, int](time.Minute, simplecache.Identity[int])

	cache.Set("answer", 42)

	value, ok := cache.Get("answer")
	fmt.Println(value, ok)

	// Output: 42 true
}

func ExampleCloneSlice() {
	cache := simplecache.MustNew[string, []string](time.Minute, simplecache.CloneSlice[string])

	tags := []string{"go", "cache"}
	cache.Set("tags", tags)
	tags[0] = "changed"

	cached, _ := cache.Get("tags")
	fmt.Println(cached)

	// Output: [go cache]
}
