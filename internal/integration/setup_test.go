package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"

	"github.com/example/taskmanager/internal/domain"
	"github.com/example/taskmanager/internal/storage/mysql"
)

const testDatabase = "taskmanager_test"

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	_ = godotenv.Load("../../.env")
	rootPassword := os.Getenv("MYSQL_ROOT_PASSWORD")
	if rootPassword == "" {
		t.Skip("MYSQL_ROOT_PASSWORD is not set; skipping integration test")
	}
	params := mysql.ConnectionParams{
		Host:     envOr("DB_HOST", "127.0.0.1"),
		Port:     envOr("DB_PORT", "3306"),
		User:     "root",
		Password: rootPassword,
	}
	ctx := context.Background()

	server, err := sql.Open("mysql", params.DSN())
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	for _, query := range []string{
		"DROP DATABASE IF EXISTS " + testDatabase,
		"CREATE DATABASE " + testDatabase,
	} {
		if _, err := server.ExecContext(ctx, query); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}

	params.Database = testDatabase
	db, err := sql.Open("mysql", params.DSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if err := mysql.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	res, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func createUser(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	return mustExec(t, db, `INSERT INTO users (email, password_hash, name) VALUES (?, 'x', ?)`,
		strings.ToLower(name)+"@example.com", name)
}

func createTeam(t *testing.T, db *sql.DB, name string, owner int64) int64 {
	t.Helper()
	id := mustExec(t, db, `INSERT INTO teams (name, created_by) VALUES (?, ?)`, name, owner)
	addMember(t, db, id, owner, domain.RoleOwner)
	return id
}

func addMember(t *testing.T, db *sql.DB, team, user int64, role domain.Role) {
	t.Helper()
	mustExec(t, db, `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`, team, user, role)
}

func createOpenTask(t *testing.T, db *sql.DB, team, creator int64, status domain.TaskStatus) int64 {
	t.Helper()
	return mustExec(t, db, `INSERT INTO tasks (team_id, title, description, status, created_by) VALUES (?, ?, '', ?, ?)`,
		team, fmt.Sprintf("open %s", status), status, creator)
}

type closedTask struct {
	assignee      int64
	closedDaysAgo int
	hoursToClose  int
}

func createClosedTask(t *testing.T, db *sql.DB, team, creator int64, task closedTask) int64 {
	t.Helper()
	return mustExec(t, db, `
		INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at)
		VALUES (?, 'closed', '', 'done', ?, ?,
		        NOW() - INTERVAL ? DAY - INTERVAL ? HOUR,
		        NOW() - INTERVAL ? DAY,
		        NOW() - INTERVAL ? DAY)`,
		team, creator, task.assignee,
		task.closedDaysAgo, task.hoursToClose, task.closedDaysAgo, task.closedDaysAgo)
}

func addComment(t *testing.T, db *sql.DB, task, user int64) {
	t.Helper()
	mustExec(t, db, `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, 'hi')`, task, user)
}

func newStorage(db *sql.DB) *mysql.Storage { return mysql.New(db) }
