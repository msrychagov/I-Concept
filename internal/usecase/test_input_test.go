package usecase

import (
	"hash/fnv"
	"os"
	"path/filepath"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

// fixedTestSession answers every question of the bank version by a hash of
// its code, so the session is the same on every run.
func fixedTestSession(a *Service, version string) model.Session {
	s := model.Session{ID: "fixed", Kind: "student", BankVersion: version, Methods: true, Activity: true, Answers: map[string]int{}, Notes: map[string]string{}}
	for _, item := range a.itemsFor(s) {
		h := fnv.New32a()
		h.Write([]byte(item.ID))
		n := h.Sum32()
		switch {
		case item.Type == "textarea":
			if n%5 == 0 {
				s.Answers[item.ID] = 0
				s.Notes[item.ID] = "Участвовал в мероприятии " + item.ID + "."
			}
		case len(item.Options) > 0:
			s.Answers[item.ID] = int(n % uint32(len(item.Options)))
		default:
			s.Answers[item.ID] = item.Min + int(n%uint32(item.Max-item.Min+1))
		}
	}
	return s
}

// The input of the instruction v1 must not change: the stored portraits are
// found by its fingerprint. A session of pilot-2.4 and one of the current bank
// with the same answers give the same input. With -update the file is rewritten.
func TestTestPayloadV1IsKept(t *testing.T) {
	a := testApp(t)
	path := filepath.Join("testdata", "test_input", "v1.json")
	for _, version := range []string{"pilot-2.4", a.bank.Version} {
		data := a.TestPortrait(fixedTestSession(a, version))
		if data == nil {
			t.Fatalf("%s: no portrait", version)
		}
		got := indentedJSON(testPayload(data))
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s: the input of v1 changed", version)
		}
	}
}
