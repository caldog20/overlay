package node

import "sync"

type Queue struct {
	items [][]byte
	mu    sync.Mutex
	cond  *sync.Cond
}

func NewQueue() *Queue {
	q := &Queue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *Queue) Push(in []byte) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append(q.items, in)
	q.cond.Signal()
}

func (q *Queue) Pop() []byte {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 {
		q.cond.Wait()
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item
}

func (q *Queue) IsEmpty() bool {
	return len(q.items) == 0
}

func (q *Queue) Len() int {
	return len(q.items)
}
