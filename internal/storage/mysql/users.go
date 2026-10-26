package mysql

import (
	"context"

	"github.com/example/taskmanager/internal/domain"
)

func (s *Storage) CreateUser(ctx context.Context, u *domain.User) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, name, created_at) VALUES (?, ?, ?, ?)`,
		u.Email, u.PasswordHash, u.Name, u.CreatedAt)
	if err != nil {
		return translateError(err)
	}
	u.ID, err = res.LastInsertId()
	return err
}

func (s *Storage) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	return s.getUser(ctx, `SELECT id, email, password_hash, name, created_at FROM users WHERE id = ?`, id)
}

func (s *Storage) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.getUser(ctx, `SELECT id, email, password_hash, name, created_at FROM users WHERE email = ?`, email)
}

func (s *Storage) getUser(ctx context.Context, query string, arg any) (*domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, query, arg).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if err != nil {
		return nil, translateError(err)
	}
	return &u, nil
}
