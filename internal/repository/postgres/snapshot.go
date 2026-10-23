package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"vucdiagnostic/internal/domain/model"
)

func readSnapshot(ctx context.Context, tx pgx.Tx) (db model.Database, err error) {
	if db.VUCs, err = centers.list(ctx, tx); err != nil {
		return
	}
	if db.Platoons, err = platoons.list(ctx, tx); err != nil {
		return
	}
	if db.Students, err = accounts.list(ctx, tx); err != nil {
		return
	}
	if db.Profiles, err = profiles.list(ctx, tx); err != nil {
		return
	}
	if db.Assignments, err = assignments.list(ctx, tx); err != nil {
		return
	}
	if db.Logins, err = logins.list(ctx, tx); err != nil {
		return
	}
	if db.Invitations, err = invitations.list(ctx, tx); err != nil {
		return
	}
	if db.Sessions, err = sessions.list(ctx, tx); err != nil {
		return
	}
	if db.Submissions, err = submissions.list(ctx, tx); err != nil {
		return
	}
	if db.Portraits, err = portraits.list(ctx, tx); err != nil {
		return
	}
	if db.Instructors, err = instructors.list(ctx, tx); err != nil {
		return
	}
	db.Cohorts, err = cohorts.list(ctx, tx)
	return
}

func saveSnapshot(ctx context.Context, tx pgx.Tx, before, after model.Database) error {
	operations := []func() error{
		func() error { return centers.sync(ctx, tx, before.VUCs, after.VUCs) },
		func() error { return platoons.sync(ctx, tx, before.Platoons, after.Platoons) },
		func() error { return accounts.sync(ctx, tx, before.Students, after.Students) },
		func() error { return profiles.sync(ctx, tx, before.Profiles, after.Profiles) },
		func() error { return assignments.sync(ctx, tx, before.Assignments, after.Assignments) },
		func() error { return logins.sync(ctx, tx, before.Logins, after.Logins) },
		func() error { return invitations.sync(ctx, tx, before.Invitations, after.Invitations) },
		func() error { return sessions.sync(ctx, tx, before.Sessions, after.Sessions) },
		func() error { return submissions.sync(ctx, tx, before.Submissions, after.Submissions) },
		func() error { return portraits.sync(ctx, tx, before.Portraits, after.Portraits) },
		func() error { return instructors.sync(ctx, tx, before.Instructors, after.Instructors) },
		func() error { return cohorts.sync(ctx, tx, before.Cohorts, after.Cohorts) },
	}
	for _, operation := range operations {
		if err := operation(); err != nil {
			return err
		}
	}
	return nil
}
