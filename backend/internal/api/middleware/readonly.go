package middleware

import (
	"net/http"

	"github.com/orlandoburli/feature-bacon/internal/api/problem"
)

func ReadOnly(readOnly bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !readOnly {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				problem.Write(w, problem.ReadOnlyMode(r.URL.Path))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
