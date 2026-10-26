package usecase

import (
	"net/http"

	"github.com/example/taskmanager/internal/domain"
)

const (
	defaultActivityPageSize = 50
	maxActivityPageSize     = 200
)

func (u *Usecase) TaskHistory(w http.ResponseWriter, r *http.Request) {
	taskID, page, err := taskActivityParams(r)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if _, _, err := getVisibleTask(r.Context(), u.storage, currentUserID(r), taskID); err != nil {
		u.writeError(w, r, err)
		return
	}

	entries, err := u.storage.ListTaskHistory(r.Context(), taskID, page)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": entries})
}

type addCommentRequest struct {
	Content string `json:"content"`
}

func (u *Usecase) AddComment(w http.ResponseWriter, r *http.Request) {
	taskID, err := pathID(r, "id")
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	var req addCommentRequest
	if err := decodeJSON(r, &req); err != nil {
		u.writeError(w, r, err)
		return
	}
	content, err := requiredText("content", req.Content, maxCommentLength)
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	actorID := currentUserID(r)
	_, actor, err := getVisibleTask(r.Context(), u.storage, actorID, taskID)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if err := actor.CanComment(); err != nil {
		u.writeError(w, r, err)
		return
	}

	comment := &domain.Comment{TaskID: taskID, UserID: actorID, Content: content, CreatedAt: nowUTC()}
	if err := u.storage.AddComment(r.Context(), comment); err != nil {
		u.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (u *Usecase) ListComments(w http.ResponseWriter, r *http.Request) {
	taskID, page, err := taskActivityParams(r)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if _, _, err := getVisibleTask(r.Context(), u.storage, currentUserID(r), taskID); err != nil {
		u.writeError(w, r, err)
		return
	}

	comments, err := u.storage.ListComments(r.Context(), taskID, page)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": comments})
}

func taskActivityParams(r *http.Request) (taskID int64, page Page, err error) {
	if taskID, err = pathID(r, "id"); err != nil {
		return 0, Page{}, err
	}
	if page, err = pageParams(r); err != nil {
		return 0, Page{}, err
	}
	page, err = page.withDefaults(defaultActivityPageSize, maxActivityPageSize)
	return taskID, page, err
}
