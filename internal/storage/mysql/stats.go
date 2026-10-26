package mysql

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/example/taskmanager/internal/domain"
)

const teamStatsQuery = `
WITH team_tasks AS (
    SELECT id, status, assignee_id, created_at, closed_at
    FROM tasks
    WHERE team_id = ?
),
task_counts AS (
    SELECT
        COALESCE(SUM(status = 'todo'), 0)                               AS todo_tasks,
        COALESCE(SUM(status = 'in_progress'), 0)                        AS in_progress_tasks,
        COALESCE(SUM(status = 'done'), 0)                               AS done_tasks,
        COALESCE(SUM(status = 'done' AND closed_at IS NOT NULL), 0)     AS closed_tasks,
        AVG(CASE WHEN status = 'done'
                 THEN TIMESTAMPDIFF(SECOND, created_at, closed_at) END) AS avg_close_seconds
    FROM team_tasks
),
top_assignees_30d AS (
    SELECT u.id AS user_id, u.name, u.email, COUNT(*) AS closed_tasks
    FROM team_tasks t
    JOIN users u ON u.id = t.assignee_id
    WHERE t.status = 'done'
      AND t.closed_at >= NOW() - INTERVAL 30 DAY
    GROUP BY u.id, u.name, u.email
    ORDER BY closed_tasks DESC, u.id
    LIMIT 3
),
comment_counts AS (
    SELECT COUNT(c.id) AS comments, COUNT(DISTINCT c.task_id) AS tasks_with_comments
    FROM team_tasks t
    JOIN task_comments c ON c.task_id = t.id
)
SELECT
    task_counts.todo_tasks,
    task_counts.in_progress_tasks,
    task_counts.done_tasks,
    task_counts.closed_tasks,
    task_counts.avg_close_seconds,
    comment_counts.comments,
    comment_counts.tasks_with_comments,
    (SELECT JSON_ARRAYAGG(JSON_OBJECT(
                'user_id', user_id, 'name', name, 'email', email, 'closed_tasks', closed_tasks))
     FROM top_assignees_30d) AS top_assignees_30d
FROM task_counts
CROSS JOIN comment_counts`

func (s *Storage) TeamStats(ctx context.Context, teamID int64) (*domain.TeamStats, error) {
	stats := &domain.TeamStats{TeamID: teamID, TopAssignees30d: []domain.AssigneeStat{}}
	var todo, inProgress, done int64
	var topAssigneesJSON []byte
	err := s.db.QueryRowContext(ctx, teamStatsQuery, teamID).Scan(
		&todo, &inProgress, &done,
		&stats.ClosedTasksTotal,
		&stats.AvgCloseTimeSec,
		&stats.Comments.Total,
		&stats.Comments.TasksWithComments,
		&topAssigneesJSON,
	)
	if err != nil {
		return nil, err
	}

	stats.TasksByStatus = map[domain.TaskStatus]int64{
		domain.StatusTodo:       todo,
		domain.StatusInProgress: inProgress,
		domain.StatusDone:       done,
	}
	stats.TotalTasks = todo + inProgress + done
	if stats.TotalTasks > 0 {
		stats.Comments.AvgPerTask = float64(stats.Comments.Total) / float64(stats.TotalTasks)
	}

	if topAssigneesJSON != nil {
		if err := json.Unmarshal(topAssigneesJSON, &stats.TopAssignees30d); err != nil {
			return nil, err
		}
		sort.Slice(stats.TopAssignees30d, func(i, j int) bool {
			a, b := stats.TopAssignees30d[i], stats.TopAssignees30d[j]
			if a.ClosedTasks != b.ClosedTasks {
				return a.ClosedTasks > b.ClosedTasks
			}
			return a.UserID < b.UserID
		})
	}
	return stats, nil
}
