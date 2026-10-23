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
