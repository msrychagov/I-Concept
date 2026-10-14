package main

import (
	"context"
	"log"
	"time"
	"vucdiagnostic/internal/migrations"
	"vucdiagnostic/internal/platform/database"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := database.Open(ctx, database.URLFromEnv())
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err = migrations.Up(ctx, pool); err != nil {
		log.Fatal(err)
	}
	log.Print("Database migrations applied")
}
