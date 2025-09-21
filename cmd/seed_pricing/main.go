package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Database connection
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=salome_db sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Connected to database successfully")

	// Update apps with pricing data
	apps := map[string]int{
		"netflix":         150000,
		"disney_plus":     200000,
		"spotify":         120000,
		"youtube_premium": 180000,
		"apple_music":     100000,
		"canva":           250000,
		"adobe_creative":  300000,
		"office_365":      80000,
		"calm":            160000,
		"headspace":       140000,
	}

	for appID, totalPrice := range apps {
		query := `
			UPDATE apps SET 
				total_price = $1,
				is_active = true,
				is_available = true
			WHERE id = $2
		`

		result, err := db.Exec(query, totalPrice, appID)
		if err != nil {
			log.Printf("Error updating %s: %v", appID, err)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("Updated %s: total_price=%d, rows_affected=%d\n",
			appID, totalPrice, rowsAffected)
	}

	fmt.Println("Pricing data seeding completed!")
}
