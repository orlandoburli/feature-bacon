package middleware

import "net/http"

var Version = "dev"

func VersionHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Bacon-Version", Version)
		next.ServeHTTP(w, r)
	})
}
