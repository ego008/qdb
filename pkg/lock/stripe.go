package lock

import (
	"hash/fnv"
	"sync"
)

type Locker struct {
	stripes uint32
	locks   []sync.RWMutex
}

func NewLocker(stripes uint32) *Locker {
	if stripes == 0 {
		stripes = 64
	}
	return &Locker{
		stripes: stripes,
		locks:   make([]sync.RWMutex, stripes),
	}
}

func (l *Locker) getLock(key string) *sync.RWMutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	idx := h.Sum32() % l.stripes
	return &l.locks[idx]
}

func (l *Locker) Lock(key string) {
	l.getLock(key).Lock()
}

func (l *Locker) Unlock(key string) {
	l.getLock(key).Unlock()
}

func (l *Locker) RLock(key string) {
	l.getLock(key).RLock()
}

func (l *Locker) RUnlock(key string) {
	l.getLock(key).RUnlock()
}
