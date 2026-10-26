package usecase

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/example/taskmanager/internal/domain"
)

func decodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return domain.Validation("invalid JSON: " + err.Error())
	}
	return nil
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.Validation(name + " must be a positive integer")
	}
	return id, nil
}

func queryID(r *http.Request, name string) (*int64, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return nil, domain.Validation(name + " must be a positive integer")
	}
	return &id, nil
}

func queryInt(r *http.Request, name string) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return 0, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, domain.Validation(name + " must be an integer")
	}
	return number, nil
}

func pageParams(r *http.Request) (Page, error) {
	limit, err := queryInt(r, "limit")
	if err != nil {
		return Page{}, err
	}
	offset, err := queryInt(r, "offset")
	if err != nil {
		return Page{}, err
	}
	return Page{Limit: limit, Offset: offset}, nil
}

type nullableID struct {
	provided bool
	id       *int64
}

func (n *nullableID) UnmarshalJSON(data []byte) error {
	n.provided = true
	return json.Unmarshal(data, &n.id)
}
