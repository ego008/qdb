package bbolt

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	go_bbolt "go.etcd.io/bbolt"
)

// 辅助函数：生成 8 字节 BigEndian 的 int Key
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// 模拟参数配置
const (
	numHashes    = 1000 // 模拟 1000 个不同的 Hash 表 (name)
	itemsPerHash = 500  // 每个 Hash 表下有 500 条 KV
)

// ============================================================================
// 1. 单个大 Bucket 模式 (Single Bucket + Prefix)
// ============================================================================
func BenchmarkSingleBucket_Put_Random(b *testing.B) {
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "single.db")
	db, err := go_bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		b.Fatal(err)
	}
	db.NoSync = true
	defer db.Close()

	mainBucket := []byte("qdb_data")
	_ = db.Update(func(tx *go_bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(mainBucket)
		return err
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 每次随机选一个 Hash name 和 一个 Key 写入
		hashID := rand.Intn(numHashes)
		keyID := rand.Intn(itemsPerHash)

		hashName := fmt.Sprintf("hash_%d", hashID)
		realKey := append([]byte(hashName+":"), itob(keyID)...)
		val := []byte("value_payload_data_bytes")

		_ = db.Update(func(tx *go_bbolt.Tx) error {
			return tx.Bucket(mainBucket).Put(realKey, val)
		})
	}

	b.StopTimer()
	fi, _ := os.Stat(dbPath)
	b.ReportMetric(float64(fi.Size())/1024/1024, "DB_Size_MB")
}

func BenchmarkSingleBucket_Scan(b *testing.B) {
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "single_scan.db")
	db, _ := go_bbolt.Open(dbPath, 0600, nil)
	defer db.Close()

	mainBucket := []byte("qdb_data")
	_ = db.Update(func(tx *go_bbolt.Tx) error {
		bkt, _ := tx.CreateBucketIfNotExists(mainBucket)
		for h := 0; h < numHashes; h++ {
			hashPrefix := fmt.Sprintf("hash_%d:", h)
			for k := 0; k < itemsPerHash; k++ {
				_ = bkt.Put(append([]byte(hashPrefix), itob(k)...), []byte("val"))
			}
		}
		return nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		targetHash := fmt.Sprintf("hash_%d:", rand.Intn(numHashes))
		_ = db.View(func(tx *go_bbolt.Tx) error {
			c := tx.Bucket(mainBucket).Cursor()
			prefix := []byte(targetHash)
			count := 0
			for k, _ := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == targetHash; k, _ = c.Next() {
				count++
			}
			return nil
		})
	}
}

// ============================================================================
// 2. 动态多个 Bucket 模式 (Multi Bucket)
// ============================================================================
func BenchmarkMultiBucket_Put_Random(b *testing.B) {
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "multi.db")
	db, err := go_bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		b.Fatal(err)
	}
	db.NoSync = true
	defer db.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hashID := rand.Intn(numHashes)
		keyID := rand.Intn(itemsPerHash)

		bucketName := []byte(fmt.Sprintf("hash_%d", hashID))
		key := itob(keyID)
		val := []byte("value_payload_data_bytes")

		_ = db.Update(func(tx *go_bbolt.Tx) error {
			bkt, err := tx.CreateBucketIfNotExists(bucketName)
			if err != nil {
				return err
			}
			return bkt.Put(key, val)
		})
	}

	b.StopTimer()
	fi, _ := os.Stat(dbPath)
	b.ReportMetric(float64(fi.Size())/1024/1024, "DB_Size_MB")
}

func BenchmarkMultiBucket_Scan(b *testing.B) {
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "multi_scan.db")
	db, _ := go_bbolt.Open(dbPath, 0600, nil)
	defer db.Close()

	_ = db.Update(func(tx *go_bbolt.Tx) error {
		for h := 0; h < numHashes; h++ {
			bkt, _ := tx.CreateBucketIfNotExists([]byte(fmt.Sprintf("hash_%d", h)))
			for k := 0; k < itemsPerHash; k++ {
				_ = bkt.Put(itob(k), []byte("val"))
			}
		}
		return nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		targetHash := []byte(fmt.Sprintf("hash_%d", rand.Intn(numHashes)))
		_ = db.View(func(tx *go_bbolt.Tx) error {
			bkt := tx.Bucket(targetHash)
			if bkt == nil {
				return nil
			}
			c := bkt.Cursor()
			count := 0
			for k, _ := c.First(); k != nil; k, _ = c.Next() {
				count++
			}
			return nil
		})
	}
}
