package simplecache_test

import (
	"fmt"
	"time"

	simplecache "github.com/MsN-12/simplecache"
)

type exampleNotifier interface {
	Notify(string)
}

type exampleEmailNotifier struct {
	Sent []string
}

func (n *exampleEmailNotifier) Notify(message string) {
	n.Sent = append(n.Sent, message)
}

type exampleProfile struct {
	Bio  string
	Tags []string
}

type exampleUser struct {
	Name     string
	Profile  exampleProfile
	Notifier exampleNotifier
}

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

func ExampleMustNewAuto() {
	cache := simplecache.MustNewAuto[string, exampleUser](time.Minute)

	user := exampleUser{
		Name:     "Alice",
		Profile:  exampleProfile{Bio: "Software Engineer", Tags: []string{"go", "cache"}},
		Notifier: &exampleEmailNotifier{Sent: []string{"created"}},
	}
	cache.Set("user", user)

	user.Profile.Tags[0] = "changed"
	user.Notifier.(*exampleEmailNotifier).Sent[0] = "changed"

	cached, _ := cache.Get("user")
	fmt.Println(cached.Name, cached.Profile.Tags, cached.Notifier.(*exampleEmailNotifier).Sent)

	// Output: Alice [go cache] [created]
}

func ExampleCache_StartCleanup() {
	cache := simplecache.MustNewAuto[string, string](time.Minute)

	if err := cache.StartCleanup(time.Minute); err != nil {
		return
	}
	defer cache.StopCleanup()

	cache.Set("key", "value")
	fmt.Println(cache.Has("key"))

	// Output: true
}
