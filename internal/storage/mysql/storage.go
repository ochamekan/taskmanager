package mysql

import (
	"context"
	"database/sql"
	"errors"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/example/taskmanager/internal/usecase"
)

const errDuplicateEntry = 1062

type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Storage struct {
	db   dbtx
	pool *sql.DB
}

func New(pool *sql.DB) *Storage {
	return &Storage{db: pool, pool: pool}
}

func (s *Storage) InTx(ctx context.Context, fn func(tx usecase.Storage) error) error {
	tx, err := s.pool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(&Storage{db: tx, pool: s.pool}); err != nil {
		return err
	}
	return tx.Commit()
}

func translateError(err error) error {
	var mysqlErr *mysqldriver.MySQLError
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return usecase.ErrRecordNotFound
	case errors.As(err, &mysqlErr) && mysqlErr.Number == errDuplicateEntry:
		return usecase.ErrDuplicateRecord
	default:
		return err
	}
}
