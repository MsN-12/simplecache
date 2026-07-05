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

func ExampleCache_GetOrSet() {
	cache := simplecache.MustNew[string, string](time.Minute, simplecache.Identity[string])

	value, cached, err := cache.GetOrSet("name", func() (string, error) {
		return "mohsen", nil
	})
	if err != nil {
		return
	}

	fmt.Println(value, cached)

	// Output: mohsen false
}

func ExampleCache_SetWithTTL() {
	cache := simplecache.MustNew[string, string](time.Minute, simplecache.Identity[string])

	_ = cache.SetWithTTL("session", "abc", 5*time.Minute)
	fmt.Println(cache.Has("session"))

	// Output: true
}
