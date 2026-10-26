package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/example/taskmanager/internal/domain"
)

func (s *Storage) CreateTeam(ctx context.Context, t *domain.Team) error {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO teams (name, created_by, created_at) VALUES (?, ?, ?)`,
		t.Name, t.CreatedBy, t.CreatedAt)
	if err != nil {
		return err
	}
	t.ID, err = res.LastInsertId()
	return err
}

func (s *Storage) ListTeamsForUser(ctx context.Context, userID int64) ([]domain.Team, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.created_by, t.created_at, m.role
		FROM teams t
		JOIN team_members m ON m.team_id = t.id
		WHERE m.user_id = ?
		ORDER BY t.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := []domain.Team{}
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedBy, &t.CreatedAt, &t.CurrentUserRole); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (s *Storage) AddTeamMember(ctx context.Context, teamID, userID int64, role domain.Role, joinedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role, created_at) VALUES (?, ?, ?, ?)`,
		teamID, userID, role, joinedAt)
	return translateError(err)
}

func (s *Storage) GetMembership(ctx context.Context, teamID, userID int64) (domain.Membership, error) {
	membership := domain.Membership{UserID: userID}
	err := s.db.QueryRowContext(ctx,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID).Scan(&membership.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return membership, nil
	}
	if err != nil {
		return membership, err
	}
	membership.IsMember = true
	return membership, nil
}
