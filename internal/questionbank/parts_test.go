package questionbank

import "testing"

// pilot-2.7 is pilot-2.6 without «Мое самочувствие», split into three parts
// (Вики/Тестирование/План разбивки теста на три части.md): every module the
// student answers is in exactly one part; observation (E) is in none.
func TestPilot27HasThreePartsWithoutPS(t *testing.T) {
	bank, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if bank.Version != "pilot-2.7" || len(bank.Parts) != 3 {
		t.Fatalf("version %s, %d parts", bank.Version, len(bank.Parts))
	}
	if _, ok := bank.Modules["PS"]; ok {
		t.Fatal("the module PS is kept")
	}
	for _, indicator := range bank.Indicators {
		if indicator.Block == "psychological" {
			t.Fatalf("indicator %s of PS is kept", indicator.ID)
		}
	}
	modules := map[string]int{}
	for _, item := range bank.Items {
		if item.Module == "PS" {
			t.Fatalf("question %s of PS is kept", item.ID)
		}
		modules[item.Module]++
	}
	in := map[string]string{}
	for _, part := range bank.Parts {
		if part.ID == "" || part.Name == "" {
			t.Fatalf("part without id or name: %+v", part)
		}
		for _, module := range part.Modules {
			if in[module] != "" {
				t.Fatalf("module %s is in %s and %s", module, in[module], part.ID)
			}
			if modules[module] == 0 {
				t.Fatalf("module %s of %s has no questions", module, part.ID)
			}
			in[module] = part.ID
		}
	}
	for module := range modules {
		if module != "E" && in[module] == "" {
			t.Fatalf("module %s is in no part", module)
		}
	}
	if in["E"] != "" {
		t.Fatal("observation is a part of the test")
	}
}

// Old sessions keep their bank: pilot-2.6 with PS stays among the previous.
func TestPreviousKeepsPilot26(t *testing.T) {
	banks, err := Previous()
	if err != nil {
		t.Fatal(err)
	}
	for _, bank := range banks {
		if bank.Version == "pilot-2.6" {
			if _, ok := bank.Modules["PS"]; !ok || len(bank.Parts) != 0 || bank.Climate == nil {
				t.Fatal("pilot-2.6 is not kept as it was")
			}
			return
		}
	}
	t.Fatal("pilot-2.6 is not kept")
}
