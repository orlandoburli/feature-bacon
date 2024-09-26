package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPing(t *testing.T) {

	tests := []struct {
		name     string
		want     int
		wantBody PingResponse
	}{
		{
			name:     "Test Ping Normal Response",
			want:     http.StatusOK,
			wantBody: PingResponse{Message: "pong"},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				t.Fatalf("Could not create request: %v", err)
			}
			w := httptest.NewRecorder()

			Ping(w, req)

			res := w.Result()
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					fmt.Printf("Could not close response body: %v", err)
				}
			}(res.Body)

			if res.StatusCode != tt.want {
				t.Errorf("Ping() status = %v; want %v", res.StatusCode, tt.want)
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("Could not read response: %v", err)
			}

			var payload PingResponse
			err = json.Unmarshal(body, &payload)
			if err != nil {
				t.Fatalf("Could not unmarshal response: %v", err)
			}

			if payload.Message != tt.wantBody.Message {
				t.Errorf("Ping() message = %v; want %v", payload.Message, tt.wantBody.Message)
			}
		})
	}
}
