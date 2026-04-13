package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	bacon "github.com/orlandoburli/feature-bacon-go"
)

func main() {
	baconURL := envOr("BACON_URL", "http://localhost:8080")
	apiKey := os.Getenv("BACON_API_KEY")
	port := envOr("PORT", "3000")

	client := bacon.NewClient(baconURL, bacon.WithAPIKey(apiKey))

	http.HandleFunc("GET /", handleHome(client))
	http.HandleFunc("GET /products", handleProducts(client))
	http.HandleFunc("GET /health", handleHealth(client))

	log.Printf("Product service starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHome(client *bacon.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user")
		if userID == "" {
			userID = "anonymous"
		}

		ctx := bacon.EvaluationContext{
			SubjectID:   userID,
			Environment: envOr("ENVIRONMENT", "production"),
			Attributes:  map[string]any{"source": "web"},
		}

		results, err := client.EvaluateBatch(r.Context(), []string{
			"dark_mode",
			"new_pricing",
			"beta_features",
			"checkout_redesign",
		}, ctx)

		features := make(map[string]any)
		if err == nil {
			for _, res := range results {
				features[res.FlagKey] = map[string]any{
					"enabled": res.Enabled,
					"variant": res.Variant,
					"reason":  res.Reason,
				}
			}
		}

		respond(w, map[string]any{
			"service":  "product-catalog",
			"user":     userID,
			"features": features,
		})
	}
}

func handleProducts(client *bacon.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user")
		if userID == "" {
			userID = "anonymous"
		}

		ctx := bacon.EvaluationContext{
			SubjectID:   userID,
			Environment: envOr("ENVIRONMENT", "production"),
		}

		showNewPricing := client.IsEnabled(r.Context(), "new_pricing", ctx)
		checkoutVariant := client.GetVariant(r.Context(), "checkout_redesign", ctx)

		products := []map[string]any{
			{"id": 1, "name": "Widget Pro", "price": price(29.99, showNewPricing)},
			{"id": 2, "name": "Widget Basic", "price": price(9.99, showNewPricing)},
			{"id": 3, "name": "Widget Enterprise", "price": price(99.99, showNewPricing)},
		}

		respond(w, map[string]any{
			"products":         products,
			"checkoutVariant":  checkoutVariant,
			"newPricingActive": showNewPricing,
		})
	}
}

func handleHealth(client *bacon.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthy := client.Healthy(r.Context())
		status := "ok"
		if !healthy {
			status = "degraded"
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		respond(w, map[string]any{
			"status":       status,
			"baconHealthy": healthy,
		})
	}
}

func price(base float64, newPricing bool) float64 {
	if newPricing {
		return base * 0.9
	}
	return base
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func respond(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
