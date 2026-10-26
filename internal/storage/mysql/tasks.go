package mysql

import (
	"context"
	"strings"

	"github.com/example/taskmanager/internal/domain"
	"github.com/example/taskmanager/internal/usecase"
)

const taskColumns = `id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version`

func (s *Storage) CreateTask(ctx context.Context, t *domain.Task) error {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.TeamID, t.Title, t.Description, t.Status, t.CreatedBy, t.AssigneeID, t.CreatedAt, t.UpdatedAt, t.ClosedAt, t.Version)
	if err != nil {
		return err
	}
	t.ID, err = res.LastInsertId()
	return err
}

func (s *Storage) GetTask(ctx context.Context, id int64) (*domain.Task, error) {
	return s.getTask(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
}

func (s *Storage) GetTaskForUpdate(ctx context.Context, id int64) (*domain.Task, error) {
	return s.getTask(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ? FOR UPDATE`, id)
}

func (s *Storage) getTask(ctx context.Context, query string, id int64) (*domain.Task, error) {
	var t domain.Task
	err := s.db.QueryRowContext(ctx, query, id).Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Status,
		&t.CreatedBy, &t.AssigneeID, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt, &t.Version)
	if err != nil {
		return nil, translateError(err)
	}
	return &t, nil
}

func (s *Storage) UpdateTask(ctx context.Context, t *domain.Task) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, assignee_id = ?, closed_at = ?, updated_at = ?, version = ?
		WHERE id = ?`,
		t.Title, t.Description, t.Status, t.AssigneeID, t.ClosedAt, t.UpdatedAt, t.Version, t.ID)
	return err
}

func (s *Storage) ListTasks(ctx context.Context, f usecase.TaskFilter) ([]domain.Task, bool, error) {
	where, args := taskWhere(f)
	query := `SELECT ` + taskColumns + ` FROM tasks` + where + ` ORDER BY id DESC LIMIT ?`
	args = append(args, f.Limit+1)
	if f.BeforeID == 0 && f.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, f.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	tasks := []domain.Task{}
	for rows.Next() {
		var t domain.Task
		err := rows.Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Status,
			&t.CreatedBy, &t.AssigneeID, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt, &t.Version)
		if err != nil {
			return nil, false, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasNextPage := len(tasks) > f.Limit
	if hasNextPage {
		tasks = tasks[:f.Limit]
	}
	return tasks, hasNextPage, nil
}

func (s *Storage) CountTasks(ctx context.Context, f usecase.TaskFilter) (int64, error) {
	f.BeforeID = 0
	where, args := taskWhere(f)
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`+where, args...).Scan(&count)
	return count, err
}

func taskWhere(f usecase.TaskFilter) (string, []any) {
	conditions := []string{"team_id = ?"}
	args := []any{f.TeamID}
	if f.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *f.Status)
	}
	if f.AssigneeID != nil {
		conditions = append(conditions, "assignee_id = ?")
		args = append(args, *f.AssigneeID)
	}
	if f.BeforeID > 0 {
		conditions = append(conditions, "id < ?")
		args = append(args, f.BeforeID)
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}
