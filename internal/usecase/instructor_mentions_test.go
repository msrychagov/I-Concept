package usecase

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

var mentionList = []model.Instructor{
	{ID: "ivanov", Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович", Rank: "полковник"},
	{ID: "petrova", Surname: "Петрова", Name: "Анна", Patronymic: "Сергеевна"},
	{ID: "sidorov", Surname: "Сидоров", Name: "Игорь", Patronymic: "Петрович", Rank: "майор"},
	{ID: "kovalchuk", Surname: "Ковальчук", Name: "Олег", Patronymic: "Ильич", Rank: "подполковник"},
	{ID: "belo", Surname: "Белоцерковский", Name: "Юрий", Patronymic: "Андреевич", Rank: "капитан"},
	{ID: "kuz-senior", Surname: "Кузнецов", Name: "Илья", Patronymic: "Павлович", Rank: "старший лейтенант"},
	{ID: "kuz", Surname: "Кузнецов", Name: "Никита", Patronymic: "Львович", Rank: "лейтенант"},
	{ID: "shevchenko", Surname: "Шевченко", Name: "Павел", Patronymic: "Олегович"},
	{ID: "zima", Surname: "Зима", Name: "Андрей", Patronymic: "Ильич"},
	{ID: "semenov", Surname: "Семёнов", Name: "Пётр", Patronymic: "Алексеевич"},
	{ID: "orlov-colonel", Surname: "Орлов", Name: "Олег", Patronymic: "Олегович", Rank: "полковник запаса"},
	{ID: "orlov", Surname: "Орлов", Name: "Олег", Patronymic: "Олегович"},
	{ID: "chaika", Surname: "Чайковская", Name: "Мария"},
	{ID: "rk", Surname: "Римский-Корсаков", Name: "Юрий"},
}

func maskOne(text string) (string, []string) {
	masked, labels := MaskInstructors(mentionList, []string{text})
	ids := []string{}
	for _, label := range labels {
		ids = append(ids, label.Instructor.ID)
	}
	return masked[0], ids
}

func TestMaskInstructorsFindsMentionsInAllCases(t *testing.T) {
	cases := []struct{ text, want, id string }{
		{"Спасибо полковнику Иванову за поддержку.", "Спасибо [преподаватель 1] за поддержку.", "ivanov"},
		{"Иванов помог", "[преподаватель 1] помог", "ivanov"},
		{"нет Иванова", "нет [преподаватель 1]", "ivanov"},
		{"с Ивановым", "с [преподаватель 1]", "ivanov"},
		{"об Иванове", "об [преподаватель 1]", "ivanov"},
		{"Игоря Петровича Иванова", "[преподаватель 1]", "ivanov"},
		{"полковник Игорь Петрович Иванов сказал", "[преподаватель 1] сказал", "ivanov"},
		{"товарищ полковник Иванов", "[преподаватель 1]", "ivanov"},
		{"генерал-майору Иванову", "[преподаватель 1]", "ivanov"},
		{"Иванов И. П. пришёл", "[преподаватель 1] пришёл", "ivanov"},
		{"И.П. Иванов пришёл", "[преподаватель 1] пришёл", "ivanov"},
		{"майор Сидоров", "[преподаватель 1]", "sidorov"},
		{"с Петровой Анной Сергеевной", "с [преподаватель 1]", "petrova"},
		{"Анну Сергеевну", "[преподаватель 1]", "petrova"},
		{"у Анны Сергеевны", "у [преподаватель 1]", "petrova"},
		{"Петровой", "[преподаватель 1]", "petrova"},
		{"Ковальчуку", "[преподаватель 1]", "kovalchuk"},
		{"Ковальчуком", "[преподаватель 1]", "kovalchuk"},
		{"Олегу Ильичу", "[преподаватель 1]", "kovalchuk"},
		{"капитана Белоцерковского", "[преподаватель 1]", "belo"},
		{"Белоцерковскому", "[преподаватель 1]", "belo"},
		{"Юрию Андреевичу", "[преподаватель 1]", "belo"},
		{"старший лейтенант Кузнецов", "[преподаватель 1]", "kuz-senior"},
		{"старшему лейтенанту Кузнецову", "[преподаватель 1]", "kuz-senior"},
		{"лейтенанту Кузнецову", "[преподаватель 1]", "kuz"},
		{"Ильи Павловича", "[преподаватель 1]", "kuz-senior"},
		{"Ильёй Павловичем", "[преподаватель 1]", "kuz-senior"},
		{"Никите Львовичу", "[преподаватель 1]", "kuz"},
		{"Кузнецовым Никитой", "[преподаватель 1]", "kuz"},
		{"Шевченко", "[преподаватель 1]", "shevchenko"},
		{"Павла Олеговича", "[преподаватель 1]", "shevchenko"},
		{"Семенову Петру Алексеевичу", "[преподаватель 1]", "semenov"},
		{"Семёнов", "[преподаватель 1]", "semenov"},
		{"Зимой Андреем Ильичом", "[преподаватель 1]", "zima"},
		{"полковник Орлов", "[преподаватель 1]", "orlov-colonel"},
		{"майор Игорь Петрович", "[преподаватель 1]", "sidorov"},
		{"Иванову Игорю Петровичу, полковнику", "[преподаватель 1], полковнику", "ivanov"},
		{"Чайковской", "[преподаватель 1]", "chaika"},
		{"Чайковскую", "[преподаватель 1]", "chaika"},
		{"Римскому-Корсакову", "[преподаватель 1]", "rk"},
		// Case does not matter (the user's decision on October 23).
		{"иванову", "[преподаватель 1]", "ivanov"},
		{"ИВАНОВ", "[преподаватель 1]", "ivanov"},
		{"зима была холодной", "[преподаватель 1] была холодной", "zima"},
		{"«Иванов» сказал: (Иванову)!", "«[преподаватель 1]» сказал: ([преподаватель 1])!", "ivanov"},
	}
	for _, c := range cases {
		got, ids := maskOne(c.text)
		if got != c.want || len(ids) != 1 || ids[0] != c.id {
			t.Errorf("%q: got %q %v, want %q [%s]", c.text, got, ids, c.want, c.id)
		}
	}
}

func TestMaskInstructorsLeavesAmbiguousAndPartialMentions(t *testing.T) {
	for _, text := range []string{
		"Игорь Петрович помог",         // Иванов and Сидоров
		"Игоря Петровича",              // the same in another case
		"Кузнецов объяснил",            // two Кузнецовы
		"капитан Кузнецов",             // the rank fits neither of them
		"Орлов",                        // full namesakes
		"Олег Олегович",                // full namesakes
		"Игорь сказал",                 // a name alone
		"полковник сказал",             // a rank alone
		"товарищ полковник, разрешите", // a rank alone
		"Ивановка и петрович",          // neither a form nor a mention
		"Иванов-Петров",                // another surname
		"",
	} {
		if got, ids := maskOne(text); got != text || len(ids) != 0 {
			t.Errorf("%q changed: %q %v", text, got, ids)
		}
	}
}

func TestMaskInstructorsNumbersLabelsAcrossTexts(t *testing.T) {
	masked, labels := MaskInstructors(mentionList, []string{
		"Петрова помогла, а Иванов, Сидоров — нет.",
		"Полковнику Иванову и Анне Сергеевне спасибо.",
		"Кузнецов и полковник Игорь Петрович Иванов.",
	})
	want := []string{
		"[преподаватель 1] помогла, а [преподаватель 2], [преподаватель 3] — нет.",
		"[преподаватель 2] и [преподаватель 1] спасибо.",
		"Кузнецов и [преподаватель 2].",
	}
	for i := range want {
		if masked[i] != want[i] {
			t.Errorf("text %d: %q, want %q", i, masked[i], want[i])
		}
	}
	ids := []string{"petrova", "ivanov", "sidorov"}
	if len(labels) != len(ids) {
		t.Fatalf("labels: %+v", labels)
	}
	for i, id := range ids {
		if labels[i].Number != i+1 || labels[i].Label != instructorLabel(i+1) || labels[i].Instructor.ID != id {
			t.Errorf("label %d: %+v", i, labels[i])
		}
	}
}

func TestMaskInstructorsWithoutList(t *testing.T) {
	masked, labels := MaskInstructors(nil, []string{"Полковник Иванов помог."})
	if masked[0] != "Полковник Иванов помог." || len(labels) != 0 {
		t.Fatalf("%q %+v", masked[0], labels)
	}
}
