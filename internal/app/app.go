// Package app is the composition root: it wires infrastructure to application
// ports and exposes the HTTP handler and explicit maintenance commands.
package app

import (
	"encoding/json"
	"net/http"
	"vucdiagnostic/internal/formbank"
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
	forms, err := formbank.Load()
	if err != nil {
		return nil, err
	}
	service := usecase.New(bank, repository, previous...)
	service.RegisterForms(forms...)
	if err := service.NormalizeOwnership(); err != nil {
		return nil, err
	}
	handler := httptransport.New(service, web.Files, httptransport.Config{AdminUser: config.AdminUser, AdminPassword: config.AdminPassword})
	return &App{Handler: handler, service: service}, nil
}
func (a *App) ResetDemo() (string, error) { return a.service.ResetDemo() }

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
