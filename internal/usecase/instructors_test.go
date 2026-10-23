package usecase

import (
	"errors"
	"strings"
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func instructorApp(t *testing.T) *Service {
	t.Helper()
	a := testApp(t)
	if err := a.repo.Update(func(db *model.Database) error {
		db.VUCs = []model.VUC{{ID: "v1", Name: "ВУЦ один"}, {ID: "v2", Name: "ВУЦ два"}}
		db.Students = []model.Account{
			{ID: "t1", Role: "teacher", Name: "Первый Преподаватель", VUC: "ВУЦ один"},
			{ID: "t2", Role: "teacher", Name: "Второй Преподаватель", VUC: "ВУЦ один"},
			{ID: "t3", Role: "teacher", Name: "Чужой Преподаватель", VUC: "ВУЦ два"},
			{ID: "t4", Role: "teacher", Name: "Без ВУЦ"},
			{ID: "s1", Role: "student", Name: "Студент", VUC: "ВУЦ один"},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return a
}

func errorKind(err error) ErrorKind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return ""
}

func listed(t *testing.T, a *Service, teacherID string) []InstructorView {
	t.Helper()
	value, err := a.InstructorList(teacherID)
	if err != nil {
		t.Fatal(err)
	}
	return value.(map[string]any)["instructors"].([]InstructorView)
}

func TestInstructorListIsSharedWithinCenter(t *testing.T) {
	a := instructorApp(t)
	saved, err := a.SaveInstructor("t1", "", InstructorInput{Surname: "  Петров ", Name: "Олег", Rank: "майор"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.SaveInstructor("t1", "", InstructorInput{Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович  ", Rank: "полковник"}); err != nil {
		t.Fatal(err)
	}
	// Namesakes are allowed.
	if _, err = a.SaveInstructor("t1", "", InstructorInput{Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович"}); err != nil {
		t.Fatal(err)
	}
	list := listed(t, a, "t2")
	if len(list) != 3 || list[0].Surname != "Иванов" || list[2].Surname != "Петров" || list[2].UpdatedByName != "Первый Преподаватель" {
		t.Fatalf("list of the center: %+v", list)
	}
	// Another teacher of the center edits the entry.
	id := saved.(model.Instructor).ID
	if _, err = a.SaveInstructor("t2", id, InstructorInput{Surname: "Петров", Name: "Олег", Patronymic: "Ильич", Rank: "подполковник"}); err != nil {
		t.Fatal(err)
	}
	list = listed(t, a, "t1")
	if list[2].Patronymic != "Ильич" || list[2].Rank != "подполковник" || list[2].UpdatedByName != "Второй Преподаватель" || list[2].VUC != "ВУЦ один" {
		t.Fatalf("edited entry: %+v", list[2])
	}
	// Another center sees nothing and cannot change or delete the entry.
	if got := listed(t, a, "t3"); len(got) != 0 {
		t.Fatalf("other center sees: %+v", got)
	}
	if _, err = a.SaveInstructor("t3", id, InstructorInput{Surname: "Чужой", Name: "Иван"}); errorKind(err) != NotFound {
		t.Fatalf("other center edited: %v", err)
	}
	if _, err = a.DeleteInstructor("t3", id); errorKind(err) != NotFound {
		t.Fatalf("other center deleted: %v", err)
	}
	if _, err = a.DeleteInstructor("t2", id); err != nil {
		t.Fatal(err)
	}
	if got := listed(t, a, "t1"); len(got) != 2 {
		t.Fatalf("after delete: %+v", got)
	}
	if _, err = a.DeleteInstructor("t2", id); errorKind(err) != NotFound {
		t.Fatalf("deleted twice: %v", err)
	}
}

func TestInstructorListRejectsInvalidInputAndAccounts(t *testing.T) {
	a := instructorApp(t)
	for _, input := range []InstructorInput{{Name: "Игорь"}, {Surname: "Иванов", Name: "  "}, {Surname: "Иванов", Name: "12"},
		{Surname: strings.Repeat("я", 61), Name: "Игорь"}} {
		if _, err := a.SaveInstructor("t1", "", input); errorKind(err) != Invalid {
			t.Fatalf("%+v accepted: %v", input, err)
		}
	}
	if _, err := a.SaveInstructor("t4", "", InstructorInput{Surname: "Иванов", Name: "Игорь"}); errorKind(err) != Forbidden {
		t.Fatalf("teacher without a center: %v", err)
	}
	if _, err := a.InstructorList("s1"); errorKind(err) != Unauthenticated {
		t.Fatalf("student reads the list: %v", err)
	}
	if got := mustSnapshot(t, a.repo).Instructors; len(got) != 0 {
		t.Fatalf("rejected input stored: %+v", got)
	}
}

func TestInstructorLabelsAreStable(t *testing.T) {
	a := instructorApp(t)
	save := func(teacherID string, input InstructorInput) model.Instructor {
		t.Helper()
		value, err := a.SaveInstructor(teacherID, "", input)
		if err != nil {
			t.Fatal(err)
		}
		return value.(model.Instructor)
	}
	ivanov := save("t1", InstructorInput{Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович", Rank: "полковник"})
	petrova := save("t2", InstructorInput{Surname: "Петрова", Name: "Анна"})
	last := save("t1", InstructorInput{Surname: "Сидоров", Name: "Олег"})
	other := save("t3", InstructorInput{Surname: "Чужой", Name: "Иван"})
	if ivanov.Number != 1 || petrova.Number != 2 || last.Number != 3 || other.Number != 1 {
		t.Fatalf("numbers: %d %d %d, other center %d", ivanov.Number, petrova.Number, last.Number, other.Number)
	}
	// An edit keeps the label; a deleted number is not given again.
	if _, err := a.SaveInstructor("t2", ivanov.ID, InstructorInput{Surname: "Иванов", Name: "Игорь", Patronymic: "Петрович", Rank: "генерал-майор"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.DeleteInstructor("t1", last.ID); err != nil {
		t.Fatal(err)
	}
	if next := save("t1", InstructorInput{Surname: "Новиков", Name: "Павел"}); next.Number != 4 {
		t.Fatalf("deleted number reused: %d", next.Number)
	}
	db := mustSnapshot(t, a.repo)
	want := []InstructorLabel{
		{"[преподаватель 1]", "Иванов Игорь Петрович, генерал-майор"},
		{"[преподаватель 2]", "Петрова Анна"},
		{"[преподаватель 4]", "Новиков Павел"},
	}
	got := instructorLabels(db, "ВУЦ один")
	if len(got) != len(want) {
		t.Fatalf("labels: %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("label %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	if found, ok := instructorByLabel(db, "ВУЦ один", "[преподаватель 2]"); !ok || found.ID != petrova.ID {
		t.Fatalf("label 2: %+v %v", found, ok)
	}
	for _, label := range []string{"[преподаватель 3]", "[преподаватель 9]", "преподаватель 1", "Иванов"} {
		if _, ok := instructorByLabel(db, "ВУЦ один", label); ok {
			t.Errorf("%q resolved", label)
		}
	}
	// A label of another center is not trusted.
	if found, ok := instructorByLabel(db, "ВУЦ два", "[преподаватель 2]"); ok {
		t.Fatalf("other center resolved: %+v", found)
	}
}

func TestDemoHasInstructors(t *testing.T) {
	a := testApp(t)
	if _, err := a.ResetDemo(); err != nil {
		t.Fatal(err)
	}
	db := mustSnapshot(t, a.repo)
	labels := instructorLabels(db, "Демонстрационный ВУЦ")
	if len(labels) != 7 || labels[0] != (InstructorLabel{"[преподаватель 1]", "Иванов Игорь Петрович, полковник"}) ||
		labels[4] != (InstructorLabel{"[преподаватель 5]", "Петрова Анна Сергеевна"}) || labels[6].Label != "[преподаватель 7]" {
		t.Fatalf("demo labels: %+v", labels)
	}
	// The next entry continues the numbering.
	var teacherID string
	for _, account := range db.Students {
		if account.Role == "teacher" {
			teacherID = account.ID
		}
	}
	value, err := a.SaveInstructor(teacherID, "", InstructorInput{Surname: "Новиков", Name: "Павел"})
	if err != nil {
		t.Fatal(err)
	}
	if number := value.(model.Instructor).Number; number != 8 {
		t.Fatalf("next number after demo: %d", number)
	}
}
