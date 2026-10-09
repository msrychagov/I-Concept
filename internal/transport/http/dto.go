package httptransport

import (
	"encoding/json"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/usecase"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Platoon  string `json:"platoon,omitempty"`
}
type profileRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	CurrentPassword string `json:"currentPassword,omitempty"`
}
type passwordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}
type assignmentRequest struct {
	Title      string `json:"title"`
	ProfileID  string `json:"profileId"`
	Platoon    string `json:"platoon"`
	Activity   bool   `json:"activity"`
	Motivation bool   `json:"motivation"`
}
type assignmentStatusRequest struct {
	Active *bool `json:"active"`
}
type consentRequest struct {
	Value    *bool `json:"value"`
	Revision *int  `json:"revision"`
}
type answerRequest struct {
	Value    json.RawMessage `json:"value"`
	Note     string          `json:"note"`
	Revision *int            `json:"revision"`
}
type completionRequest struct {
	AllowPartial bool `json:"allowPartial"`
	Revision     *int `json:"revision"`
}
type planRequest struct {
	Plan []model.PlanItem `json:"plan"`
}

// Translation into application commands is contained in the transport boundary.
func commandInput(value any) usecase.Input {
	data, _ := json.Marshal(value)
	var input usecase.Input
	_ = json.Unmarshal(data, &input)
	return input
}
func setField(input usecase.Input, key string, value any) { input[key], _ = json.Marshal(value) }
