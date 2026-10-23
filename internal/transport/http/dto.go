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
	Invite   string `json:"invite"`
}
type teacherInvitationRequest struct {
	VUC   string `json:"vuc"`
	Email string `json:"email"`
}
type studentRegistrationLinkRequest struct {
	PlatoonID string `json:"platoonId"`
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
	FormID     string `json:"formId"`
	CohortID   string `json:"cohortId"`
	Platoon    string `json:"platoon"`
	Activity   bool   `json:"activity"`
	Motivation bool   `json:"motivation"`
	Methods    bool   `json:"methods"`
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
type sectionRequest struct {
	Answers  map[string]json.RawMessage `json:"answers"`
	Revision *int                       `json:"revision"`
}
type formCompletionRequest struct {
	Revision *int `json:"revision"`
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
