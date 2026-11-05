package theme

import "testing"

func TestNamesAndGet(t *testing.T) {
	names := Names()
	if len(names) < 5 {
		t.Fatalf("expected several built-in themes, got %v", names)
	}
	for _, name := range names {
		if _, ok := Get(name); !ok {
			t.Errorf("Get(%q) failed", name)
		}
	}
	if _, ok := Get("DRACULA"); !ok {
		t.Error("Get should be case-insensitive")
	}
	if _, ok := Get("does-not-exist"); ok {
		t.Error("unknown theme resolved")
	}
}

func TestNextCycles(t *testing.T) {
	names := Names()
	seen := map[string]bool{}
	current := Default().Name
	for i := 1; i < len(names); i++ {
		current = Next(current).Name
		if seen[current] {
			t.Fatalf("cycle repeated %q early", current)
		}
		seen[current] = true
	}
	if got := Next(current).Name; got != Default().Name {
		t.Fatalf("cycle did not wrap: got %q, want %q", got, Default().Name)
	}
}

func TestUnknownNextStartsAtDefault(t *testing.T) {
	if got := Next("nope").Name; got != Default().Name {
		t.Fatalf("Next(unknown) = %q, want %q", got, Default().Name)
	}
}
