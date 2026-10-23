package usecase

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"
	"vucdiagnostic/internal/domain/model"
)

// A cohort is visible to every teacher of the center (the rule «a platoon is in
// one cohort only» needs it), but only its creator changes it, deletes it and
// assigns an essay to it.

type CohortInput struct {
	Name       string   `json:"name"`
	Year       int      `json:"year"`
	PlatoonIDs []string `json:"platoonIds"`
}

// CohortView adds who created the cohort, whether the caller did, and the
// number of essays handed to it.
type CohortView struct {
	model.Cohort
	CreatedByName string `json:"createdByName,omitempty"`
	Mine          bool   `json:"mine"`
	Assignments   int    `json:"assignments"`
	Students      int    `json:"students"`
}

func cohortIn(db model.Database, id, vuc string) (model.Cohort, bool) {
	for _, cohort := range db.Cohorts {
		if cohort.ID == id && sameVUC(cohort.VUC, vuc) {
			return cohort, true
		}
	}
	return model.Cohort{}, false
}

// cohortStudent says whether the account is a student of a platoon of the cohort.
func cohortStudent(cohort model.Cohort, account model.Account) bool {
	return account.Role == "student" && sameVUC(account.VUC, cohort.VUC) && account.PlatoonID != "" && slices.Contains(cohort.PlatoonIDs, account.PlatoonID)
}

func (a *Service) CohortList(teacherID string) (any, error) {
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
	platoons := []model.Platoon{}
	for _, p := range db.Platoons {
		if sameVUC(p.VUC, t.VUC) {
			platoons = append(platoons, p)
		}
	}
	list := []CohortView{}
	for _, cohort := range db.Cohorts {
		if !sameVUC(cohort.VUC, t.VUC) {
			continue
		}
		view := CohortView{Cohort: cohort, CreatedByName: names[cohort.CreatedBy], Mine: cohort.CreatedBy == t.ID}
		if view.PlatoonIDs == nil {
			view.PlatoonIDs = []string{}
		}
		for _, as := range db.Assignments {
			if as.CohortID == cohort.ID {
				view.Assignments++
			}
		}
		for _, account := range db.Students {
			if cohortStudent(cohort, account) {
				view.Students++
			}
		}
		list = append(list, view)
	}
	slices.SortStableFunc(list, func(x, y CohortView) int {
		if x.Year != y.Year {
			return y.Year - x.Year
		}
		return strings.Compare(normalWord(x.Name), normalWord(y.Name))
	})
	return map[string]any{"vuc": t.VUC, "platoons": platoons, "cohorts": list}, nil
}

func cleanCohort(input CohortInput) (CohortInput, error) {
	input.Name = strings.Join(strings.Fields(input.Name), " ")
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 80 {
		return input, problem(Invalid, "Укажите название выпуска (до 80 символов).")
	}
	if input.Year < 2000 || input.Year > 2100 {
		return input, problem(Invalid, "Укажите год выпуска числом, например 2027.")
	}
	seen := map[string]bool{}
	ids := []string{}
	for _, id := range input.PlatoonIDs {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	input.PlatoonIDs = ids
	return input, nil
}

// SaveCohort adds a cohort when id is empty and edits it otherwise.
func (a *Service) SaveCohort(teacherID, id string, input CohortInput) (any, error) {
	input, err := cleanCohort(input)
	if err != nil {
		return nil, err
	}
	return a.transaction(func(db *model.Database) (any, error) {
		t, err := managementTeacher(*db, teacherID)
		if err != nil {
			return nil, err
		}
		for _, platoonID := range input.PlatoonIDs {
			if _, ok := platoonIn(*db, platoonID, t.VUC); !ok {
				return nil, problem(NotFound, "Взвод не найден в вашем ВУЦ.")
			}
		}
		for _, other := range db.Cohorts {
			if !sameVUC(other.VUC, t.VUC) || other.ID == id {
				continue
			}
			if other.Year == input.Year && strings.EqualFold(other.Name, input.Name) {
				return nil, problem(Conflict, "Выпуск с таким названием и годом уже есть.")
			}
			for _, platoonID := range input.PlatoonIDs {
				if slices.Contains(other.PlatoonIDs, platoonID) {
					return nil, problem(Conflict, "Взвод уже входит в выпуск «"+other.Name+"». Взвод может быть только в одном выпуске.")
				}
			}
		}
		if id == "" {
			cohort := model.Cohort{ID: uuid(), VUC: t.VUC, Name: input.Name, Year: input.Year, PlatoonIDs: input.PlatoonIDs, CreatedBy: t.ID, Created: time.Now().UTC().Format(time.RFC3339)}
			db.Cohorts = append(db.Cohorts, cohort)
			return cohort, nil
		}
		for i, cohort := range db.Cohorts {
			if cohort.ID != id || !sameVUC(cohort.VUC, t.VUC) {
				continue
			}
			if cohort.CreatedBy != t.ID {
				return nil, problem(Forbidden, "Выпуск изменяет только тот, кто его создал.")
			}
			cohort.Name, cohort.Year, cohort.PlatoonIDs = input.Name, input.Year, input.PlatoonIDs
			db.Cohorts[i] = cohort
			return cohort, nil
		}
		return nil, problem(NotFound, "Выпуск не найден в вашем ВУЦ.")
	})
}

func (a *Service) DeleteCohort(teacherID, id string) (any, error) {
	return a.transaction(func(db *model.Database) (any, error) {
		t, err := managementTeacher(*db, teacherID)
		if err != nil {
			return nil, err
		}
		for i, cohort := range db.Cohorts {
			if cohort.ID != id || !sameVUC(cohort.VUC, t.VUC) {
				continue
			}
			if cohort.CreatedBy != t.ID {
				return nil, problem(Forbidden, "Выпуск удаляет только тот, кто его создал.")
			}
			for _, as := range db.Assignments {
				if as.CohortID == id {
					return nil, problem(Conflict, "Выпуску выдавали эссе, удалить его нельзя.")
				}
			}
			db.Cohorts = append(db.Cohorts[:i], db.Cohorts[i+1:]...)
			return map[string]any{"ok": true}, nil
		}
		return nil, problem(NotFound, "Выпуск не найден в вашем ВУЦ.")
	})
}
