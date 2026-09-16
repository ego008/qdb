package ttl

import (
	"container/heap"
	"sync"
	"time"
)

type item struct {
	key      string
	expireAt time.Time
	index    int
}

type priorityQueue []*item

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].expireAt.Before(pq[j].expireAt) }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	it := x.(*item)
	it.index = n
	*pq = append(*pq, it)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	it.index = -1
	*pq = old[0 : n-1]
	return it
}

type OnExpireFunc func(key string)

type TTLEngine struct {
	mu       sync.Mutex
	pq       priorityQueue
	items    map[string]*item
	onExpire OnExpireFunc
	stopCh   chan struct{}
}

func NewTTLEngine(onExpire OnExpireFunc) *TTLEngine {
	te := &TTLEngine{
		pq:       make(priorityQueue, 0),
		items:    make(map[string]*item),
		onExpire: onExpire,
		stopCh:   make(chan struct{}),
	}
	heap.Init(&te.pq)
	go te.loop()
	return te
}

func (t *TTLEngine) SetTTL(key string, ttl time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	expireAt := time.Now().Add(ttl)
	if it, ok := t.items[key]; ok {
		it.expireAt = expireAt
		heap.Fix(&t.pq, it.index)
	} else {
		it := &item{key: key, expireAt: expireAt}
		t.items[key] = it
		heap.Push(&t.pq, it)
	}
}

// IsExpired 无副作用纯只读检查
func (t *TTLEngine) IsExpired(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	it, ok := t.items[key]
	if !ok {
		// Key 如果不在 items 中，说明已经被后台 clean 清理掉，或者根本没设过，算作已过期/不存在
		return true
	}

	return !time.Now().Before(it.expireAt)
}

func (t *TTLEngine) Stop() {
	close(t.stopCh)
}

func (t *TTLEngine) loop() {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.clean()
		}
	}
}

func (t *TTLEngine) clean() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for t.pq.Len() > 0 {
		head := t.pq[0]
		if now.Before(head.expireAt) {
			break
		}
		expiredItem := heap.Pop(&t.pq).(*item)
		delete(t.items, expiredItem.key)

		if t.onExpire != nil {
			go t.onExpire(expiredItem.key)
		}
	}
}
