package model

import "encoding/json"

// Portrait is the text a language model wrote for one task. It belongs to the
// content, not to a student: equal anonymised input of equal task, task version
// and model gives one record. ID is the hash of those four, see usecase/portraits.go.
type Portrait struct {
	ID          string `json:"id"`
	Task        string `json:"task"`
	TaskVersion string `json:"taskVersion"`
	// Fingerprint is the hash of the input alone.
	Fingerprint  string `json:"fingerprint"`
	Model        string `json:"model"`
	ModelVersion string `json:"modelVersion,omitempty"`
	Created      string `json:"created"`
	// Input is the anonymised JSON exactly as it was sent. It is kept as text:
	// PostgreSQL jsonb would reorder the keys of an object.
	Input   string          `json:"input"`
	Output  json.RawMessage `json:"output"`
	Usage   PortraitUsage   `json:"usage"`
	Seconds float64         `json:"seconds"`
}

type PortraitUsage struct {
	In  int `json:"in"`
	Out int `json:"out"`
}
