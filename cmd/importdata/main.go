// Importdata transfers an existing runtime JSON file into an empty SQL database.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"vucdiagnostic/internal/app"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/platform/database"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	path := flag.String("file", "", "existing runtime JSON file (required)")
	flag.Parse()
	if *path == "" {
		return fmt.Errorf("-file is required")
	}
	dsn := database.URLFromEnv()
	if dsn == "" {
		return fmt.Errorf("import requires DATABASE_URL or DB_HOST")
	}
	file, err := os.Open(*path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var db model.Database
	if err := decoder.Decode(&db); err != nil {
		return fmt.Errorf("decode source JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("source must contain exactly one JSON object")
	}
	a, err := app.New(app.Config{DatabaseURL: dsn})
	if err != nil {
		return err
	}
	defer a.Close()
	if err := a.ImportData(db); err != nil {
		return err
	}
	log.Print("Runtime data imported successfully; source JSON unchanged")
	return nil
}
