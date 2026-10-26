package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/example/taskmanager/internal/domain"
)

var (
	ErrRecordNotFound  = errors.New("record not found")
	ErrDuplicateRecord = errors.New("record already exists")
)

type Storage interface {
	InTx(ctx context.Context, fn func(tx Storage) error) error

	CreateUser(ctx context.Context, u *domain.User) error
	GetUserByID(ctx context.Context, id int64) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)

	CreateTeam(ctx context.Context, t *domain.Team) error
	ListTeamsForUser(ctx context.Context, userID int64) ([]domain.Team, error)
	AddTeamMember(ctx context.Context, teamID, userID int64, role domain.Role, joinedAt time.Time) error
	GetMembership(ctx context.Context, teamID, userID int64) (domain.Membership, error)

	CreateTask(ctx context.Context, t *domain.Task) error
	GetTask(ctx context.Context, id int64) (*domain.Task, error)
	GetTaskForUpdate(ctx context.Context, id int64) (*domain.Task, error)
	UpdateTask(ctx context.Context, t *domain.Task) error
	ListTasks(ctx context.Context, filter TaskFilter) (tasks []domain.Task, hasNextPage bool, err error)
	CountTasks(ctx context.Context, filter TaskFilter) (int64, error)

	AddTaskHistory(ctx context.Context, taskID, changedBy int64, changes domain.Changes, at time.Time) error
	ListTaskHistory(ctx context.Context, taskID int64, page Page) ([]domain.HistoryEntry, error)
	AddComment(ctx context.Context, c *domain.Comment) error
	ListComments(ctx context.Context, taskID int64, page Page) ([]domain.Comment, error)

	TeamStats(ctx context.Context, teamID int64) (*domain.TeamStats, error)
}

type TaskListCache interface {
	TeamVersion(ctx context.Context, teamID int64) (version string, available bool)
	Get(ctx context.Context, version string, filter TaskFilter) (page *TaskPage, found bool)
	Set(ctx context.Context, version string, filter TaskFilter, page *TaskPage)
	InvalidateTeam(ctx context.Context, teamID int64)
}

func nowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
