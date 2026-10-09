package usecase

type ErrorKind string

const (
	Invalid         ErrorKind = "invalid"
	Unauthenticated ErrorKind = "unauthenticated"
	Forbidden       ErrorKind = "forbidden"
	NotFound        ErrorKind = "not_found"
	Conflict        ErrorKind = "conflict"
	Gone            ErrorKind = "gone"
	Incomplete      ErrorKind = "incomplete"
)

type Error struct {
	Kind    ErrorKind
	Details map[string]any
}

func (e *Error) Error() string { s, _ := e.Details["error"].(string); return s }
func failure(kind ErrorKind, details map[string]any) error {
	return &Error{Kind: kind, Details: details}
}
func problem(kind ErrorKind, message string) error {
	return failure(kind, map[string]any{"error": message})
}
