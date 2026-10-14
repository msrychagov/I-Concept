// Seed is explicit, refuses nonempty databases and never runs at startup.
package main

import (
	"log"
	"vucdiagnostic/internal/app"
	"vucdiagnostic/internal/platform/database"
)

func main() {
	dsn := database.URLFromEnv()
	if dsn == "" {
		log.Fatal("seed requires DATABASE_URL or DB_HOST")
	}
	a, err := app.New(app.Config{DatabaseURL: dsn})
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()
	if err := a.SeedDemo(); err != nil {
		log.Fatal(err)
	}
	log.Print("Demo data created: 1 teacher, 9 students, 3 platoons, both assessment types")
}
