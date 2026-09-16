package qdb_test

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"qdb"
)

func TestDB_LongTerm_Features(t *testing.T) {
	dir := t.TempDir()

	engines := []qdb.EngineType{qdb.EngineBBolt, qdb.EnginePebble}

	for _, eng := range engines {
		t.Run(string(eng), func(t *testing.T) {
			dbPath := filepath.Join(dir, string(eng))
			db, err := qdb.Open(eng, dbPath)
			if err != nil {
				t.Fatalf("Failed to open db: %v", err)
			}

			// 1. 写入测试
			_ = db.HSet("users", []byte("u1"), []byte("v1"))
			_ = db.ZSet("ranks", []byte("m1"), 99.5)

			// 2. TTL 测试
			testKey := "session_key"
			db.Expire(testKey, 50*time.Millisecond)

			// 立即检查：应该尚未过期 (IsExpired 应为 false)
			if db.IsExpired(testKey) {
				t.Fatalf("Key should not be expired immediately")
			}

			// 等待足够长的时间（包含后台 clean 和时间片）
			time.Sleep(100 * time.Millisecond)

			// 再次检查：此时无论被后台清理了还是时间到了，IsExpired 都必须返回 true
			if !db.IsExpired(testKey) {
				t.Fatalf("Key should be expired after duration")
			}

			// 3. 快照 Dump & Load
			var buf bytes.Buffer
			if err := db.DumpSnapshot(&buf); err != nil {
				t.Fatalf("DumpSnapshot failed: %v", err)
			}

			_ = db.Close()

			// 从 Snapshot 全量恢复
			newDbPath := filepath.Join(dir, string(eng)+"_restore")
			newDb, err := qdb.Open(eng, newDbPath)
			if err != nil {
				t.Fatalf("Failed to open restore db: %v", err)
			}
			defer newDb.Close()

			if err := newDb.LoadSnapshot(&buf); err != nil {
				t.Fatalf("LoadSnapshot failed: %v", err)
			}

			// 校验恢复后的完整性
			val, err := newDb.HGet("users", []byte("u1"))
			if err != nil || string(val) != "v1" {
				t.Fatalf("Restored Hash value mismatch, expected v1, got %s", val)
			}

			score, err := newDb.ZGet("ranks", []byte("m1"))
			if err != nil || score != 99.5 {
				t.Fatalf("Restored ZSet score mismatch, expected 99.5, got %f", score)
			}
		})
	}
}
