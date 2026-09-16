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

	_ = h.HSet(name, []byte("name"), []byte("Alice"))
	_ = h.HSet(name, []byte("age"), []byte("30"))

	// 测试 HLen
	lenVal, err := h.HLen(name)
	if err != nil || lenVal != 2 {
		t.Fatalf("HLen expected 2, got %d, err: %v", lenVal, err)
	}

	// 测试 HIncrBy
	newAge, err := h.HIncrBy(name, []byte("age"), 5)
	if err != nil || newAge != 35 {
		t.Fatalf("HIncrBy expected 35, got %d, err: %v", newAge, err)
	}

	// 测试 HDel
	ok, err := h.HDel(name, []byte("name"))
	if err != nil || !ok {
		t.Fatalf("HDel failed: %v", err)
	}

	lenVal, _ = h.HLen(name)
	if lenVal != 1 {
		t.Fatalf("HLen expected 1 after HDel, got %d", lenVal)
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
