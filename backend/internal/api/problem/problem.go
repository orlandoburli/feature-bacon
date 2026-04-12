package problem

import (
	"encoding/json"
	"net/http"
)

const typePrefix = "https://bacon.dev/problems/"

type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

func (p *Problem) Error() string {
	if p.Detail != "" {
		return p.Detail
	}
	return p.Title
}

func Write(w http.ResponseWriter, p *Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p)
}

func Unauthorized(detail, instance string) *Problem {
	return &Problem{
		Type:     typePrefix + "unauthorized",
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: instance,
	}
}

func Forbidden(detail, instance string) *Problem {
	return &Problem{
		Type:     typePrefix + "forbidden",
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   detail,
		Instance: instance,
	}
}

func NotFound(detail, instance string) *Problem {
	return &Problem{
		Type:     typePrefix + "not-found",
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   detail,
		Instance: instance,
	}
}

func Conflict(detail, instance string) *Problem {
	return &Problem{
		Type:     typePrefix + "conflict",
		Title:    "Conflict",
		Status:   http.StatusConflict,
		Detail:   detail,
		Instance: instance,
	}
}

func ValidationError(detail, instance string) *Problem {
	return &Problem{
		Type:     typePrefix + "validation-error",
		Title:    "Validation Error",
		Status:   http.StatusUnprocessableEntity,
		Detail:   detail,
		Instance: instance,
	}
}

func ReadOnlyMode(instance string) *Problem {
	return &Problem{
		Type:     "/problems/read-only-mode",
		Title:    "Read-Only Mode",
		Status:   http.StatusConflict,
		Instance: instance,
	}
}

func InternalError(detail, instance string) *Problem {
	return &Problem{
		Type:     typePrefix + "internal-error",
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   detail,
		Instance: instance,
	}
}
