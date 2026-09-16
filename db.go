package qdb

import (
	"io"
	"time"

	"qdb/backend"
	"qdb/backend/bbolt"
	"qdb/backend/pebble"
	"qdb/backup"
	"qdb/hash"
	"qdb/pkg/ttl"
	"qdb/zset"
)

type EngineType string

const (
	EngineBBolt  EngineType = "bbolt"
	EnginePebble EngineType = "pebble"
)

type DB struct {
	be   backend.Backend
	hash *hash.HashEngine
	zset *zset.ZSetEngine
	ttl  *ttl.TTLEngine
}

func Open(engine EngineType, path string) (*DB, error) {
	var be backend.Backend
	var err error

	switch engine {
	case EngineBBolt:
		be, err = bbolt.NewBBoltBackend(path)
	case EnginePebble:
		be, err = pebble.NewPebbleBackend(path)
	}

	if err != nil {
		return nil, err
	}

	db := &DB{
		be:   be,
		hash: hash.NewHashEngine(be),
		zset: zset.NewZSetEngine(be),
	}

	// 注册 TTL 到期自动清理回调
	db.ttl = ttl.NewTTLEngine(func(key string) {
		_ = be.Delete([]byte(key))
	})

	return db, nil
}

func (db *DB) Close() error {
	db.ttl.Stop()
	return db.be.Close()
}

// TTL API
func (db *DB) Expire(key string, ttl time.Duration) {
	db.ttl.SetTTL(key, ttl)
}

func (db *DB) IsExpired(key string) bool {
	return db.ttl.IsExpired(key)
}

// 备份 API
func (db *DB) DumpSnapshot(w io.Writer) error {
	return backup.ExportSnapshot(db.be, w)
}

func (db *DB) LoadSnapshot(r io.Reader) error {
	return backup.ImportSnapshot(db.be, r)
}

func (db *DB) DumpToFile(filepath string) error {
	return backup.ExportToFile(db.be, filepath)
}

func (db *DB) LoadFromFile(filepath string) error {
	return backup.ImportFromFile(db.be, filepath)
}

// Hash API
func (db *DB) HSet(name string, key, value []byte) error    { return db.hash.HSet(name, key, value) }
func (db *DB) HGet(name string, key []byte) ([]byte, error) { return db.hash.HGet(name, key) }
func (db *DB) HDel(name string, key []byte) (bool, error)   { return db.hash.HDel(name, key) }
func (db *DB) HLen(name string) (int, error)                { return db.hash.HLen(name) }
func (db *DB) HIncrBy(name string, key []byte, inc int64) (int64, error) {
	return db.hash.HIncrBy(name, key, inc)
}
func (db *DB) HScan(name string, keyStart []byte, limit int) ([][]byte, error) {
	return db.hash.HScan(name, keyStart, limit)
}
func (db *DB) HRScan(name string, keyStart []byte, limit int) ([][]byte, error) {
	return db.hash.HRScan(name, keyStart, limit)
}

// ZSet API
func (db *DB) ZSet(name string, member []byte, score float64) error {
	return db.zset.ZSet(name, member, score)
}
func (db *DB) ZGet(name string, member []byte) (float64, error) {
	return db.zset.ZGet(name, member)
}
func (db *DB) ZRem(name string, member []byte) (bool, error) {
	return db.zset.ZRem(name, member)
}
func (db *DB) ZRank(name string, member []byte) (int64, error) {
	return db.zset.ZRank(name, member)
}
func (db *DB) ZCount(name string, minScore, maxScore float64) (int64, error) {
	return db.zset.ZCount(name, minScore, maxScore)
}
func (db *DB) ZScan(name string, memberStart []byte, scoreStart float64, limit int) ([]interface{}, error) {
	return db.zset.ZScan(name, memberStart, scoreStart, limit)
}
func (db *DB) ZRScan(name string, memberStart []byte, scoreStart float64, limit int) ([]interface{}, error) {
	return db.zset.ZRScan(name, memberStart, scoreStart, limit)
}
