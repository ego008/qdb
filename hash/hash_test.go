package hash_test

import (
	"path/filepath"
	"testing"

	"qdb/backend"
	"qdb/backend/bbolt"
	"qdb/backend/pebble"
	"qdb/hash"
)

func runHashTests(t *testing.T, be backend.Backend) {
	h := hash.NewHashEngine(be)
	name := "user_info"

	// 1. 测试 HSet 与 HGet
	err := h.HSet(name, []byte("name"), []byte("Alice"))
	if err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	err = h.HSet(name, []byte("age"), []byte("30"))
	if err != nil {
		t.Fatalf("HSet failed: %v", err)
	}

	val, err := h.HGet(name, []byte("name"))
	if err != nil || string(val) != "Alice" {
		t.Fatalf("HGet expected Alice, got %s, err: %v", val, err)
	}

	// 2. 测试 HScan (正向扫描)
	_ = h.HSet(name, []byte("k1"), []byte("v1"))
	_ = h.HSet(name, []byte("k2"), []byte("v2"))
	_ = h.HSet(name, []byte("k3"), []byte("v3"))
	_ = h.HSet(name, []byte("k4"), []byte("v4"))

	// 从 keyStart="" 开始，限制 2 条 (返回 key/val 键值对)
	res, err := h.HScan(name, []byte(""), 2)
	if err != nil {
		t.Fatalf("HScan failed: %v", err)
	}
	if len(res) != 4 {
		t.Fatalf("HScan len expected 4 slices, got %d", len(res))
	}
	if string(res[0]) != "age" || string(res[2]) != "k1" {
		t.Fatalf("HScan order incorrect, got: %s, %s", string(res[0]), string(res[2]))
	}

	// 从 keyStart="k2" 开始正向 Seek 扫描
	res, err = h.HScan(name, []byte("k2"), 10)
	if err != nil {
		t.Fatalf("HScan with keyStart failed: %v", err)
	}
	if len(res) < 4 || string(res[0]) != "k3" || string(res[2]) != "k4" {
		t.Fatalf("HScan Seek filtering failed, got first key: %s", string(res[0]))
	}

	// 3. 测试 HRScan (反向扫描)
	res, err = h.HRScan(name, []byte(""), 2)
	if err != nil {
		t.Fatalf("HRScan failed: %v", err)
	}
	if len(res) != 4 || string(res[0]) != "name" || string(res[2]) != "k4" {
		t.Fatalf("HRScan reverse order incorrect, got: %s, %s", string(res[0]), string(res[2]))
	}
}

func TestHashEngine_BBolt(t *testing.T) {
	dir := t.TempDir()
	be, err := bbolt.NewBBoltBackend(filepath.Join(dir, "bbolt.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	runHashTests(t, be)
}

func TestHashEngine_Pebble(t *testing.T) {
	dir := t.TempDir()
	be, err := pebble.NewPebbleBackend(filepath.Join(dir, "pebble.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	runHashTests(t, be)
}
