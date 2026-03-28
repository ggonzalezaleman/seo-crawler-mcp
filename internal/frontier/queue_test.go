package frontier

import "testing"

func TestPushPopDepthOrder(t *testing.T) {
	q := New()
	q.Push(Item{URLID: 1, NormalizedURL: "/deep", Depth: 3})
	q.Push(Item{URLID: 2, NormalizedURL: "/mid", Depth: 1})
	q.Push(Item{URLID: 3, NormalizedURL: "/root", Depth: 0})
	q.Push(Item{URLID: 4, NormalizedURL: "/mid2", Depth: 1})

	want := []int{0, 1, 1, 3}
	for i, wantDepth := range want {
		item, ok := q.Pop()
		if !ok {
			t.Fatalf("pop %d: expected item, got empty", i)
		}
		if item.Depth != wantDepth {
			t.Errorf("pop %d: depth = %d, want %d", i, item.Depth, wantDepth)
		}
	}

	_, ok := q.Pop()
	if ok {
		t.Error("expected empty queue after popping all items")
	}
}

func TestDedupSameURLID(t *testing.T) {
	q := New()
	q.Push(Item{URLID: 1, NormalizedURL: "/a", Depth: 0})
	q.Push(Item{URLID: 1, NormalizedURL: "/a", Depth: 0})
	q.Push(Item{URLID: 1, NormalizedURL: "/a", Depth: 5})

	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}

	item, ok := q.Pop()
	if !ok {
		t.Fatal("expected item")
	}
	if item.URLID != 1 {
		t.Errorf("URLID = %d, want 1", item.URLID)
	}

	_, ok = q.Pop()
	if ok {
		t.Error("expected empty after single deduped item")
	}
}

func TestEmptyPop(t *testing.T) {
	q := New()
	_, ok := q.Pop()
	if ok {
		t.Error("Pop on empty queue should return false")
	}
}

func TestContains(t *testing.T) {
	q := New()
	if q.Contains(42) {
		t.Error("Contains(42) should be false on empty queue")
	}

	q.Push(Item{URLID: 42, Depth: 0})
	if !q.Contains(42) {
		t.Error("Contains(42) should be true after push")
	}

	// Pop it — Contains should still return true (seen set).
	q.Pop()
	if !q.Contains(42) {
		t.Error("Contains(42) should be true even after pop")
	}
}

func TestLen(t *testing.T) {
	q := New()
	if q.Len() != 0 {
		t.Errorf("Len() = %d, want 0", q.Len())
	}

	q.Push(Item{URLID: 1, Depth: 0})
	q.Push(Item{URLID: 2, Depth: 1})
	if q.Len() != 2 {
		t.Errorf("Len() = %d, want 2", q.Len())
	}

	q.Pop()
	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}
}
