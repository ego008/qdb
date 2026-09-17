package qdb

import (
	"fmt"
	"io"
	"os"
	"sync"
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
	mu     sync.RWMutex // 用于保护 db.be / db.hash / db.zset 句柄切换的读写锁
	engine EngineType
	path   string

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
	default:
		return nil, fmt.Errorf("unsupported engine type: %s", engine)
	}

	if err != nil {
		return nil, err
	}

	db := &DB{
		engine: engine,
		path:   path,
		be:     be,
		hash:   hash.NewHashEngine(be),
		zset:   zset.NewZSetEngine(be),
	}

	db.ttl = ttl.NewTTLEngine(func(key string) {
		db.mu.RLock()
		defer db.mu.RUnlock()
		_ = db.be.Delete([]byte(key))
	})

	return db, nil
}

func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.ttl.Stop()
	return db.be.Close()
}

// CompactTo 压缩另存为指定路径
func (db *DB) CompactTo(dstPath string) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.be.Compact(dstPath)
}

// OnlineCompact 在线压缩并热替换当前数据库文件（零停机/毫秒级无感切换）
func (db *DB) OnlineCompact() error {
	// Pebble 是基于 LSM-Tree 的，可以直接触发原位物理 Compact
	if db.engine == EnginePebble {
		db.mu.RLock()
		defer db.mu.RUnlock()
		return db.be.Compact("")
	}

	// BBolt 模式：通过生成临时文件 + 原子替换实现热压缩
	tmpPath := db.path + ".tmp_compact"
	_ = os.Remove(tmpPath) // 确保临时文件干净

	// 1. 在后台读取原 DB 并将紧凑数据写入临时文件（期间允许并发读写）
	err := func() error {
		db.mu.RLock()
		defer db.mu.RUnlock()
		return db.be.Compact(tmpPath)
	}()
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("compact write temp file failed: %w", err)
	}

	// 2. 申请独占写锁，毫秒级完成句柄切换与文件替换
	db.mu.Lock()
	defer db.mu.Unlock()

	// 关闭旧底层句柄，释放文件锁
	if err := db.be.Close(); err != nil {
		return fmt.Errorf("close old backend failed: %w", err)
	}

	// 原子重命名覆盖旧数据库文件
	if err := os.Rename(tmpPath, db.path); err != nil {
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	// 重新打开压缩后的新数据库文件
	newBE, err := bbolt.NewBBoltBackend(db.path)
	if err != nil {
		return fmt.Errorf("reopen backend failed: %w", err)
	}

	// 刷新内部 Engine 关联的 backend 实例指针
	db.be = newBE
	db.hash = hash.NewHashEngine(newBE)
	db.zset = zset.NewZSetEngine(newBE)

	return nil
}

// TTL 模块（仅内存逻辑，无需 db.mu 锁）
func (db *DB) Expire(key string, ttl time.Duration) {
	db.ttl.SetTTL(key, ttl)
}

func (db *DB) IsExpired(key string) bool {
	return db.ttl.IsExpired(key)
}

// 快照备份与恢复
func (db *DB) DumpSnapshot(w io.Writer) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return backup.ExportSnapshot(db.be, w)
}

func (db *DB) LoadSnapshot(r io.Reader) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return backup.ImportSnapshot(db.be, r)
}

func (db *DB) DumpToFile(filepath string) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return backup.ExportToFile(db.be, filepath)
}

func (db *DB) LoadFromFile(filepath string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return backup.ImportFromFile(db.be, filepath)
}

// Hash 模块 API (全量加 RLock)
func (db *DB) HSet(name string, key, value []byte) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.hash.HSet(name, key, value)
}

func (db *DB) HGet(name string, key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.hash.HGet(name, key)
}

func (db *DB) HDel(name string, key []byte) (bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.hash.HDel(name, key)
}

func (db *DB) HLen(name string) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.hash.HLen(name)
}

func (db *DB) HIncrBy(name string, key []byte, inc int64) (int64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.hash.HIncrBy(name, key, inc)
}

func (db *DB) HScan(name string, keyStart []byte, limit int) ([][]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.hash.HScan(name, keyStart, limit)
}

func (db *DB) HRScan(name string, keyStart []byte, limit int) ([][]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.hash.HRScan(name, keyStart, limit)
}

// ZSet 模块 API (全量加 RLock)
func (db *DB) ZSet(name string, member []byte, score float64) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.zset.ZSet(name, member, score)
}

func (db *DB) ZGet(name string, member []byte) (float64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.zset.ZGet(name, member)
}

func (db *DB) ZRem(name string, member []byte) (bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.zset.ZRem(name, member)
}

func (db *DB) ZRank(name string, member []byte) (int64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.zset.ZRank(name, member)
}

func (db *DB) ZCount(name string, minScore, maxScore float64) (int64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.zset.ZCount(name, minScore, maxScore)
}

func (db *DB) ZScan(name string, memberStart []byte, scoreStart float64, limit int) ([]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.zset.ZScan(name, memberStart, scoreStart, limit)
}

func (db *DB) ZRScan(name string, memberStart []byte, scoreStart float64, limit int) ([]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.zset.ZRScan(name, memberStart, scoreStart, limit)
}
