package bbolt

import (
	"bytes"

	"qdb/backend"

	go_bbolt "go.etcd.io/bbolt"
)

var bucketName = []byte("qdb_data")

type BBoltBackend struct {
	db *go_bbolt.DB
}

func NewBBoltBackend(path string) (*BBoltBackend, error) {
	db, err := go_bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *go_bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &BBoltBackend{db: db}, nil
}

func (b *BBoltBackend) Get(key []byte) ([]byte, error) {
	var val []byte
	err := b.db.View(func(tx *go_bbolt.Tx) error {
		bkt := tx.Bucket(bucketName)
		v := bkt.Get(key)
		if v == nil {
			return backend.ErrNotFound
		}
		val = append([]byte{}, v...)
		return nil
	})
	return val, err
}

func (b *BBoltBackend) Put(key, value []byte) error {
	return b.db.Update(func(tx *go_bbolt.Tx) error {
		return tx.Bucket(bucketName).Put(key, value)
	})
}

func (b *BBoltBackend) Delete(key []byte) error {
	return b.db.Update(func(tx *go_bbolt.Tx) error {
		return tx.Bucket(bucketName).Delete(key)
	})
}

func (b *BBoltBackend) NewBatch() backend.Batch {
	tx, _ := b.db.Begin(true)
	return &bboltBatch{tx: tx, bkt: tx.Bucket(bucketName)}
}

func (b *BBoltBackend) NewIterator(prefix []byte) backend.Iterator {
	tx, _ := b.db.Begin(false)
	c := tx.Bucket(bucketName).Cursor()
	return &bboltIter{tx: tx, c: c, prefix: prefix}
}

func (b *BBoltBackend) Close() error {
	return b.db.Close()
}

type bboltBatch struct {
	tx  *go_bbolt.Tx
	bkt *go_bbolt.Bucket
}

func (m *bboltBatch) Put(k, v []byte) { _ = m.bkt.Put(k, v) }
func (m *bboltBatch) Delete(k []byte) { _ = m.bkt.Delete(k) }
func (m *bboltBatch) Commit() error   { return m.tx.Commit() }
func (m *bboltBatch) Close() error    { return m.tx.Rollback() }

type bboltIter struct {
	tx     *go_bbolt.Tx
	c      *go_bbolt.Cursor
	prefix []byte
	k, v   []byte
	valid  bool
}

func (i *bboltIter) First() bool {
	i.k, i.v = i.c.Seek(i.prefix)
	i.valid = i.k != nil && bytes.HasPrefix(i.k, i.prefix)
	return i.valid
}

func (i *bboltIter) Last() bool {
	// 找到 > prefix 的开端并往回退一格
	limit := append(append([]byte{}, i.prefix...), 0xFF)
	k, _ := i.c.Seek(limit)
	if k == nil {
		i.k, i.v = i.c.Last()
	} else {
		i.k, i.v = i.c.Prev()
	}
	i.valid = i.k != nil && bytes.HasPrefix(i.k, i.prefix)
	return i.valid
}

func (i *bboltIter) Next() bool {
	i.k, i.v = i.c.Next()
	i.valid = i.k != nil && bytes.HasPrefix(i.k, i.prefix)
	return i.valid
}

func (i *bboltIter) Prev() bool {
	i.k, i.v = i.c.Prev()
	i.valid = i.k != nil && bytes.HasPrefix(i.k, i.prefix)
	return i.valid
}

func (i *bboltIter) Seek(key []byte) bool {
	i.k, i.v = i.c.Seek(key)
	i.valid = i.k != nil && bytes.HasPrefix(i.k, i.prefix)
	return i.valid
}

func (i *bboltIter) Key() []byte   { return i.k }
func (i *bboltIter) Value() []byte { return i.v }
func (i *bboltIter) Valid() bool   { return i.valid }
func (i *bboltIter) Error() error  { return nil }
func (i *bboltIter) Close() error  { return i.tx.Rollback() }
