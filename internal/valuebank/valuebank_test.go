package valuebank

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// The copy of the program is the file of the wiki byte for byte; the wiki is in
// the working folder, not in the repository.
func TestCopyIsTheFileOfTheWiki(t *testing.T) {
	wiki, err := os.ReadFile("../../../Вики/Онтограф ценностей 1.1/онтограф_ценности1.1_nodes.json")
	if err != nil {
		t.Skip("the wiki folder is not next to the repository")
	}
	if !bytes.Equal(wiki, file) {
		t.Fatal("internal/valuebank/ontograph-1.1.json differs from the file of the wiki: copy it again")
	}
}

func TestOntographHasWhatTheCountingStandsOn(t *testing.T) {
	o, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for class, want := range map[string]int{Idea: 5, Value: 17, Aspect: 178, Antivalue: 17, Everyday: 8, EverydayAnti: 44} {
		if got := len(o.Of(class)); got != want {
			t.Errorf("%s: %d nodes, want %d", class, got, want)
		}
	}
	ideas := []string{}
	for _, idea := range o.Of(Idea) {
		ideas = append(ideas, idea.ID)
	}
	if want := "pravda rodina vera my volya"; strings.Join(ideas, " ") != want {
		t.Errorf("ideas: %v, want the order of the formula %s", ideas, want)
	}
	for id, want := range map[string]struct {
		target   string
		everyday bool
		idea     string
	}{
		"v_aid":      {"v_aid", false, "my"},
		"aid_help":   {"v_aid", false, "my"},
		"honest":     {"v_fam", false, "my"},  // two values in the graph, one parent as the rules say
		"freedom":    {"v_lab", false, "volya"},
		"ev_craft":   {"v_lab", true, "volya"},
		"av_col":     {"av_col", false, "my"},
		"ev_a_col_alone": {"av_cit", true, "rodina"},
	} {
		target, everyday := o.Counted(id)
		if target != want.target || everyday != want.everyday || o.IdeaOf(target) != want.idea {
			t.Errorf("%s: counts for %s (everyday %v, idea %s)", id, target, everyday, o.IdeaOf(target))
		}
	}
	if o.Coded("pravda") || o.Coded("th_west") || !o.Coded("ev_a_life_joke") {
		t.Error("an idea or a threat may be coded")
	}
}
