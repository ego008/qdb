package qdb

import (
	"qdb/backend"
	"qdb/backend/bbolt"
	"qdb/backend/pebble"
	"qdb/hash"
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

	return &DB{
		be:   be,
		hash: hash.NewHashEngine(be),
		zset: zset.NewZSetEngine(be),
	}, nil
}

func (db *DB) Close() error {
	return db.be.Close()
}

// Hash API 转发
func (db *DB) HSet(name string, key, value []byte) error    { return db.hash.HSet(name, key, value) }
func (db *DB) HGet(name string, key []byte) ([]byte, error) { return db.hash.HGet(name, key) }
func (db *DB) HScan(name string, keyStart []byte, limit int) ([][]byte, error) {
	return db.hash.HScan(name, keyStart, limit)
}
func (db *DB) HRScan(name string, keyStart []byte, limit int) ([][]byte, error) {
	return db.hash.HRScan(name, keyStart, limit)
}

// ZSet API 转发
func (db *DB) ZSet(name string, member []byte, score uint64) error {
	return db.zset.ZSet(name, member, score)
}
func (db *DB) ZGet(name string, member []byte) (uint64, error) {
	return db.zset.ZGet(name, member)
}
func (db *DB) ZScan(name string, memberStart []byte, scoreStart uint64, limit int) ([]interface{}, error) {
	return db.zset.ZScan(name, memberStart, scoreStart, limit)
}
func (db *DB) ZRScan(name string, memberStart []byte, scoreStart uint64, limit int) ([]interface{}, error) {
	return db.zset.ZRScan(name, memberStart, scoreStart, limit)
}
