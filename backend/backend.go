package backend

import "errors"

var ErrNotFound = errors.New("key not found")

type Iterator interface {
	First() bool
	Last() bool
	Next() bool
	Prev() bool
	Seek(key []byte) bool
	Key() []byte
	Value() []byte
	Valid() bool
	Error() error
	Close() error
}

type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
	Commit() error
	Close() error
}

type Backend interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Delete(key []byte) error
	NewBatch() Batch
	NewIterator(prefix []byte) Iterator
	Close() error
}
