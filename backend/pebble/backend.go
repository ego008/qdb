package pebble

import (
	"bytes"
	"errors"

	"qdb/backend"

	cockroach_pebble "github.com/cockroachdb/pebble"
)

type PebbleBackend struct {
	db *cockroach_pebble.DB
}

func NewPebbleBackend(path string) (*PebbleBackend, error) {
	db, err := cockroach_pebble.Open(path, &cockroach_pebble.Options{})
	if err != nil {
		return nil, err
	}
	return &PebbleBackend{db: db}, nil
}

func (p *PebbleBackend) Get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if errors.Is(err, cockroach_pebble.ErrNotFound) {
		return nil, backend.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	res := make([]byte, len(val))
	copy(res, val)
	return res, nil
}

func (p *PebbleBackend) Put(key, value []byte) error {
	return p.db.Set(key, value, cockroach_pebble.Sync)
}

func (p *PebbleBackend) Delete(key []byte) error {
	return p.db.Delete(key, cockroach_pebble.Sync)
}

func (p *PebbleBackend) Compact(dstPath string) error {
	// Pebble 强制合并全量 Range LSM-Tree SST 页面
	start := []byte{0x00}
	end := []byte{0xff}
	return p.db.Compact(start, end, true)
}

func (p *PebbleBackend) NewBatch() backend.Batch {
	return &pebbleBatch{b: p.db.NewBatch(), db: p.db}
}

func (p *PebbleBackend) NewIterator(prefix []byte) backend.Iterator {
	iter, _ := p.db.NewIter(nil)
	return &pebbleIter{iter: iter, prefix: prefix}
}

func (p *PebbleBackend) Close() error {
	return p.db.Close()
}

type pebbleBatch struct {
	b  *cockroach_pebble.Batch
	db *cockroach_pebble.DB
}

func (m *pebbleBatch) Put(k, v []byte) { _ = m.b.Set(k, v, nil) }
func (m *pebbleBatch) Delete(k []byte) { _ = m.b.Delete(k, nil) }
func (m *pebbleBatch) Commit() error   { return m.b.Commit(cockroach_pebble.Sync) }
func (m *pebbleBatch) Close() error    { return m.b.Close() }

type pebbleIter struct {
	iter   *cockroach_pebble.Iterator
	prefix []byte
}

func (i *pebbleIter) First() bool {
	if !i.iter.SeekGE(i.prefix) {
		return false
	}
	return bytes.HasPrefix(i.iter.Key(), i.prefix)
}

func (i *pebbleIter) Last() bool {
	limit := append(append([]byte{}, i.prefix...), 0xFF)
	if !i.iter.SeekLT(limit) {
		return false
	}
	return bytes.HasPrefix(i.iter.Key(), i.prefix)
}

func (i *pebbleIter) Next() bool {
	if !i.iter.Next() {
		return false
	}
	return bytes.HasPrefix(i.iter.Key(), i.prefix)
}

func (i *pebbleIter) Prev() bool {
	if !i.iter.Prev() {
		return false
	}
	return bytes.HasPrefix(i.iter.Key(), i.prefix)
}

func (i *pebbleIter) Seek(key []byte) bool {
	if !i.iter.SeekGE(key) {
		return false
	}
	return i.iter.Valid() && bytes.HasPrefix(i.iter.Key(), i.prefix)
}

func (i *pebbleIter) Key() []byte   { return i.iter.Key() }
func (i *pebbleIter) Value() []byte { return i.iter.Value() }
func (i *pebbleIter) Valid() bool   { return i.iter.Valid() && bytes.HasPrefix(i.iter.Key(), i.prefix) }
func (i *pebbleIter) Error() error  { return i.iter.Error() }
func (i *pebbleIter) Close() error  { return i.iter.Close() }
