package usecase

import (
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"vucdiagnostic/internal/domain/model"
)

// The instructor list is one per center: any teacher of the center reads and
// edits it. Entries with the same full name are allowed (namesakes exist).

type InstructorInput struct {
	Surname    string `json:"surname"`
	Name       string `json:"name"`
	Patronymic string `json:"patronymic"`
	Rank       string `json:"rank"`
}

// InstructorView adds the name of the teacher who saved the entry last.
type InstructorView struct {
	model.Instructor
	UpdatedByName string `json:"updatedByName,omitempty"`
}

func (a *Service) InstructorList(teacherID string) (any, error) {
	db, err := a.repo.ReadSnapshot()
	if err != nil {
		return nil, err
	}
	t, err := managementTeacher(db, teacherID)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, account := range db.Students {
		if account.Role == "teacher" {
			names[account.ID] = account.Name
		}
	}
	list := []InstructorView{}
	for _, instructor := range centerInstructors(db, t.VUC) {
		list = append(list, InstructorView{instructor, names[instructor.UpdatedBy]})
	}
	return map[string]any{"vuc": t.VUC, "instructors": list}, nil
}

// centerInstructors returns the list of a center in alphabetical order.
func centerInstructors(db model.Database, vuc string) []model.Instructor {
	list := []model.Instructor{}
	for _, instructor := range db.Instructors {
		if sameVUC(instructor.VUC, vuc) {
			list = append(list, instructor)
		}
	}
	key := func(i model.Instructor) string {
		return normalWord(i.Surname + " " + i.Name + " " + i.Patronymic + " " + i.Rank)
	}
	sort.SliceStable(list, func(i, j int) bool { return key(list[i]) < key(list[j]) })
	return list
}

func cleanInstructor(input InstructorInput) (InstructorInput, error) {
	clean := func(value string) string { return strings.Join(strings.Fields(value), " ") }
	input = InstructorInput{clean(input.Surname), clean(input.Name), clean(input.Patronymic), clean(input.Rank)}
	hasLetter := func(value string) bool { return strings.IndexFunc(value, unicode.IsLetter) >= 0 }
	if !hasLetter(input.Surname) || !hasLetter(input.Name) {
		return input, problem(Invalid, "Укажите фамилию и имя преподавателя.")
	}
	for _, field := range []string{input.Surname, input.Name, input.Patronymic} {
		if utf8.RuneCountInString(field) > 60 {
			return input, problem(Invalid, "Фамилия, имя и отчество — до 60 символов каждое.")
		}
	}
	if utf8.RuneCountInString(input.Rank) > 80 {
		return input, problem(Invalid, "Звание — до 80 символов.")
	}
	return input, nil
}

// SaveInstructor adds an entry when id is empty and edits it otherwise.
func (a *Service) SaveInstructor(teacherID, id string, input InstructorInput) (any, error) {
	input, err := cleanInstructor(input)
	if err != nil {
		return nil, err
	}
	return a.transaction(func(db *model.Database) (any, error) {
		t, err := managementTeacher(*db, teacherID)
		if err != nil {
			return nil, err
		}
		entry := model.Instructor{Surname: input.Surname, Name: input.Name, Patronymic: input.Patronymic, Rank: input.Rank,
			UpdatedBy: t.ID, Updated: time.Now().UTC().Format(time.RFC3339)}
		if id == "" {
			entry.ID, entry.VUC = uuid(), t.VUC
			db.Instructors = append(db.Instructors, entry)
			return entry, nil
		}
		for i, instructor := range db.Instructors {
			if instructor.ID == id && sameVUC(instructor.VUC, t.VUC) {
				entry.ID, entry.VUC = instructor.ID, instructor.VUC
				db.Instructors[i] = entry
				return entry, nil
			}
		}
		return nil, problem(NotFound, "Преподаватель не найден в списке вашего ВУЦ.")
	})
}

func (a *Service) DeleteInstructor(teacherID, id string) (any, error) {
	return a.transaction(func(db *model.Database) (any, error) {
		t, err := managementTeacher(*db, teacherID)
		if err != nil {
			return nil, err
		}
		for i, instructor := range db.Instructors {
			if instructor.ID == id && sameVUC(instructor.VUC, t.VUC) {
				db.Instructors = append(db.Instructors[:i], db.Instructors[i+1:]...)
				return map[string]any{"ok": true}, nil
			}
		}
		return nil, problem(NotFound, "Преподаватель не найден в списке вашего ВУЦ.")
	})
}

// normalWord lowercases and treats «ё» as «е» for sorting.
func normalWord(value string) string { return strings.ReplaceAll(strings.ToLower(value), "ё", "е") }
