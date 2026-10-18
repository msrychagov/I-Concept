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

// v1 is the text of Вики/Автобиография/Сравнение моделей/Инструкция.md word for
// word; the file is in the working folder, not in the repository.
func TestV1MatchesTheWikiWhenItIsThere(t *testing.T) {
	wiki, err := os.ReadFile("../../../Вики/Автобиография/Сравнение моделей/Инструкция.md")
	if err != nil {
		t.Skip("the wiki folder is not next to the repository")
	}
	prompts, _ := Load()
	prompt, _ := Find(prompts, "autobiography", "v1")
	if !strings.Contains(string(wiki), "```text\n"+prompt.Instruction+"\n```") || !strings.Contains(string(wiki), "```json\n"+string(prompt.Schema)+"\n```") {
		t.Fatal("v1 differs from Инструкция.md: a changed wording must be a new version folder")
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
