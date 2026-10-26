package usecase

import "net/http"

func (u *Usecase) TeamStats(w http.ResponseWriter, r *http.Request) {
	teamID, err := pathID(r, "id")
	if err != nil {
		u.writeError(w, r, err)
		return
	}

	actor, err := u.storage.GetMembership(r.Context(), teamID, currentUserID(r))
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	if err := actor.CanViewStats(); err != nil {
		u.writeError(w, r, err)
		return
	}

	stats, err := u.storage.TeamStats(r.Context(), teamID)
	if err != nil {
		u.writeError(w, r, err)
		return
	}
	stats.GeneratedAt = nowUTC()
	writeJSON(w, http.StatusOK, stats)
}
