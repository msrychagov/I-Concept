package questionbank

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

// pilot-2.5 adds the method of А. А. Азбель in two blocks of twenty questions
// with one key: every option gives one or two points to one of four scales,
// every scale gets at most 25, and its five ranges start at zero.
func TestPilot25HasTheIdentityMethod(t *testing.T) {
	bank, err := load25()
	if err != nil {
		t.Fatal(err)
	}
	if bank.Version != "pilot-2.5" || bank.Identity == nil || len(bank.Methods) == 0 {
		t.Fatalf("version %s, identity %v", bank.Version, bank.Identity != nil)
	}
	identity := bank.Identity
	if len(identity.Ranges) != 5 || len(identity.Scales) != 4 || len(identity.Blocks) != 2 {
		t.Fatalf("identity: %+v", identity)
	}
	scales := map[string]bool{}
	for _, scale := range identity.Scales {
		scales[scale.ID] = true
		if len(scale.Bounds) != len(identity.Ranges) || scale.Bounds[0] != 0 || !slices.IsSorted(scale.Bounds) {
			t.Errorf("%s: bounds %v", scale.ID, scale.Bounds)
		}
	}
	items := map[string]model.Item{}
	for _, item := range bank.Items {
		items[item.ID] = item
	}
	var key [][]model.Option
	for b, block := range identity.Blocks {
		if len(block.Items) != 20 || bank.Modules[items[block.Items[0]].Module] != block.Name {
			t.Fatalf("%s: %d questions", block.ID, len(block.Items))
		}
		highest := map[string]int{}
		for n, id := range block.Items {
			item, ok := items[id]
			if !ok || len(item.Options) != 4 || item.Min != 0 || item.Max != 3 || item.Type != "choice" || item.Indicator != "" || item.Module != items[block.Items[0]].Module {
				t.Fatalf("%s: question %s", block.ID, id)
			}
			seen := map[string]bool{}
			for _, o := range item.Options {
				if !scales[o.Scale] || seen[o.Scale] || o.Score < 1 || o.Score > 2 {
					t.Fatalf("%s: option %s of %s", block.ID, o.ID, id)
				}
				seen[o.Scale] = true
				highest[o.Scale] += int(o.Score)
			}
			if b == 0 {
				key = append(key, item.Options)
				continue
			}
			for k, o := range item.Options {
				if o.Scale != key[n][k].Scale || o.Score != key[n][k].Score {
					t.Fatalf("%s: the key of %s differs from the first block", block.ID, id)
				}
			}
		}
		for scale := range scales {
			if highest[scale] != 25 {
				t.Errorf("%s: %s gets at most %d", block.ID, scale, highest[scale])
			}
		}
	}
}

// The questions, the options, the key and the instruction are those of the
// approved document word for word.
func TestIdentityMethodIsTheApprovedText(t *testing.T) {
	data, err := os.ReadFile("../../../Вики/Тестирование/Азбель — адаптация для ВУЦ.md")
	if err != nil {
		t.Skip("the wiki folder is not next to the repository")
	}
	doc := string(data)
	bank, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	items := map[string]model.Item{}
	for _, item := range bank.Items {
		items[item.ID] = item
	}
	short := map[string]string{}
	for _, scale := range bank.Identity.Scales {
		short[scale.ID] = scale.Short
		if !strings.Contains(doc, scale.Short+" — "+scale.Name) {
			t.Errorf("scale %s is not named as in the document", scale.ID)
		}
		row := "| " + scale.Short + " |"
		for n, bound := range scale.Bounds {
			if n == len(scale.Bounds)-1 {
				row += fmt.Sprintf(" %d+ |", bound)
			} else {
				row += fmt.Sprintf(" %d–%d |", bound, scale.Bounds[n+1]-1)
			}
		}
		if !strings.Contains(doc, row) {
			t.Errorf("ranges of %s: %s", scale.ID, row)
		}
	}
	if !strings.Contains(doc, "| Шкала | "+strings.Join(capitalized(bank.Identity.Ranges), " | ")+" |") {
		t.Errorf("names of the ranges: %v", bank.Identity.Ranges)
	}
	letters := []string{"а", "б", "в", "г"}
	for b, block := range bank.Identity.Blocks {
		title := fmt.Sprintf("## Блок %d. %s\n", b+1, block.Name)
		start := strings.Index(doc, title)
		if start < 0 {
			t.Fatalf("no block %s in the document", block.Name)
		}
		section := doc[start+len(title):]
		if end := strings.Index(section, "\n## "); end >= 0 {
			section = section[:end]
		}
		module := items[block.Items[0]].Module
		for _, text := range strings.Split(bank.Instructions[module], ". ") {
			if !strings.Contains(strings.Join(strings.Fields(doc), " "), strings.TrimSuffix(text, ".")) {
				t.Errorf("%s: instruction «%s» is not in the document", block.ID, text)
			}
		}
		for n, id := range block.Items {
			item := items[id]
			question := fmt.Sprintf("%d. %s\n", n+1, item.Text)
			at := strings.Index(section, question)
			if at < 0 {
				t.Fatalf("%s: question %d differs", block.ID, n+1)
			}
			rest := section[at+len(question):]
			row := fmt.Sprintf("| %d |", n+1)
			for k, o := range item.Options {
				line := letters[k] + ") " + o.Text + "\n"
				if !strings.HasPrefix(strings.TrimLeft(rest, " "), line) {
					t.Fatalf("%s: option %s of question %d differs", block.ID, letters[k], n+1)
				}
				rest = strings.TrimLeft(rest, " ")[len(line):]
				row += fmt.Sprintf(" %s %d |", short[o.Scale], int(o.Score))
			}
			if !strings.Contains(doc, row+"\n") {
				t.Fatalf("key of question %d: %s", n+1, row)
			}
		}
	}
}

func capitalized(words []string) []string {
	result := []string{}
	for _, word := range words {
		r := []rune(word)
		result = append(result, strings.ToUpper(string(r[0]))+string(r[1:]))
	}
	return result
}
