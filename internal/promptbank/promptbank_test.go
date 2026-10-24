package promptbank

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAutobiographyV1(t *testing.T) {
	prompts, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	prompt, ok := Find(prompts, "autobiography", "v1")
	if !ok {
		t.Fatal("autobiography v1 is not embedded")
	}
	if !strings.HasPrefix(prompt.Instruction, "Ты помогаешь преподавателю военного учебного центра") || !strings.HasSuffix(prompt.Instruction, "Пиши по-русски.") {
		t.Fatalf("instruction: %.80q…%q", prompt.Instruction, prompt.Instruction[len(prompt.Instruction)-20:])
	}
	if !strings.Contains(string(prompt.Schema), `"required": ["summary", "potential", "attention"]`) {
		t.Fatalf("schema: %s", prompt.Schema)
	}
}

// The latest version of each task is the text of its instruction in the wiki
// word for word; the wiki is in the working folder, not in the repository. The
// first version of the test keeps its own file as the history of v1.
func TestLatestVersionMatchesTheWikiWhenItIsThere(t *testing.T) {
	prompts, _ := Load()
	for _, c := range []struct{ task, version, path string }{
		{"autobiography", "", "../../../Вики/Автобиография/Сравнение моделей/Инструкция.md"},
		{"test", "", "../../../Вики/Тестирование/Инструкция v3.md"},
		{"test", "v2", "../../../Вики/Тестирование/Инструкция v2.md"},
		{"test", "v1", "../../../Вики/Тестирование/Инструкция.md"},
		{"test-platoon", "", "../../../Вики/Тестирование/Инструкция по взводу.md"},
		{"essay", "", "../../../Вики/Эссе выпускника/Инструкция.md"},
		{"essay-cohort", "", "../../../Вики/Эссе выпускника/Инструкция по выпуску.md"},
	} {
		t.Run(c.task+c.version, func(t *testing.T) {
			wiki, err := os.ReadFile(c.path)
			if err != nil {
				t.Skip("the wiki folder is not next to the repository")
			}
			prompt, ok := Latest(prompts, c.task)
			if c.version != "" {
				prompt, ok = Find(prompts, c.task, c.version)
			}
			if !ok || !strings.Contains(string(wiki), "```text\n"+prompt.Instruction+"\n```") || !strings.Contains(string(wiki), "```json\n"+string(prompt.Schema)+"\n```") {
				t.Errorf("%s: version %s differs from %s: a changed wording must be a new version folder", c.task, prompt.Version, c.path)
			}
		})
	}
}

func TestEssayPromptsAreEmbedded(t *testing.T) {
	prompts, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for task, want := range map[string]string{"essay": `"required": ["summary", "moments", "positive", "negative", "attention", "teachers", "change", "conclusion"]`, "essay-cohort": `"required": ["overview", "moments", "positive", "negative", "attention", "paths", "environment", "conclusion", "change"]`} {
		prompt, ok := Latest(prompts, task)
		if !ok || prompt.Version != "v1" || !strings.HasSuffix(prompt.Instruction, "Пиши по-русски, деловым языком.") || !strings.Contains(string(prompt.Schema), want) {
			t.Errorf("%s: %+v", task, prompt.Version)
		}
	}
}

func TestParseRejectsBrokenPrompts(t *testing.T) {
	for name, files := range map[string]fstest.MapFS{
		"empty instruction": {"t/v1/instruction.txt": {Data: []byte(" \n")}, "t/v1/schema.json": {Data: []byte(`{"type":"object"}`)}},
		"no schema":         {"t/v1/instruction.txt": {Data: []byte("text")}},
		"broken schema":     {"t/v1/instruction.txt": {Data: []byte("text")}, "t/v1/schema.json": {Data: []byte(`[1`)}},
		"schema not object": {"t/v1/instruction.txt": {Data: []byte("text")}, "t/v1/schema.json": {Data: []byte(`{}`)}},
	} {
		if _, err := parse(files); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestLatestIsTheGreatestNumberNotTheLastName(t *testing.T) {
	files := fstest.MapFS{}
	for _, version := range []string{"v1", "v2", "v10", "v9"} {
		files["t/"+version+"/instruction.txt"] = &fstest.MapFile{Data: []byte("text " + version)}
		files["t/"+version+"/schema.json"] = &fstest.MapFile{Data: []byte(`{"type":"object"}`)}
	}
	files["other/v3/instruction.txt"] = &fstest.MapFile{Data: []byte("other")}
	files["other/v3/schema.json"] = &fstest.MapFile{Data: []byte(`{"type":"object"}`)}
	prompts, err := parse(files)
	if err != nil {
		t.Fatal(err)
	}
	if latest, ok := Latest(prompts, "t"); !ok || latest.Version != "v10" || latest.Instruction != "text v10" {
		t.Fatalf("latest: %+v", latest)
	}
	if latest, ok := Latest(prompts, "other"); !ok || latest.Version != "v3" {
		t.Fatalf("other: %+v", latest)
	}
	if _, ok := Latest(prompts, "missing"); ok {
		t.Fatal("a task without prompts has a latest")
	}
}

func TestVersionFoldersMustBeNamedVN(t *testing.T) {
	for _, version := range []string{"v0", "v01", "latest", "2", "v1b"} {
		files := fstest.MapFS{
			"t/" + version + "/instruction.txt": {Data: []byte("text")},
			"t/" + version + "/schema.json":     {Data: []byte(`{"type":"object"}`)},
		}
		if _, err := parse(files); err == nil {
			t.Errorf("%s accepted", version)
		}
	}
}
