package questionbank

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

// The methods of the first chapter (Вики/Я-концепция.docx, tables 9, 15–19)
// with their keys: every scale refers to known questions and its levels
// cover the whole range of the sum.
func TestPilot24KeepsTheMethodsAndTheirKeys(t *testing.T) {
	bank, err := load24()
	if err != nil {
		t.Fatal(err)
	}
	if bank.Version != "pilot-2.4" {
		t.Fatalf("version %s", bank.Version)
	}
	items := map[string]model.Item{}
	modules := map[string]int{}
	for _, item := range bank.Items {
		items[item.ID] = item
		modules[item.Module]++
	}
	if modules["W"] != 30 || modules["U"] != 10 || modules["X"] != 18 || modules["K"] != 19 || modules["V"] != 12 {
		t.Fatalf("questions by module: %v", modules)
	}
	want := map[string]int{"VPM": 12, "ZE_V": 24, "ZE_N": 16, "ZE_S": 13, "EMO": 10, "WILL": 18, "DP_A": 9, "DP_B": 4, "DP_C": 6}
	for _, m := range bank.Methods {
		if len(m.Items) != want[m.ID] {
			t.Errorf("%s: %d questions, want %d", m.ID, len(m.Items), want[m.ID])
		}
		low, high := 0, 0
		for _, id := range m.Items {
			item, ok := items[id]
			if !ok || len(item.Options) == 0 || item.Min != 0 || item.Max != len(item.Options)-1 {
				t.Fatalf("%s: question %s has no options of the key", m.ID, id)
			}
			lo, hi := item.Options[0].Score, item.Options[0].Score
			for _, o := range item.Options {
				lo, hi = min(lo, o.Score), max(hi, o.Score)
				if m.Scoring == "level" && o.Level == "" {
					t.Fatalf("%s: option of %s has no level", m.ID, id)
				}
			}
			low, high = low+int(lo), high+int(hi)
		}
		if m.Scoring != "sum" {
			continue
		}
		for sum := low; sum <= high; sum++ {
			found := 0
			for _, band := range m.Levels {
				if sum >= band.Min && sum <= band.Max {
					found++
				}
			}
			if found != 1 {
				t.Fatalf("%s: sum %d falls into %d levels", m.ID, sum, found)
			}
		}
	}
	for _, k := range bank.ActivityKinds {
		for _, id := range k.Items {
			if _, ok := items[id]; !ok {
				t.Fatalf("%s: unknown question %s", k.ID, id)
			}
		}
	}
}

func TestPreviousKeepsPilot23And24(t *testing.T) {
	banks, err := Previous()
	if err != nil {
		t.Fatal(err)
	}
	kept := map[string]bool{}
	for _, bank := range banks {
		kept[bank.Version] = true
		if bank.Version == "pilot-2.4" && (bank.Identity != nil || len(bank.Methods) == 0) {
			t.Fatal("pilot-2.4 is not kept as it was")
		}
	}
	if !kept["pilot-2.3"] || !kept["pilot-2.4"] {
		t.Fatalf("versions kept for old sessions: %v", kept)
	}
}
