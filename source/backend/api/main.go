package main

import (
	"fmt"
	"github.com/orlandoburli/feature-bacon/api/adapters/routers"
	"net/http"
)

func main() {
	port := 8080

	mux := routers.BuildRoutes()

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux)

	fmt.Printf("Server started on port %d\n", port)

	if err != nil {
		fmt.Printf("Error starting server %s\n", err)
		return
	}

}
