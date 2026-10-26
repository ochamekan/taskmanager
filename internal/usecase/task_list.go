package usecase

import (
	"context"
	"net/http"

	"github.com/example/taskmanager/internal/domain"
)

const (
	defaultTaskPageSize = 20
	maxTaskPageSize     = 100
)

type TaskFilter struct {
	TeamID     int64
	Status     *domain.TaskStatus
	AssigneeID *int64
	Limit      int
	Offset     int
	BeforeID   int64
}

type TaskPage struct {
	Items      []domain.Task `json:"items"`
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
	Total      *int64        `json:"total,omitempty"`
	NextCursor *int64        `json:"next_cursor"`
}

func (u *Usecase) ListTasks(w http.ResponseWriter, r *http.Request) {
	filter, err := taskFilterFrom(r)
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	actor, err := u.storage.GetMembership(r.Context(), filter.TeamID, currentUserID(r))
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if err := actor.CanViewTeam(); err != nil {
		u.writeError(w, r, err)
		return
	}

	cacheVersion, cacheAvailable := u.cache.TeamVersion(r.Context(), filter.TeamID)
	if cacheAvailable {
		if page, found := u.cache.Get(r.Context(), cacheVersion, filter); found {
			writeJSON(w, http.StatusOK, page)
			return
		}
	}

	page, err := u.loadTaskPage(r.Context(), filter)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if cacheAvailable {
		u.cache.Set(r.Context(), cacheVersion, filter, page)
	}
	writeJSON(w, http.StatusOK, page)
}

func taskFilterFrom(r *http.Request) (TaskFilter, error) {
	teamID, err := queryID(r, "team_id")
	if err != nil {
		return TaskFilter{}, err
	}
	if teamID == nil {
		return TaskFilter{}, domain.Validation("team_id is required")
	}
	assigneeID, err := queryID(r, "assignee_id")
	if err != nil {
		return TaskFilter{}, err
	}
	page, err := pageParams(r)
	if err != nil {
		return TaskFilter{}, err
	}
	page, err = page.withDefaults(defaultTaskPageSize, maxTaskPageSize)
	if err != nil {
		return TaskFilter{}, err
	}

	filter := TaskFilter{
		TeamID:     *teamID,
		AssigneeID: assigneeID,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}
	if status := r.URL.Query().Get("status"); status != "" {
		taskStatus := domain.TaskStatus(status)
		if err := validStatus(taskStatus); err != nil {
			return TaskFilter{}, err
		}
		filter.Status = &taskStatus
	}
	cursor, err := queryID(r, "cursor")
	if err != nil {
		return TaskFilter{}, err
	}
	if cursor != nil {
		filter.BeforeID = *cursor
		filter.Offset = 0
	}
	return filter, nil
}

func (u *Usecase) loadTaskPage(ctx context.Context, filter TaskFilter) (*TaskPage, error) {
	tasks, hasNextPage, err := u.storage.ListTasks(ctx, filter)
	if err != nil {
		return nil, err
	}

	page := &TaskPage{Items: tasks, Limit: filter.Limit, Offset: filter.Offset}
	if hasNextPage {
		page.NextCursor = &tasks[len(tasks)-1].ID
	}

	usesCursor := filter.BeforeID != 0
	if !usesCursor {
		total, err := u.storage.CountTasks(ctx, filter)
		if err != nil {
			return nil, err
		}
		page.Total = &total
	}
	return page, nil
}
