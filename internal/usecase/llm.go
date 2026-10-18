package usecase

import (
	"context"
	"encoding/json"
)

// TextModel is the port to a language model. Name is the model from the
// settings: it is part of the key of a stored portrait, so another model gives
// another portrait. An implementation must not put keys or tokens into errors.
type TextModel interface {
	Name() string
	Complete(ctx context.Context, c Completion) (Reply, error)
}

// Completion is one request: the instruction, the data and the schema the
// answer must follow.
type Completion struct {
	System, User string
	Schema       json.RawMessage
}

// Reply is the answer text, the model version that gave it, the tokens spent
// and the seconds it took.
type Reply struct {
	Text, Version string
	In, Out       int
	Seconds       float64
}
