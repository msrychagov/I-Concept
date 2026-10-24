// Package app is the composition root: it wires infrastructure to application
// ports and exposes the HTTP handler and explicit maintenance commands.
package app

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/formbank"
	"vucdiagnostic/internal/platform/database"
	"vucdiagnostic/internal/promptbank"
	"vucdiagnostic/internal/questionbank"
	"vucdiagnostic/internal/repository/jsonfile"
	"vucdiagnostic/internal/repository/postgres"
	httptransport "vucdiagnostic/internal/transport/http"
	"vucdiagnostic/internal/usecase"
	"vucdiagnostic/web"
)

type Config struct{ DataFile, DatabaseURL, AdminUser, AdminPassword string }
type App struct {
	http.Handler
	service *usecase.Service
	close   func()
}

var _ usecase.Repository = (*jsonfile.Repository)(nil)
var _ usecase.Repository = (*postgres.Repository)(nil)

func New(config Config) (*App, error) {
	bank, err := questionbank.Load()
	if err != nil {
		return nil, err
	}
	previous, err := questionbank.Previous()
	if err != nil {
		return nil, err
	}
	forms, err := formbank.Load()
	if err != nil {
		return nil, err
	}
	prompts, err := promptbank.Load()
	if err != nil {
		return nil, err
	}
	defender, err := questionbank.Defender()
	if err != nil {
		return nil, err
	}
	var repository usecase.Repository
	closeRepository := func() {}
	if config.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pool, openErr := database.Open(ctx, config.DatabaseURL)
		if openErr != nil {
			return nil, openErr
		}
		closeRepository = pool.Close
		repository, err = postgres.New(ctx, pool)
	} else {
		repository, err = jsonfile.Open(config.DataFile)
	}
	if err != nil {
		closeRepository()
		return nil, err
	}
	service := usecase.New(bank, repository, previous...)
	service.RegisterForms(forms...)
	service.RegisterPrompts(prompts...)
	service.RegisterDefender(defender)
	// PostgreSQL receives legacy records through ImportData, which normalizes
	// ownership before the import transaction. Never replace a live SQL snapshot.
	if config.DatabaseURL == "" {
		if err := service.NormalizeOwnership(); err != nil {
			closeRepository()
			return nil, err
		}
	}
	handler := httptransport.New(service, web.Files, httptransport.Config{AdminUser: config.AdminUser, AdminPassword: config.AdminPassword})
	return &App{Handler: handler, service: service, close: closeRepository}, nil
}
func (a *App) Close() {
	a.service.Close()
	a.close()
}

// UseModel connects the language model that writes the portraits. It is called
// only by the server: the maintenance commands must not start it.
func (a *App) UseAnalysisModel(m usecase.TextModel) { a.service.UseAnalysisModel(m) }
func (a *App) UseModel(m usecase.TextModel)         { a.service.UseModel(m) }
func (a *App) SeedDemo() error                      { return a.service.SeedDemo() }
func (a *App) ImportData(db model.Database) error   { return a.service.ImportData(db) }
func (a *App) ResetDemo() (string, error)           { return a.service.ResetDemo() }

func (a *App) CreateVUC(name string) (any, error) { return a.service.CreateVUC(name) }

func (a *App) InviteTeacher(email, vuc string) (map[string]any, error) {
	data, _ := json.Marshal(map[string]string{"email": email, "vuc": vuc})
	var input usecase.Input
	_ = json.Unmarshal(data, &input)
	value, err := a.service.CreateTeacherInvitation(input)
	if err != nil {
		return nil, err
	}
	return value.(map[string]any), nil
}
