package domain

import "time"

type TeamStats struct {
	TeamID           int64                `json:"team_id"`
	TasksByStatus    map[TaskStatus]int64 `json:"tasks_by_status"`
	TotalTasks       int64                `json:"total_tasks"`
	TopAssignees30d  []AssigneeStat       `json:"top_assignees_30d"`
	ClosedTasksTotal int64                `json:"closed_tasks_total"`
	AvgCloseTimeSec  *float64             `json:"avg_close_time_seconds"`
	Comments         CommentStats         `json:"comments"`
	GeneratedAt      time.Time            `json:"generated_at"`
}

type AssigneeStat struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	ClosedTasks int64  `json:"closed_tasks"`
}

type CommentStats struct {
	Total             int64   `json:"total"`
	TasksWithComments int64   `json:"tasks_with_comments"`
	AvgPerTask        float64 `json:"avg_per_task"`
}
