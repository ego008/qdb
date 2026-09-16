package zset_test

import (
	"path/filepath"
	"testing"

	"qdb/backend"
	"qdb/backend/bbolt"
	"qdb/backend/pebble"
	"qdb/zset"
)

func runZSetTests(t *testing.T, be backend.Backend) {
	z := zset.NewZSetEngine(be)
	name := "game_rank"

	// 1. 写入包含负数与小数的 Score
	_ = z.ZSet(name, []byte("player_a"), -10.5)
	_ = z.ZSet(name, []byte("player_b"), 200.0)
	_ = z.ZSet(name, []byte("player_c"), 0.0)

	score, err := z.ZGet(name, []byte("player_a"))
	if err != nil || score != -10.5 {
		t.Fatalf("ZGet expected -10.5, got %f, err: %v", score, err)
	}

	// 2. 测试 ZCount (区间 [-15.0, 50.0] 应包含 player_a(-10.5) 和 player_c(0.0))
	count, err := z.ZCount(name, -15.0, 50.0)
	if err != nil || count != 2 {
		t.Fatalf("ZCount expected 2, got %d, err: %v", count, err)
	}

	// 3. 测试 ZRank (升序: player_a(-10.5)[0] -> player_c(0.0)[1] -> player_b(200.0)[2])
	rank, err := z.ZRank(name, []byte("player_c"))
	if err != nil || rank != 1 {
		t.Fatalf("ZRank for player_c expected 1, got %d, err: %v", rank, err)
	}

	// 4. 测试 ZRem
	ok, err := z.ZRem(name, []byte("player_a"))
	if err != nil || !ok {
		t.Fatalf("ZRem player_a failed: %v", err)
	}

	rank, _ = z.ZRank(name, []byte("player_c"))
	if rank != 0 {
		t.Fatalf("ZRank for player_c after ZRem expected 0, got %d", rank)
	}
}

func TestZSetEngine_BBolt(t *testing.T) {
	dir := t.TempDir()
	be, err := bbolt.NewBBoltBackend(filepath.Join(dir, "bbolt.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	runZSetTests(t, be)
}

func TestZSetEngine_Pebble(t *testing.T) {
	dir := t.TempDir()
	be, err := pebble.NewPebbleBackend(filepath.Join(dir, "pebble.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	runZSetTests(t, be)
}
