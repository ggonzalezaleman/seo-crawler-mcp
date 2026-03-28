// Package frontier provides a URL priority queue for crawl scheduling.
package frontier

import (
	"container/heap"
	"sync"
)

// Item represents a URL queued for crawling.
type Item struct {
	URLID         int64
	NormalizedURL string
	Host          string
	Depth         int
}

// Queue is a thread-safe BFS priority queue (lower depth first) with dedup.
type Queue struct {
	mu   sync.Mutex
	h    itemHeap
	seen map[int64]bool
}

// New creates an empty frontier queue.
func New() *Queue {
	return &Queue{
		h:    itemHeap{},
		seen: map[int64]bool{},
	}
}

// Push adds an item to the queue. Duplicate urlIDs are silently ignored.
func (q *Queue) Push(item Item) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.seen[item.URLID] {
		return
	}
	q.seen[item.URLID] = true
	heap.Push(&q.h, item)
}

// Pop removes and returns the lowest-depth item. Returns false if empty.
func (q *Queue) Pop() (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.h) == 0 {
		return Item{}, false
	}
	return heap.Pop(&q.h).(Item), true
}

// Len returns the number of items in the queue.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.h)
}

// Peek returns the first item without removing it.
func (q *Queue) Peek() Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.h) == 0 {
		return Item{}
	}
	return q.h[0]
}

// Contains returns true if the given urlID has been pushed (even if already popped).
func (q *Queue) Contains(urlID int64) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.seen[urlID]
}

// itemHeap implements heap.Interface for BFS ordering by depth.
type itemHeap []Item

func (h itemHeap) Len() int            { return len(h) }
func (h itemHeap) Less(i, j int) bool  { return h[i].Depth < h[j].Depth }
func (h itemHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }

func (h *itemHeap) Push(x any) {
	*h = append(*h, x.(Item))
}

func (h *itemHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
