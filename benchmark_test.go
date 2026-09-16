package qdb_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"qdb"
)

func BenchmarkHashHSet_BBolt(b *testing.B) {
	dir := b.TempDir()
	db, err := qdb.Open(qdb.EngineBBolt, filepath.Join(dir, "bbolt.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		_ = db.HSet("bench_hash", []byte(key), []byte("value_data"))
	}
}

func BenchmarkHashHSet_Pebble(b *testing.B) {
	dir := b.TempDir()
	db, err := qdb.Open(qdb.EnginePebble, filepath.Join(dir, "pebble.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		_ = db.HSet("bench_hash", []byte(key), []byte("value_data"))
	}
}

func BenchmarkZSetZSet_Pebble(b *testing.B) {
	dir := b.TempDir()
	db, err := qdb.Open(qdb.EnginePebble, filepath.Join(dir, "pebble.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		member := fmt.Sprintf("player_%d", i)
		_ = db.ZSet("bench_zset", []byte(member), float64(i))
	}
}
