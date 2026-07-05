package simplecache

import (
	"errors"
	"testing"
	"time"
)

type testNotifier interface {
	Notify(string)
}

type testEmailNotifier struct {
	Sent []string
}

func (n *testEmailNotifier) Notify(message string) {
	n.Sent = append(n.Sent, message)
}

type testAddress struct {
	Street  string
	City    string
	Country string
}

type testProfile struct {
	Bio      string
	Age      int
	Address  testAddress
	Tags     []string
	Counters map[string][]int
}

type testUser struct {
	ID       string
	Name     string
	Email    string
	Profile  testProfile
	Notifier testNotifier
}

func TestNewAutoRejectsInvalidTTL(t *testing.T) {
	_, err := NewAuto[string, int](0)
	if !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("expected ErrInvalidTTL, got %v", err)
	}
}

func TestNewAutoDeepClonesNestedStruct(t *testing.T) {
	cache := MustNewAuto[string, testUser](time.Minute)

	user := testUser{
		ID:    "u-123",
		Name:  "Alice",
		Email: "alice@example.com",
		Profile: testProfile{
			Bio:     "Software Engineer",
			Age:     30,
			Address: testAddress{Street: "123 Main St", City: "Berlin", Country: "Germany"},
			Tags:    []string{"go", "cache"},
			Counters: map[string][]int{
				"login": []int{1, 2},
			},
		},
		Notifier: &testEmailNotifier{Sent: []string{"created"}},
	}

	cache.Set("user", user)
	user.Profile.Tags[0] = "changed"
	user.Profile.Counters["login"][0] = 99
	user.Notifier.(*testEmailNotifier).Sent[0] = "changed"

	got, ok := cache.Get("user")
	if !ok {
		t.Fatal("expected user to exist")
	}

	if got.Profile.Tags[0] != "go" {
		t.Fatalf("expected cloned tags, got %+v", got.Profile.Tags)
	}
	if got.Profile.Counters["login"][0] != 1 {
		t.Fatalf("expected cloned nested map slice, got %+v", got.Profile.Counters)
	}
	if got.Notifier.(*testEmailNotifier).Sent[0] != "created" {
		t.Fatalf("expected cloned interface concrete value, got %+v", got.Notifier)
	}

	got.Profile.Tags[1] = "mutated"
	got.Profile.Counters["login"][1] = 88
	got.Notifier.(*testEmailNotifier).Sent[0] = "mutated"

	gotAgain, ok := cache.Get("user")
	if !ok {
		t.Fatal("expected user to still exist")
	}
	if gotAgain.Profile.Tags[1] != "cache" {
		t.Fatalf("expected Get result mutation not to affect cache, got %+v", gotAgain.Profile.Tags)
	}
	if gotAgain.Profile.Counters["login"][1] != 2 {
		t.Fatalf("expected Get result nested mutation not to affect cache, got %+v", gotAgain.Profile.Counters)
	}
	if gotAgain.Notifier.(*testEmailNotifier).Sent[0] != "created" {
		t.Fatalf("expected Get result interface mutation not to affect cache, got %+v", gotAgain.Notifier)
	}
}

func TestDeepClonePointerCycle(t *testing.T) {
	type node struct {
		Value int
		Next  *node
	}

	original := &node{Value: 1}
	original.Next = original

	cloned := DeepClone(original)
	if cloned == original {
		t.Fatal("expected pointer to be cloned")
	}
	if cloned.Next != cloned {
		t.Fatal("expected cloned cycle to point to cloned node")
	}
}

func TestDeepCloneMapCycle(t *testing.T) {
	original := map[string]any{}
	original["self"] = original

	cloned := DeepClone(original)
	self, ok := cloned["self"].(map[string]any)
	if !ok {
		t.Fatalf("expected cloned self reference, got %T", cloned["self"])
	}

	self["marker"] = true
	if cloned["marker"] != true {
		t.Fatal("expected cloned map cycle to point to cloned map")
	}
}

func TestDeepCloneHandlesNilValues(t *testing.T) {
	type value struct {
		Slice []string
		Map   map[string]int
		Ptr   *int
		Any   any
	}

	cloned := DeepClone(value{})
	if cloned.Slice != nil || cloned.Map != nil || cloned.Ptr != nil || cloned.Any != nil {
		t.Fatalf("expected nil fields to stay nil, got %+v", cloned)
	}
}

func TestDeepCloneKeepsUnsupportedRuntimeValuesAsIs(t *testing.T) {
	type value struct {
		Events chan string
		Fn     func() string
	}

	original := value{
		Events: make(chan string),
		Fn: func() string {
			return "ok"
		},
	}

	cloned := DeepClone(original)
	if cloned.Events != original.Events {
		t.Fatal("expected channel to be copied as-is")
	}
	if cloned.Fn() != "ok" {
		t.Fatal("expected function to be copied as-is")
	}
}

func TestDeepCloneShallowCopiesUnexportedFields(t *testing.T) {
	type value struct {
		Name   string
		hidden []string
	}

	original := value{Name: "visible", hidden: []string{"secret"}}
	cloned := DeepClone(original)

	original.hidden[0] = "changed"
	if cloned.hidden[0] != "changed" {
		t.Fatal("expected unexported mutable field to be shallow-copied")
	}
}
