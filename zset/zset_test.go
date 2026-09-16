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

	// 1. 测试 ZSet 与 ZGet
	_ = z.ZSet(name, []byte("player_a"), 100)
	_ = z.ZSet(name, []byte("player_b"), 200)
	_ = z.ZSet(name, []byte("player_c"), 150)

	score, err := z.ZGet(name, []byte("player_c"))
	if err != nil || score != 150 {
		t.Fatalf("ZGet expected 150, got %d, err: %v", score, err)
	}

	// 更新 score
	_ = z.ZSet(name, []byte("player_a"), 300)
	score, _ = z.ZGet(name, []byte("player_a"))
	if score != 300 {
		t.Fatalf("ZSet update score failed, expected 300, got %d", score)
	}

	// 2. 测试 ZScan (按 Score 升序)
	res, err := z.ZScan(name, []byte(""), 0, 10)
	if err != nil {
		t.Fatalf("ZScan failed: %v", err)
	}
	if len(res) != 6 {
		t.Fatalf("ZScan len expected 6, got %d", len(res))
	}

	m0, m2, m4 := res[0].(string), res[2].(string), res[4].(string)
	if m0 != "player_c" || m2 != "player_b" || m4 != "player_a" {
		t.Fatalf("ZScan sorted order error: %s, %s, %s", m0, m2, m4)
	}

	// 3. 测试 ZRScan (按 Score 降序)
	res, err = z.ZRScan(name, []byte(""), 0, 10)
	if err != nil {
		t.Fatalf("ZRScan failed: %v", err)
	}
	if len(res) != 6 {
		t.Fatalf("ZRScan len expected 6, got %d", len(res))
	}

	rm0, rm2, rm4 := res[0].(string), res[2].(string), res[4].(string)
	if rm0 != "player_a" || rm2 != "player_b" || rm4 != "player_c" {
		t.Fatalf("ZRScan reverse order error: %s, %s, %s", rm0, rm2, rm4)
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
