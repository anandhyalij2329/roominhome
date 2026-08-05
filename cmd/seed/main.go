package main

import (
	"fmt"
	"log"

	"room-rental/internal/config"
	"room-rental/internal/database"
	"room-rental/internal/seed"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	n, err := seed.Run(db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nSeeded %d properties (10 room + 10 home + 10 pg + 10 shop).\n", n)
	fmt.Println("Owner : owner@adnivara.test / owner123")
	fmt.Println("Seeker: seeker@adnivara.test / seeker123")
}
