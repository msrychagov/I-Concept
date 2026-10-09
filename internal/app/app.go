// Package app is the composition root: it wires infrastructure to application
// ports and exposes the HTTP handler and explicit maintenance commands.
package app

import (
	"net/http"
	"vucdiagnostic/internal/questionbank"
	"vucdiagnostic/internal/repository/jsonfile"
	httptransport "vucdiagnostic/internal/transport/http"
	"vucdiagnostic/internal/usecase"
	"vucdiagnostic/web"
)

type Config struct{ DataFile, AdminUser, AdminPassword string }
type App struct {
	http.Handler
	service *usecase.Service
}

var _ usecase.Repository = (*jsonfile.Repository)(nil)

func New(config Config) (*App, error) {
	bank, err := questionbank.Load()
	if err != nil {
		return nil, err
	}
	repository, err := jsonfile.Open(config.DataFile)
	if err != nil {
		return nil, err
	}
	previous, err := questionbank.Previous()
	if err != nil {
		return nil, err
	}
	service := usecase.New(bank, repository, previous...)
	handler := httptransport.New(service, web.Files, httptransport.Config{AdminUser: config.AdminUser, AdminPassword: config.AdminPassword})
	return &App{Handler: handler, service: service}, nil
}
func (a *App) ResetDemo() (string, error) { return a.service.ResetDemo() }
