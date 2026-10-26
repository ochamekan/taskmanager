package usecase

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/example/taskmanager/internal/domain"
)

var errTaskNotFound = domain.NotFound("task not found")

type createTaskRequest struct {
	TeamID      int64             `json:"team_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      domain.TaskStatus `json:"status"`
	AssigneeID  *int64            `json:"assignee_id"`
}

func (u *Usecase) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		u.writeError(w, r, err)
		return
	}
	actorID := currentUserID(r)
	task, err := newTask(actorID, req)
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	err = u.storage.InTx(r.Context(), func(tx Storage) error {
		actor, err := tx.GetMembership(r.Context(), task.TeamID, actorID)
		if err != nil {
			return err
		}
		if err := actor.CanCreateTask(); err != nil {
			return err
		}
		if task.AssigneeID != nil {
			if err := requireTeamMember(r.Context(), tx, task.TeamID, *task.AssigneeID); err != nil {
				return err
			}
		}
		if err := tx.CreateTask(r.Context(), task); err != nil {
			return err
		}
		return tx.AddTaskHistory(r.Context(), task.ID, actorID, creationChanges(task), task.CreatedAt)
	})
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	u.cache.InvalidateTeam(r.Context(), task.TeamID)
	writeJSON(w, http.StatusCreated, task)
}

func newTask(actorID int64, req createTaskRequest) (*domain.Task, error) {
	if req.TeamID <= 0 {
		return nil, domain.Validation("team_id is required")
	}
	if req.AssigneeID != nil && *req.AssigneeID <= 0 {
		return nil, domain.Validation("assignee_id must be a positive integer")
	}
	title, err := requiredText("title", req.Title, maxTitleLength)
	if err != nil {
		return nil, err
	}
	description, err := optionalText("description", req.Description, maxDescriptionLength)
	if err != nil {
		return nil, err
	}
	if req.Status == "" {
		req.Status = domain.StatusTodo
	}
	if err := validStatus(req.Status); err != nil {
		return nil, err
	}

	now := nowUTC()
	return &domain.Task{
		TeamID:      req.TeamID,
		Title:       title,
		Description: description,
		Status:      req.Status,
		CreatedBy:   actorID,
		AssigneeID:  req.AssigneeID,
		CreatedAt:   now,
		UpdatedAt:   now,
		ClosedAt:    closedAtForStatus(req.Status, now),
		Version:     1,
	}, nil
}

func creationChanges(t *domain.Task) domain.Changes {
	changes := domain.Changes{
		domain.FieldTitle:       {Old: nil, New: t.Title},
		domain.FieldDescription: {Old: nil, New: t.Description},
		domain.FieldStatus:      {Old: nil, New: t.Status},
		domain.FieldAssigneeID:  {Old: nil, New: t.AssigneeID},
	}
	if t.ClosedAt != nil {
		changes[domain.FieldClosedAt] = domain.FieldChange{Old: nil, New: t.ClosedAt}
	}
	return changes
}

func requireTeamMember(ctx context.Context, storage Storage, teamID, userID int64) error {
	membership, err := storage.GetMembership(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if !membership.IsMember {
		return domain.Validation("assignee must be a member of the task's team")
	}
	return nil
}

func checkTaskVisible(ctx context.Context, storage Storage, task *domain.Task, actorID int64) (domain.Membership, error) {
	actor, err := storage.GetMembership(ctx, task.TeamID, actorID)
	if err != nil {
		return domain.Membership{}, err
	}
	if actor.CanViewTeam() != nil {
		return domain.Membership{}, errTaskNotFound
	}
	return actor, nil
}

func getVisibleTask(ctx context.Context, storage Storage, actorID, taskID int64) (*domain.Task, domain.Membership, error) {
	task, err := storage.GetTask(ctx, taskID)
	if errors.Is(err, ErrRecordNotFound) {
		return nil, domain.Membership{}, errTaskNotFound
	}
	if err != nil {
		return nil, domain.Membership{}, err
	}
	actor, err := checkTaskVisible(ctx, storage, task, actorID)
	return task, actor, err
}

type updateTaskRequest struct {
	Title       *string            `json:"title"`
	Description *string            `json:"description"`
	Status      *domain.TaskStatus `json:"status"`
	AssigneeID  nullableID         `json:"assignee_id"`
}

func (u *Usecase) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathID(r, "id")
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	var req updateTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		u.writeError(w, r, err)
		return
	}
	req, err = validateUpdate(req)
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	actorID := currentUserID(r)
	var updated domain.Task
	var changedFields []string
	err = u.storage.InTx(r.Context(), func(tx Storage) error {
		current, err := tx.GetTaskForUpdate(r.Context(), taskID)
		if errors.Is(err, ErrRecordNotFound) {
			return errTaskNotFound
		}
		if err != nil {
			return err
		}
		actor, err := checkTaskVisible(r.Context(), tx, current, actorID)
		if err != nil {
			return err
		}

		var changes domain.Changes
		updated, changes, changedFields = applyUpdate(*current, req, nowUTC())
		if err := actor.CanUpdateTask(*current, changedFields); err != nil {
			return err
		}
		if len(changedFields) == 0 {
			return nil
		}
		if _, reassigned := changes[domain.FieldAssigneeID]; reassigned && updated.AssigneeID != nil {
			if err := requireTeamMember(r.Context(), tx, updated.TeamID, *updated.AssigneeID); err != nil {
				return err
			}
		}

		updated.Version++
		if err := tx.UpdateTask(r.Context(), &updated); err != nil {
			return err
		}
		return tx.AddTaskHistory(r.Context(), updated.ID, actorID, changes, updated.UpdatedAt)
	})
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	if len(changedFields) > 0 {
		u.cache.InvalidateTeam(r.Context(), updated.TeamID)
	}
	writeJSON(w, http.StatusOK, updated)
}

func validateUpdate(req updateTaskRequest) (updateTaskRequest, error) {
	if req.AssigneeID.id != nil && *req.AssigneeID.id <= 0 {
		return req, domain.Validation("assignee_id must be a positive integer")
	}
	if req.Title != nil {
		title, err := requiredText("title", *req.Title, maxTitleLength)
		if err != nil {
			return req, err
		}
		req.Title = &title
	}
	if req.Description != nil {
		description, err := optionalText("description", *req.Description, maxDescriptionLength)
		if err != nil {
			return req, err
		}
		req.Description = &description
	}
	if req.Status != nil {
		if err := validStatus(*req.Status); err != nil {
			return req, err
		}
	}
	return req, nil
}

func applyUpdate(task domain.Task, req updateTaskRequest, now time.Time) (updated domain.Task, changes domain.Changes, changedFields []string) {
	changes = domain.Changes{}
	recordHistory := func(field string, oldValue, newValue any) {
		changes[field] = domain.FieldChange{Old: oldValue, New: newValue}
	}
	recordChangedField := func(field string, oldValue, newValue any) {
		recordHistory(field, oldValue, newValue)
		changedFields = append(changedFields, field)
	}
	current := task

	if req.Title != nil && *req.Title != current.Title {
		task.Title = *req.Title
		recordChangedField(domain.FieldTitle, current.Title, task.Title)
	}
	if req.Description != nil && *req.Description != current.Description {
		task.Description = *req.Description
		recordChangedField(domain.FieldDescription, current.Description, task.Description)
	}
	if req.Status != nil && *req.Status != current.Status {
		task.Status = *req.Status
		recordChangedField(domain.FieldStatus, current.Status, task.Status)

		task.ClosedAt = closedAtForStatus(task.Status, now)
		if current.ClosedAt != nil || task.ClosedAt != nil {
			recordHistory(domain.FieldClosedAt, current.ClosedAt, task.ClosedAt)
		}
	}
	if req.AssigneeID.provided && !sameID(current.AssigneeID, req.AssigneeID.id) {
		task.AssigneeID = req.AssigneeID.id
		recordChangedField(domain.FieldAssigneeID, current.AssigneeID, task.AssigneeID)
	}

	if len(changedFields) > 0 {
		task.UpdatedAt = now
	}
	return task, changes, changedFields
}

func closedAtForStatus(status domain.TaskStatus, now time.Time) *time.Time {
	if status == domain.StatusDone {
		return &now
	}
	return nil
}

func sameID(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
