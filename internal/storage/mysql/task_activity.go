package mysql

import (
	"context"
	"encoding/json"
	"time"

	"github.com/example/taskmanager/internal/domain"
	"github.com/example/taskmanager/internal/usecase"
)

func (s *Storage) AddTaskHistory(ctx context.Context, taskID, changedBy int64, changes domain.Changes, at time.Time) error {
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO task_history (task_id, changed_by, changes, created_at) VALUES (?, ?, ?, ?)`,
		taskID, changedBy, changesJSON, at)
	return err
}

func (s *Storage) ListTaskHistory(ctx context.Context, taskID int64, page usecase.Page) ([]domain.HistoryEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, changed_by, changes, created_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY id
		LIMIT ? OFFSET ?`, taskID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []domain.HistoryEntry{}
	for rows.Next() {
		var h domain.HistoryEntry
		var changesJSON []byte
		if err := rows.Scan(&h.ID, &h.TaskID, &h.ChangedBy, &changesJSON, &h.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(changesJSON, &h.Changes); err != nil {
			return nil, err
		}
		entries = append(entries, h)
	}
	return entries, rows.Err()
}

func (s *Storage) AddComment(ctx context.Context, c *domain.Comment) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO task_comments (task_id, user_id, content, created_at) VALUES (?, ?, ?, ?)`,
		c.TaskID, c.UserID, c.Content, c.CreatedAt)
	if err != nil {
		return err
	}
	c.ID, err = res.LastInsertId()
	return err
}

func (s *Storage) ListComments(ctx context.Context, taskID int64, page usecase.Page) ([]domain.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, user_id, content, created_at
		FROM task_comments
		WHERE task_id = ?
		ORDER BY id
		LIMIT ? OFFSET ?`, taskID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []domain.Comment{}
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}
