package usecase

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// inputTexts are the keys and the texts of an input but the words of the
// students themselves.
func inputTexts(value any, stories bool, texts *[]string) {
	switch v := value.(type) {
	case *object:
		for _, key := range v.keys {
			*texts = append(*texts, key)
			if studentKeys[key] || stories && key == "ответ" {
				continue
			}
			inputTexts(v.values[key], stories || key == "рассказы_о_деятельности", texts)
		}
	case []any:
		for _, item := range v {
			inputTexts(item, stories, texts)
		}
	case string:
		*texts = append(*texts, v)
	}
}

// demoPlatoons are the demo platoons 102 and 104 with the methods.
func demoPlatoons(t *testing.T, a *Service) map[string][]studentPortrait {
	t.Helper()
	db, err := a.demoDatabase()
	if err != nil {
		t.Fatal(err)
	}
	teacherID := ""
	for _, account := range db.Students {
		if account.Role == "teacher" {
			teacherID = account.ID
		}
	}
	platoons := map[string][]studentPortrait{}
	for _, group := range []string{"102", "104"} {
		platoons[group] = a.platoonStudents(db, teacherID, group)
		if len(platoons[group]) == 0 {
			t.Fatalf("demo platoon %s: %d students", group, len(platoons[group]))
		}
	}
	return platoons
}

// The input of test/v6 and test-platoon/v4 has none of the words the check
// of the answer forbids (decision Д9), nor the dictionary.
func TestPlainInputHasNoTerms(t *testing.T) {
	a := testApp(t)
	payloads := map[string]*object{
		"fixed":         a.testPayloadFor("v6", a.TestPortrait(fixedTestSession(a, a.bank.Version))),
		"fixed platoon": a.platoonPayloadFor("v4", a.platoonPortrait("101", fixedPlatoon(a, a.bank.Version))),
	}
	for group, portraits := range demoPlatoons(t, a) {
		payloads["platoon "+group] = a.platoonPayloadFor("v4", a.platoonPortrait(group, portraits))
		for _, p := range portraits {
			payloads[p.student.Name] = a.testPayloadFor("v6", p.data)
		}
	}
	for name, payload := range payloads {
		if _, ok := payload.values["словарь"]; ok {
			t.Errorf("%s: the dictionary is in the input", name)
		}
		texts := []string{}
		inputTexts(payload, false, &texts)
		for _, text := range texts {
			if problems := termProblems("input", strings.ReplaceAll(text, "_", " ")); len(problems) > 0 {
				t.Errorf("%s: %q: %v", name, text, problems)
			}
		}
	}
}

// The words of the student himself stay as he wrote them.
func TestPlainInputKeepsTheWordsOfTheStudent(t *testing.T) {
	a := testApp(t)
	s := fixedTestSession(a, a.bank.Version)
	s.Notes["ACT_18"], s.Answers["ACT_18"] = "Чувство патриотизма и любовь к Родине — мой компонент.", 0
	payload := a.testPayloadFor("v6", a.TestPortrait(s))
	if !strings.Contains(string(canonicalJSON(payload)), "Чувство патриотизма и любовь к Родине — мой компонент.") {
		t.Fatal("the story of the student changed")
	}
}

func TestPlainInputsAreKept(t *testing.T) {
	a := testApp(t)
	keptInput(t, "v6.json", indentedJSON(a.testPayloadFor("v6", a.TestPortrait(fixedTestSession(a, a.bank.Version)))))
	keptInput(t, "platoon-v4.json", indentedJSON(a.platoonPayloadFor("v4", a.platoonPortrait("101", fixedPlatoon(a, a.bank.Version)))))
}

// The input of v6: plain keys and groups with their qualities, the
// conditions rated in words, the meaning of the leading status, the kinds
// without the lists their criteria have.
func TestPlainTestPayload(t *testing.T) {
	a := testApp(t)
	data := a.TestPortrait(fixedTestSession(a, a.bank.Version))
	payload := a.testPayloadFor("v6", data)
	for _, key := range []string{"методики", "образы_я", "отношения", "образ_защитника", "профессиональная_идентичность", "условия_среды", "среда_по_главе_1", "виды_деятельности", "каталог_рекомендаций"} {
		if _, ok := payload.values[key]; !ok {
			t.Errorf("no %s in %v", key, payload.keys)
		}
	}
	first := objectList(payload.values["образ_защитника"])[0]
	if objectText(first, "группа") != "верность долгу и преданность Отечеству" || !strings.HasPrefix(objectText(first, "качества"), "верность долгу и присяге") {
		t.Fatalf("group: %s", canonicalJSON(first))
	}
	if _, ok := first.values["какой_сейчас"]; !ok {
		t.Fatalf("group keys: %v", first.keys)
	}
	words := map[string]bool{}
	for _, level := range objectList(payload.values["условия_среды"]) {
		for _, c := range objectList(level.values["показатели"]) {
			if _, ok := c.values["балл"]; ok {
				t.Errorf("a score of a condition: %s", canonicalJSON(c))
			}
			words[objectText(c, "оценка")] = true
		}
	}
	if !words[ratedHigher] || !words[ratedLower] {
		t.Fatalf("ratings: %v", words)
	}
	for _, block := range objectList(payload.values["профессиональная_идентичность"]) {
		if objectText(block, "что_значит") == "" {
			t.Errorf("no meaning: %s", canonicalJSON(block))
		}
	}
	for _, kind := range objectList(payload.values["виды_деятельности"]) {
		covered := map[string]bool{}
		for _, c := range objectList(kind.values["критерии"]) {
			for _, point := range objectList(c.values["данные"]) {
				covered[objectText(point, "вопрос_анкеты")+objectText(point, "показатель")+objectText(point, "действие")] = true
			}
		}
		for _, list := range []string{"ответы_анкеты", "показатели", "частота_действий"} {
			for _, item := range objectList(kind.values[list]) {
				if covered[objectText(item, "название")+objectText(item, "показатель")+objectText(item, "действие")] {
					t.Errorf("%s: %s repeats a criterion", objectText(kind, "вид"), canonicalJSON(item))
				}
			}
		}
	}
	if n, old := len(canonicalJSON(payload)), len(canonicalJSON(a.testPayloadFor("v5", data))); n >= old {
		t.Fatalf("v6 is not shorter: %d, v5 %d", n, old)
	}
}

