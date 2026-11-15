package usecase

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarkPeopleFindsThePeople(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Майор Захаров всегда помогал.", "Майор [человек 1] всегда помогал."},
		{"Спасибо начальнику цикла Петрову и куратору взвода Сидоренко.", "Спасибо начальнику цикла [человек 1] и куратору взвода [человек 2]."},
		{"Сергей Иванович Петров объяснял. Потом Петров сказал, что Петрова не будет.", "[человек 1] объяснял. Потом [человек 1] сказал, что [человек 1] не будет."},
		{"Сергей Иванович помог, а Петров И. И. нет.", "[человек 1] помог, а [человек 2] нет."},
		{"Любовь к Родине и Слава России — главное.", "Любовь к Родине и Слава России — главное."},
		{"Вера Петровна вела занятия.", "[человек 1] вела занятия."},
		{"Мы с Саней и Димоном ходили в наряд.", "Мы с [человек 1] и [человек 2] ходили в наряд."},
		{"В Российской армии служил мой дед.", "В Российской армии служил мой дед."},
		{"Пришёл в ВУЦ после второго курса.", "Пришёл в ВУЦ после второго курса."},
		{"Цой сказал строиться. Старшина Цой был строг.", "[человек 1] сказал строиться. Старшина [человек 1] был строг."},
		{"Это было 12.05.2024 на сборах.", "Это было [скрыто] на сборах."},
		{"Благодарен Захарову, с Захаровым было легко, Захаров — молодец.", "Благодарен [человек 1], с [человек 1] было легко, [человек 1] — молодец."},
		{"Служил в Вооружённых Силах, дежурил по Кремлю.", "Служил в Вооружённых Силах, дежурил по Кремлю."},
		{"Подполковник Игорь Мельников-Сокол поддержал.", "Подполковник [человек 1] поддержал."},
		{"Готов служить Родине и Отечеству.", "Готов служить Родине и Отечеству."},
	} {
		marks := markPeople([]string{c.in})
		if got := marks.text(0); got != c.want {
			t.Errorf("%q:\n got %q\nwant %q", c.in, got, c.want)
		}
	}
}

func TestMarkPeopleNumbersOverAllTexts(t *testing.T) {
	marks := markPeople([]string{"Помог майор Захаров.", "Сергей Иванович Захаров объяснял, Иванов слушал.", "Сергей Иванович снова объяснил."})
	want := []string{"Помог майор [человек 1].", "[человек 1] объяснял, [человек 2] слушал.", "[человек 1] снова объяснил."}
	for n := range want {
		if got := marks.text(n); got != want[n] {
			t.Errorf("text %d: %q, want %q", n, got, want[n])
		}
	}
	if marks.name(1) != "Захаров" || marks.name(2) != "Иванов" {
		t.Errorf("names: %q, %q", marks.name(1), marks.name(2))
	}
}

func TestPeopleMarksRestoreTheWordsOfTheAuthor(t *testing.T) {
	marks := markPeople([]string{"Начальник цикла  Петрову я благодарен: 12.05.2024 он помог.", "Куратор взвода Ковальчук хвалил."})
	output := map[string]any{
		"formation": map[string]any{"text": "Стал увереннее.", "quotes": []any{map[string]any{"question": "1.1", "quote": "[человек 1] я благодарен: [скрыто] он"}}},
		"teachers": []any{
			map[string]any{"position": "начальник цикла", "name": "[человек 1]", "quotes": []any{map[string]any{"question": "1.1", "quote": "Начальник цикла [человек 1] я благодарен"}}},
			map[string]any{"position": "куратор взвода", "name": "[человек 2]", "quotes": []any{map[string]any{"question": "1.2", "quote": "Куратор взвода [человек 2]"}}},
		},
	}
	data, _ := json.Marshal(output)
	var got map[string]any
	if err := json.Unmarshal(marks.restore(data), &got); err != nil {
		t.Fatal(err)
	}
	quote := func(v any) string { return asMap(listOf(asMap(v), "quotes")[0])["quote"].(string) }
	if q := quote(got["formation"]); q != "Петрову я благодарен: 12.05.2024 он" {
		t.Errorf("quote: %q", q)
	}
	teachers := listOf(got, "teachers")
	if name := asMap(teachers[0])["name"]; name != "Петров" {
		t.Errorf("name: %v", name)
	}
	if q := quote(teachers[0]); q != "Начальник цикла  Петрову я благодарен" {
		t.Errorf("quote: %q", q)
	}
	if name := asMap(teachers[1])["name"]; name != "Ковальчук" {
		t.Errorf("name: %v", name)
	}
	if !strings.Contains(marks.names("Помог [человек 2]."), "Ковальчук") {
		t.Error("names: no surname")
	}
}

func TestLabelProblemFindsLabels(t *testing.T) {
	for text, want := range map[string]bool{"Куратору [человек 2] удалось": true, "человек 3 помог": true, "Один человек помог": false, "Преподаватель помог": false} {
		if got := labelProblem("x", text) != ""; got != want {
			t.Errorf("%q: %v", text, got)
		}
	}
}
