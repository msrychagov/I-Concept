package usecase

import (
	"testing"
	"vucdiagnostic/internal/domain/model"
)

func TestNormalizeOwnershipAdoptsOnlyUnambiguousLegacyData(t *testing.T) {
	one := testApp(t)
	if err := one.repo.Update(func(db *model.Database) error {
		db.Students = []model.Account{{ID: "teacher-1", Role: "teacher"}, {ID: "student-1", Role: "student"}}
		db.Assignments = []model.Assignment{{ID: "assignment-1"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := one.NormalizeOwnership(); err != nil {
		t.Fatal(err)
	}
	db := one.repo.Snapshot()
	if db.Students[1].OwnerID != "teacher-1" || db.Assignments[0].OwnerID != "teacher-1" {
		t.Fatal("single-teacher legacy data was not adopted")
	}

	many := testApp(t)
	if err := many.repo.Update(func(db *model.Database) error {
		db.Students = []model.Account{{ID: "teacher-1", Role: "teacher"}, {ID: "teacher-2", Role: "teacher"}, {ID: "student-1", Role: "student"}}
		db.Assignments = []model.Assignment{{ID: "assignment-1", Active: true}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := many.NormalizeOwnership(); err != nil {
		t.Fatal(err)
	}
	db = many.repo.Snapshot()
	if assignmentApplies(db.Assignments[0], db.Students[2]) || inTeacherScope(db.Assignments[0].OwnerID, "teacher-1") || inTeacherScope(db.Assignments[0].OwnerID, "teacher-2") {
		t.Fatal("ambiguous legacy data was exposed to an arbitrary account")
	}
}
