package model

import "encoding/json"

// Clone creates an isolated copy of JSON-persisted domain values.
func Clone[T any](value T) T {
	data, _ := json.Marshal(value)
	var copy T
	_ = json.Unmarshal(data, &copy)
	return copy
}
