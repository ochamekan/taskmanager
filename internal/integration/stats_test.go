package integration

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/example/taskmanager/internal/auth"
	"github.com/example/taskmanager/internal/domain"
	"github.com/example/taskmanager/internal/usecase"
)

func TestTeamStatsReport(t *testing.T) {
	db := openTestDB(t)

	owner := createUser(t, db, "Owner")
	admin := createUser(t, db, "Admin")
	carol := createUser(t, db, "Carol")
	dave := createUser(t, db, "Dave")
	erin := createUser(t, db, "Erin")
	otherTeamOwner := createUser(t, db, "Frank")

	team := createTeam(t, db, "Core", owner)
	addMember(t, db, team, admin, domain.RoleAdmin)
	addMember(t, db, team, carol, domain.RoleMember)
	addMember(t, db, team, dave, domain.RoleMember)
	addMember(t, db, team, erin, domain.RoleMember)

	otherTeam := createTeam(t, db, "Other", otherTeamOwner)
	addMember(t, db, otherTeam, carol, domain.RoleMember)

	commentedTodo := createOpenTask(t, db, team, owner, domain.StatusTodo)
	createOpenTask(t, db, team, carol, domain.StatusTodo)
	commentedInProgress := createOpenTask(t, db, team, dave, domain.StatusInProgress)

	closedRecently := []closedTask{
		{assignee: carol, closedDaysAgo: 1, hoursToClose: 1},
		{assignee: carol, closedDaysAgo: 2, hoursToClose: 2},
		{assignee: carol, closedDaysAgo: 3, hoursToClose: 3},
		{assignee: dave, closedDaysAgo: 4, hoursToClose: 4},
		{assignee: dave, closedDaysAgo: 5, hoursToClose: 6},
		{assignee: erin, closedDaysAgo: 6, hoursToClose: 10},
	}
	closedBeforeLast30Days := closedTask{assignee: admin, closedDaysAgo: 40, hoursToClose: 48}
	for _, task := range append(closedRecently, closedBeforeLast30Days) {
		createClosedTask(t, db, team, owner, task)
	}

	addComment(t, db, commentedTodo, carol)
	addComment(t, db, commentedTodo, dave)
	addComment(t, db, commentedInProgress, owner)

	for range 10 {
		otherTeamTask := createClosedTask(t, db, otherTeam, otherTeamOwner,
			closedTask{assignee: otherTeamOwner, closedDaysAgo: 1, hoursToClose: 100})
		addComment(t, db, otherTeamTask, otherTeamOwner)
	}
	createClosedTask(t, db, otherTeam, otherTeamOwner, closedTask{assignee: carol, closedDaysAgo: 1, hoursToClose: 100})
	createOpenTask(t, db, otherTeam, otherTeamOwner, domain.StatusTodo)

	u := usecase.New(newStorage(db), nil, zap.NewNop(), testJWTSecret, time.Hour)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /teams/{id}/stats", u.RequireAuth(u.TeamStats))

	t.Run("report contains only this team's tasks", func(t *testing.T) {
		status, report := requestStats(t, mux, owner, team)
		if status != http.StatusOK {
			t.Fatalf("owner stats: status %d", status)
		}

		wantStatus := map[string]int64{"todo": 2, "in_progress": 1, "done": 7}
		for taskStatus, count := range wantStatus {
			if report.TasksByStatus[taskStatus] != count {
				t.Errorf("tasks_by_status[%s] = %d, want %d", taskStatus, report.TasksByStatus[taskStatus], count)
			}
		}
		if report.TotalTasks != 10 {
			t.Errorf("total_tasks = %d, want 10", report.TotalTasks)
		}

		wantTop := []struct {
			userID      int64
			closedTasks int64
		}{{carol, 3}, {dave, 2}, {erin, 1}}
		if len(report.TopAssignees30d) != len(wantTop) {
			t.Fatalf("top_assignees_30d = %+v, want %d entries", report.TopAssignees30d, len(wantTop))
		}
		for i, want := range wantTop {
			got := report.TopAssignees30d[i]
			if got.UserID != want.userID || got.ClosedTasks != want.closedTasks {
				t.Errorf("top[%d] = user %d with %d, want user %d with %d",
					i, got.UserID, got.ClosedTasks, want.userID, want.closedTasks)
			}
		}
		if report.TopAssignees30d[0].Name != "Carol" || report.TopAssignees30d[0].Email != "carol@example.com" {
			t.Errorf("top[0] user details = %+v", report.TopAssignees30d[0])
		}

		if report.ClosedTasksTotal != 7 {
			t.Errorf("closed_tasks_total = %d, want 7", report.ClosedTasksTotal)
		}
		wantAvgSeconds := float64((1+2+3+4+6+10+48)*3600) / 7
		if report.AvgCloseTimeSec == nil || math.Abs(*report.AvgCloseTimeSec-wantAvgSeconds) > 1 {
			t.Errorf("avg_close_time_seconds = %v, want %.1f", report.AvgCloseTimeSec, wantAvgSeconds)
		}

		if report.Comments.Total != 3 || report.Comments.TasksWithComments != 2 {
			t.Errorf("comments = %+v, want total 3 on 2 tasks", report.Comments)
		}
		if math.Abs(report.Comments.AvgPerTask-0.3) > 1e-9 {
			t.Errorf("comments.avg_per_task = %v, want 0.3", report.Comments.AvgPerTask)
		}
	})

	t.Run("only owner and admin can read the report", func(t *testing.T) {
		if status, _ := requestStats(t, mux, admin, team); status != http.StatusOK {
			t.Errorf("admin must see stats, got status %d", status)
		}
		for name, userID := range map[string]int64{"member": carol, "outsider": otherTeamOwner} {
			if status, _ := requestStats(t, mux, userID, team); status != http.StatusForbidden {
				t.Errorf("%s: expected 403, got %d", name, status)
			}
		}
	})

	t.Run("empty team has zero counts and no average", func(t *testing.T) {
		emptyTeam := createTeam(t, db, "Empty", owner)
		status, report := requestStats(t, mux, owner, emptyTeam)
		if status != http.StatusOK {
			t.Fatalf("empty team: status %d", status)
		}
		if report.TotalTasks != 0 || report.AvgCloseTimeSec != nil || len(report.TopAssignees30d) != 0 || report.Comments.Total != 0 {
			t.Errorf("empty team stats = %+v", report)
		}
	})
}

const testJWTSecret = "integration-test-secret-0123456789abcdef"

type statsReport struct {
	TasksByStatus   map[string]int64 `json:"tasks_by_status"`
	TotalTasks      int64            `json:"total_tasks"`
	TopAssignees30d []struct {
		UserID      int64  `json:"user_id"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		ClosedTasks int64  `json:"closed_tasks"`
	} `json:"top_assignees_30d"`
	ClosedTasksTotal int64    `json:"closed_tasks_total"`
	AvgCloseTimeSec  *float64 `json:"avg_close_time_seconds"`
	Comments         struct {
		Total             int64   `json:"total"`
		TasksWithComments int64   `json:"tasks_with_comments"`
		AvgPerTask        float64 `json:"avg_per_task"`
	} `json:"comments"`
}

func requestStats(t *testing.T, handler http.Handler, userID, teamID int64) (int, statsReport) {
	t.Helper()
	token, _, err := auth.NewToken(userID, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/teams/%d/stats", teamID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var report statsReport
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code, report
}
