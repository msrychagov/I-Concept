package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

const demoEssayDir = "../../../Вики/Эссе выпускника/Выдуманные эссе"

// demoEssayText is the copy for reading: the questions as the author saw them
// with the answers, and what is meant in the answers about instructors.
func demoEssayText(form model.Form, number int, essay demoEssay) string {
	stage := "во время учёбы"
	if essay.Final {
		stage = "по окончании"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "# Эссе %d. %s, %s\n\nВыдуманное эссе для демо и прогона модели, копия из `project/internal/usecase/demo_essays.go`; руками не править — `go test ./internal/usecase -run TestDemoEssaysCopyInTheWiki -update`.\n", number, essay.Label, stage)
	for _, section := range form.Sections {
		fmt.Fprintf(&out, "\n## %s\n", section.Title)
		for _, field := range section.Fields {
			fmt.Fprintf(&out, "\n**%s. %s**\n\n%s\n", field.Number, field.Label, essay.Answers[field.Number])
		}
	}
	return out.String()
}

func TestDemoEssaysCopyInTheWiki(t *testing.T) {
	form := testApp(t).forms[essayForm]
	if _, err := os.Stat(filepath.Dir(demoEssayDir)); err != nil {
		t.Skip("the wiki folder is not next to the repository")
	}
	if *update {
		if err := os.MkdirAll(demoEssayDir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	order := []string{"a1", "b1", "v1", "a2", "b2", "g2"}
	for n, key := range order {
		for _, essay := range demoEssays {
			if essay.Key != key {
				continue
			}
			stage := "во время учёбы"
			if essay.Final {
				stage = "по окончании"
			}
			path := filepath.Join(demoEssayDir, fmt.Sprintf("%02d %s — %s.md", n+1, essay.Label, stage))
			want := demoEssayText(form, n+1, essay)
			if *update {
				if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != want {
				t.Errorf("%s differs from demo_essays.go; run go test ./internal/usecase -run TestDemoEssaysCopyInTheWiki -update", path)
			}
		}
	}
}
