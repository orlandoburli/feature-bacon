package routers

import (
	"github.com/orlandoburli/feature-bacon/api/adapters/handlers"
	"net/http"
)

func BuildRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", handlers.Ping)

	return mux
}