func TestConditionWords(t *testing.T) {
	got := conditionWords(map[string]float64{"a": 5, "b": 4.5, "c": 4.5, "d": 4, "e": 3.5, "f": 3, "g": 3, "h": 2.5, "i": 2})
	want := map[string]string{"a": ratedHigher, "b": ratedHigher, "c": ratedHigher, "d": ratedEven, "e": ratedEven,
		"f": ratedLower, "g": ratedLower, "h": ratedLower, "i": ratedInsufficient}
	if !maps.Equal(got, want) {
		t.Fatalf("words: %v", got)
	}
	if got := conditionWords(map[string]float64{"a": 3, "b": 3}); got["a"] != ratedEven || got["b"] != ratedEven {
		t.Fatalf("equal: %v", got)
	}
	if got := conditionWords(map[string]float64{"a": 4, "b": 3}); got["a"] != ratedHigher || got["b"] != ratedLower {
		t.Fatalf("two: %v", got)
	}
}

// The groups of students for separate work by the rules of Д1: every group
// has students, bases, codes of the catalogue; the same students make one
// group; the input has the first five without names.
func TestWorkGroupsOfTheDemoPlatoons(t *testing.T) {
	a := testApp(t)
	for group, portraits := range demoPlatoons(t, a) {
		data := a.platoonPortrait(group, portraits)
		if len(data.WorkGroups) == 0 {
			t.Fatalf("%s: no groups", group)
		}
		seen := map[string]bool{}
		for n, g := range data.WorkGroups {
			ids := []string{}
			for _, s := range g.Students {
				ids = append(ids, s.SessionID)
			}
			key := strings.Join(ids, "/")
			if seen[key] || len(g.Students) == 0 || len(g.Bases) == 0 || len(g.Codes) == 0 || g.Label != workGroupLabels[n] || g.Model != (n < workGroupModel) {
				t.Errorf("%s: group %d: %+v", group, n, g)
			}
			seen[key] = true
		}
		payload := a.platoonPayloadFor("v4", data)
		list := objectList(payload.values["отдельная_работа"])
		if len(list) != min(len(data.WorkGroups), workGroupModel) {
			t.Fatalf("%s: %d groups in the input of %d", group, len(list), len(data.WorkGroups))
		}
		input := string(canonicalJSON(payload))
		for _, p := range portraits {
			if strings.Contains(input, p.student.Name) {
				t.Errorf("%s: the name %s is in the input", group, p.student.Name)
			}
		}
		if slices.Index(payload.keys, "отдельная_работа") != slices.Index(payload.keys, "каталог_рекомендаций")-1 {
			t.Errorf("%s: keys %v", group, payload.keys)
		}
	}
}

// «До образа далеко» is separate work only for fewer than half of the platoon.
func TestFarIsSeparateWorkForFewerThanHalf(t *testing.T) {
	a := testApp(t)
	portraits := fixedPlatoon(a, a.bank.Version)
	far := func(n int) []studentPortrait {
		list := []studentPortrait{}
		for i, p := range portraits {
			data := *p.data
			data.Defender = slices.Clone(data.Defender)
			situation := situationAchieved
			if i < n {
				situation = situationFar
			}
			for g := range data.Defender {
				data.Defender[g].Situation = situation
				data.Defender[g].Pressure = ""
			}
			data.Methods, data.Identity = nil, nil
			list = append(list, studentPortrait{p.student, &data})
		}
		return list
	}
	if len(portraits) < 3 {
		t.Fatalf("platoon of %d", len(portraits))
	}
	if groups := a.workGroups(far(1)); len(groups) != 1 || !strings.Contains(groups[0].Bases[0], "до него далеко") || len(groups[0].Bases) != len(a.defender.Groups) {
		t.Fatalf("one of three far: %+v", groups)
	}
	if groups := a.workGroups(far(len(portraits))); len(groups) != 0 {
		t.Fatalf("all far: %+v", groups)
	}
}

func TestLooksOfOneKindGoTogether(t *testing.T) {
	looks := addLook(nil, "ситуация по качествам «дисциплина и собранность»")
	looks = addLook(looks, "Самообладание")
	looks = addLook(looks, "ситуация по качествам «товарищество и культура отношений»")
	looks = addLook(looks, "Самообладание")
	if want := []string{"ситуация по качествам «дисциплина и собранность», «товарищество и культура отношений»", "Самообладание"}; !slices.Equal(looks, want) {
		t.Fatalf("looks: %q", looks)
	}
}
