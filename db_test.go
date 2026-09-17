package qdb_test

import (
	"bytes"
	"fmt"
	"os"
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

func TestDB_Compact(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "compact_test.db")
	compactedPath := filepath.Join(dir, "compacted.db")

	db, err := qdb.Open(qdb.EngineBBolt, dbPath)
	if err != nil {
		t.Fatalf("Open DB failed: %v", err)
	}
	defer db.Close()

	// 1. 写入大量临时数据产生碎片
	for i := 0; i < 1000; i++ {
		_ = db.HSet("big_hash", []byte(fmt.Sprintf("k_%d", i)), []byte("some_large_dummy_value_data_payload"))
	}

	// 2. 彻底清空所有数据（产生大量闲置页 / 碎片）
	for i := 0; i < 1000; i++ {
		_, _ = db.HDel("big_hash", []byte(fmt.Sprintf("k_%d", i)))
	}

	// 3. 执行物理压缩整理
	err = db.CompactTo(compactedPath)
	if err != nil {
		t.Fatalf("CompactTo failed: %v", err)
	}

	// 4. 断言压缩后的文件明显小于原文件
	origInfo, _ := os.Stat(dbPath)
	compactInfo, _ := os.Stat(compactedPath)

	t.Logf("Original DB size: %d bytes, Compacted size: %d bytes", origInfo.Size(), compactInfo.Size())

	if compactInfo.Size() >= origInfo.Size() {
		t.Fatalf("Compacted file size (%d) should be smaller than original (%d)", compactInfo.Size(), origInfo.Size())
	}
}

func TestDB_OnlineCompact(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "online_compact_test.db")

	db, err := qdb.Open(qdb.EngineBBolt, dbPath)
	if err != nil {
		t.Fatalf("Open DB failed: %v", err)
	}
	defer db.Close()

	// 1. 写入大量数据并保留一条关键数据验证可用性
	_ = db.HSet("users", []byte("keep_me"), []byte("important_value"))
	for i := 0; i < 1000; i++ {
		_ = db.HSet("users", []byte(fmt.Sprintf("temp_%d", i)), []byte("large_dummy_bytes_to_fill_pages"))
	}

	// 2. 批量删除数据，产生膨胀碎页面
	for i := 0; i < 1000; i++ {
		_, _ = db.HDel("users", []byte(fmt.Sprintf("temp_%d", i)))
	}

	beforeInfo, _ := os.Stat(dbPath)

	// 3. 执行在线收缩与原位替换
	if err := db.OnlineCompact(); err != nil {
		t.Fatalf("OnlineCompact failed: %v", err)
	}

	afterInfo, _ := os.Stat(dbPath)

	t.Logf("Before OnlineCompact size: %d bytes, After: %d bytes", beforeInfo.Size(), afterInfo.Size())

	// 4. 验证体积变小
	if afterInfo.Size() >= beforeInfo.Size() {
		t.Fatalf("File size after OnlineCompact (%d) should be smaller than before (%d)", afterInfo.Size(), beforeInfo.Size())
	}

	// 5. 验证在原位替换后，数据库依然能正常读写，数据无丢失
	val, err := db.HGet("users", []byte("keep_me"))
	if err != nil || string(val) != "important_value" {
		t.Fatalf("Data mismatch after OnlineCompact: got %s, err %v", val, err)
	}
}
