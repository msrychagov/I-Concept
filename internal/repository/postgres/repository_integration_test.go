package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vucdiagnostic/internal/app"
	"vucdiagnostic/internal/domain/model"
	"vucdiagnostic/internal/migrations"
	"vucdiagnostic/internal/repository/postgres"
)

// Each run uses a private schema, never truncating existing application tables.
func TestPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration checks")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("integration_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := postgres.New(ctx, pool); err == nil {
		t.Fatal("unmigrated schema accepted")
	}
	for range 2 {
		if err := migrations.Up(ctx, pool); err != nil {
			t.Fatal(err)
		}
	}
	r, err := postgres.New(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	a, err := app.New(app.Config{DatabaseURL: u.String()})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	if err := a.SeedDemo(); err == nil {
		t.Fatal("seed overwrote data")
	}
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/teacher/login", strings.NewReader(`{"email":"teacher@example.test","password":"TeacherDemo2026!"}`))
	login.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	a.ServeHTTP(response, login)
	if response.Code != http.StatusOK {
		t.Fatalf("login: %d %s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login cookie missing")
	}
	a.Close()
	restarted, err := app.New(app.Config{DatabaseURL: u.String()})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/teachers/me", "/api/v1/dashboard", "/api/v1/teachers/me/platoons", "/api/v1/forms/autobiography/overview"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		restarted.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("after restart %s: %d %s", path, response.Code, response.Body.String())
		}
	}
	restarted.Close()
	db, err := r.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Students) != 10 || len(db.Platoons) != 3 || len(db.Assignments) != 4 || len(db.Submissions) != 2 || len(db.Sessions) == 0 {
		t.Fatal("demo entities missing")
	}
	// Cover authentication/invitation persistence alongside all demo entities.
	if err := r.Update(func(d *model.Database) error {
		d.Logins = append(d.Logins, model.Login{Hash: "login-hash", StudentID: d.Students[0].ID, Expires: time.Now().UTC().Add(time.Hour)})
		d.Invitations = append(d.Invitations, model.Invitation{Hash: "invite-hash", Role: "student", IssuerID: d.Students[0].ID, PlatoonID: d.Platoons[0].ID})
		d.Sessions = append(d.Sessions, model.SessionRecord{Session: model.Session{ID: "external-archive", StudentID: "external-student", AssignmentID: "external-assignment", TeacherID: d.Students[0].ID, Revoked: true}})
		d.Portraits = append(d.Portraits, model.Portrait{ID: "portrait-key", Task: "autobiography", TaskVersion: "v1", Fingerprint: "print", Model: "test-model", Created: "2026-10-19T10:00:00Z", Input: `{"б":1,"а":2}`, Output: []byte(`{"summary":"текст"}`), Usage: model.PortraitUsage{In: 10, Out: 5}, Seconds: 1.5})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	db, err = r.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Portraits) != 1 || db.Portraits[0].Input != `{"б":1,"а":2}` || db.Portraits[0].Usage.Out != 5 {
		t.Fatalf("portrait not stored as written: %+v", db.Portraits)
	}
	otherPool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer otherPool.Close()
	other, err := postgres.New(ctx, otherPool)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := other.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(db, roundTrip) {
		t.Fatal("cross-connection round trip changed data")
	}
	sentinel := errors.New("abort")
	if err := r.Update(func(d *model.Database) error { d.Students = nil; return sentinel }); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if err := r.Update(func(d *model.Database) error { d.Logins[0].StudentID = "missing-account"; return nil }); err == nil {
		t.Fatal("foreign key not enforced")
	}
	unchanged, err := r.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(db, unchanged) {
		t.Fatal("rollback changed data")
	}
	var wg sync.WaitGroup
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			writer := r
			if i%2 == 0 {
				writer = other
			}
			failures <- writer.Update(func(d *model.Database) error { d.Invitations[0].UseCount++; return nil })
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	updated, err := r.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if updated.Invitations[0].UseCount != 12 {
		t.Fatal("concurrent updates lost")
	}
	backup, err := r.ReplaceWithBackup(model.Database{})
	if err != nil || backup == "" {
		t.Fatalf("backup: %s %v", backup, err)
	}
	var saved model.Database
	if err := pool.QueryRow(ctx, "SELECT payload FROM repository_backups ORDER BY id DESC LIMIT 1").Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, updated) {
		t.Fatal("backup incomplete")
	}
	a, err = app.New(app.Config{DatabaseURL: u.String()})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.ImportData(saved); err != nil {
		t.Fatal(err)
	}
	if err := a.ImportData(saved); err == nil {
		t.Fatal("import overwrote nonempty database")
	}
	restored, err := r.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, restored) {
		t.Fatal("import changed data")
	}
	otherPool.Close()
	if _, err := other.ReadSnapshot(); err == nil {
		t.Fatal("connection error hidden")
	}
}
