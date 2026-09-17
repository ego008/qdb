package main

import (
	"fmt"
	"log"
	"time"

	"qdb"
)

func main() {
	// 1. 打开 Pebble 引擎数据库
	db, err := qdb.Open(qdb.EnginePebble, "./qdb_data")
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	// 2. Hash 操作示例
	fmt.Println("--- Hash Demo ---")
	_ = db.HSet("user:1001", []byte("name"), []byte("Alice"))
	_ = db.HSet("user:1001", []byte("score"), []byte("100"))

	newVal, _ := db.HIncrBy("user:1001", []byte("score"), 20)
	fmt.Printf("Updated user score: %d\n", newVal)

	// 3. ZSet 浮点数排行榜示例
	fmt.Println("\n--- ZSet Demo ---")
	_ = db.ZSet("leaderboard", []byte("player_a"), 105.5)
	_ = db.ZSet("leaderboard", []byte("player_b"), 300.2)
	_ = db.ZSet("leaderboard", []byte("player_c"), -12.0)

	rank, _ := db.ZRank("leaderboard", []byte("player_a"))
	fmt.Printf("Rank of player_a: %d\n", rank)

	// 4. TTL 动态过期示例
	fmt.Println("\n--- TTL Demo ---")
	db.Expire("temp_session", 100*time.Millisecond)
	fmt.Printf("Is expired before sleep? %v\n", db.IsExpired("temp_session"))
	time.Sleep(150 * time.Millisecond)
	fmt.Printf("Is expired after sleep? %v\n", db.IsExpired("temp_session"))

	fmt.Println("\nQDB Operations Completed Successfully!")
}

func onlineCompact() {
	db, err := qdb.Open(qdb.EngineBBolt, "./qdb_production.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 开启后台协程定时在线碎片整理
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			log.Println("Starting scheduled online DB compaction...")
			if err := db.OnlineCompact(); err != nil {
				log.Printf("Online compact failed: %v", err)
			} else {
				log.Println("Online compact finished successfully.")
			}
		}
	}()

	// 业务正常读写，不受影响
	_ = db.HSet("session", []byte("token"), []byte("abc"))
	val, _ := db.HGet("session", []byte("token"))
	fmt.Printf("Token: %s\n", val)
}
