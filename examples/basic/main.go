package main

import (
	"fmt"
	"log"
	"qdb"
)

func main() {
	db, err := qdb.Open(qdb.EnginePebble, "./qdb_pebble_data")
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	// Hash 操作示例
	_ = db.HSet("users", []byte("user_1"), []byte("Alice"))
	_ = db.HSet("users", []byte("user_2"), []byte("Bob"))

	val, _ := db.HGet("users", []byte("user_1"))
	fmt.Printf("HGet user_1: %s\n", string(val))

	hScanRes, _ := db.HRScan("users", []byte(""), 10)
	fmt.Printf("HRScan result size: %d\n", len(hScanRes)/2)

	// ZSet 操作示例
	_ = db.ZSet("leaderboard", []byte("player_1"), 100)
	_ = db.ZSet("leaderboard", []byte("player_2"), 250)

	score, _ := db.ZGet("leaderboard", []byte("player_2"))
	fmt.Printf("ZGet player_2 score: %f\n", score)

	zScanRes, _ := db.ZRScan("leaderboard", []byte(""), 300, 10)
	fmt.Printf("ZRScan result items: %d\n", len(zScanRes)/2)
}
