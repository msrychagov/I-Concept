package questionbank

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// pilot-2.6 adds the express method of the climate of the platoon by
// А. Н. Лутошкин: fourteen pairs of opposite statements, seven options from
// +3 to −3, no indicator and no levels. pilot-2.5 is kept for old sessions.
func TestPilot26HasTheClimateMethod(t *testing.T) {
	bank, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if bank.Version != "pilot-2.6" || bank.Climate == nil || bank.Identity == nil || len(bank.Methods) == 0 {
		t.Fatalf("version %s, climate %v", bank.Version, bank.Climate != nil)
	}
	if len(bank.Climate.Pairs) != 14 || bank.Modules["CL"] != "Климат взвода" || bank.Instructions["CL"] == "" {
		t.Fatalf("climate: %+v", bank.Climate)
	}
	items, module := map[string]int{}, 0
	for n, item := range bank.Items {
		items[item.ID] = n
		if item.Module == "CL" {
			module++
		}
	}
	if module != 14 {
		t.Fatalf("%d questions in the module", module)
	}
	for n, pair := range bank.Climate.Pairs {
		at, ok := items[pair.Item]
		if !ok || pair.Item != fmt.Sprintf("CL%02d", n+1) || pair.Side == "" {
			t.Fatalf("pair %d: %+v", n+1, pair)
		}
		item := bank.Items[at]
		if item.Module != "CL" || item.Type != "pair" || item.Text == "" || item.Opposite == "" || item.Indicator != "" || item.Min != 0 || item.Max != 6 || len(item.Options) != 7 {
			t.Fatalf("%s: %+v", item.ID, item)
		}
		for k, o := range item.Options {
			if int(o.Score) != 3-k {
				t.Fatalf("%s: option %d gives %v", item.ID, k, o.Score)
			}
		}
	}
	banks, err := Previous()
	if err != nil {
		t.Fatal(err)
	}
	for _, old := range banks {
		if old.Version == "pilot-2.5" && (old.Climate != nil || old.Identity == nil) {
			t.Fatal("pilot-2.5 is not kept as it was")
		}
	}
}

// The pairs, the sides, the options and the instruction are those of the
// approved document word for word.
func TestClimateMethodIsTheApprovedText(t *testing.T) {
	data, err := os.ReadFile("../../../Вики/Тестирование/Лутошкин — адаптация для ВУЦ.md")
	if err != nil {
		t.Skip("the wiki folder is not next to the repository")
	}
	doc := string(data)
	bank, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	items := map[string]int{}
	for n, item := range bank.Items {
		items[item.ID] = n
	}
	if !strings.Contains(strings.Join(strings.Fields(strings.ReplaceAll(doc, "\n> ", " ")), " "), bank.Instructions["CL"]) {
		t.Error("the instruction differs from the document")
	}
	for n, pair := range bank.Climate.Pairs {
		item := bank.Items[items[pair.Item]]
		row := fmt.Sprintf("| %d | %s | %s | %s |", n+1, item.Text, item.Opposite, pair.Side)
		if !strings.Contains(doc, row) {
			t.Errorf("pair %d differs from the document", n+1)
		}
	}
	for _, o := range bank.Items[items["CL01"]].Options {
		score := fmt.Sprintf("%+d", int(o.Score))
		if o.Score == 0 {
			score = "0"
		}
		if !strings.Contains(doc, "| "+o.Text+" | "+strings.Replace(score, "-", "−", 1)+" |") {
			t.Errorf("option «%s» differs from the document", o.Text)
		}
	}
}
