package usecase

import "encoding/json"

// Input contains application command fields assembled by transport adapters.
type Input map[string]json.RawMessage

func str(b Input, k string) string   { var v string; _ = json.Unmarshal(b[k], &v); return v }
func integer(b Input, k string) int  { var v int; _ = json.Unmarshal(b[k], &v); return v }
func boolean(b Input, k string) bool { var v bool; _ = json.Unmarshal(b[k], &v); return v }
