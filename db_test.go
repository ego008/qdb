package qdb_test

import (
	"path/filepath"
	"testing"

	"qdb"
)

func TestDB_Integration(t *testing.T) {
	dir := t.TempDir()

	engines := []qdb.EngineType{qdb.EngineBBolt, qdb.EnginePebble}

	for _, eng := range engines {
		t.Run(string(eng), func(t *testing.T) {
			dbPath := filepath.Join(dir, string(eng))
			db, err := qdb.Open(eng, dbPath)
			if err != nil {
				t.Fatalf("Failed to open db: %v", err)
			}
			defer db.Close()

			// Hash API
			err = db.HSet("my_hash", []byte("f1"), []byte("v1"))
			if err != nil {
				t.Fatal(err)
			}
			val, err := db.HGet("my_hash", []byte("f1"))
			if err != nil || string(val) != "v1" {
				t.Fatalf("HGet failed: %v", err)
			}

			// ZSet API
			err = db.ZSet("my_zset", []byte("m1"), 50)
			if err != nil {
				t.Fatal(err)
			}
			score, err := db.ZGet("my_zset", []byte("m1"))
			if err != nil || score != 50 {
				t.Fatalf("ZGet failed: %v", err)
			}
		})
	}
}
