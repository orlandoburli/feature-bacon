package bacon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client communicates with the Feature Bacon flag evaluation API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the API key used for authentication.
// The key is sent via the X-API-Key header on every request.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithHTTPClient replaces the default http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithTimeout overrides the default request timeout (5 s).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// NewClient returns a Client pointed at baseURL.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// EvaluationContext carries the subject and environment for flag evaluation.
type EvaluationContext struct {
	SubjectID   string         `json:"subjectId"`
	Environment string         `json:"environment,omitempty"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

// EvaluationResult is the server response for a single flag evaluation.
type EvaluationResult struct {
	TenantID string `json:"tenantId"`
	FlagKey  string `json:"flagKey"`
	Enabled  bool   `json:"enabled"`
	Variant  string `json:"variant"`
	Reason   string `json:"reason"`
}

type evaluateRequest struct {
	FlagKey string            `json:"flagKey"`
	Context EvaluationContext `json:"context"`
}

type batchEvaluateRequest struct {
	FlagKeys []string          `json:"flagKeys"`
	Context  EvaluationContext `json:"context"`
}

type batchEvaluateResponse struct {
	Results []EvaluationResult `json:"results"`
}

// Evaluate evaluates a single feature flag.
func (c *Client) Evaluate(ctx context.Context, flagKey string, evalCtx EvaluationContext) (*EvaluationResult, error) {
	body, err := json.Marshal(evaluateRequest{FlagKey: flagKey, Context: evalCtx})
	if err != nil {
		return nil, fmt.Errorf("bacon: marshal request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/evaluate", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var result EvaluationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("bacon: decode response: %w", err)
	}
	return &result, nil
}

// EvaluateBatch evaluates multiple feature flags in a single request.
func (c *Client) EvaluateBatch(ctx context.Context, flagKeys []string, evalCtx EvaluationContext) ([]EvaluationResult, error) {
	body, err := json.Marshal(batchEvaluateRequest{FlagKeys: flagKeys, Context: evalCtx})
	if err != nil {
		return nil, fmt.Errorf("bacon: marshal request: %w", err)
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/v1/evaluate/batch", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var batch batchEvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, fmt.Errorf("bacon: decode response: %w", err)
	}
	return batch.Results, nil
}

// IsEnabled is a convenience method that returns whether a flag is enabled,
// defaulting to false on any error.
func (c *Client) IsEnabled(ctx context.Context, flagKey string, evalCtx EvaluationContext) bool {
	result, err := c.Evaluate(ctx, flagKey, evalCtx)
	if err != nil || result == nil {
		return false
	}
	return result.Enabled
}

// GetVariant is a convenience method that returns the variant string,
// defaulting to "" on any error.
func (c *Client) GetVariant(ctx context.Context, flagKey string, evalCtx EvaluationContext) string {
	result, err := c.Evaluate(ctx, flagKey, evalCtx)
	if err != nil || result == nil {
		return ""
	}
	return result.Variant
}

// Healthy returns true when the server responds 200 to GET /healthz.
func (c *Client) Healthy(ctx context.Context) bool {
	resp, err := c.do(ctx, http.MethodGet, "/healthz", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}

// Ready returns true when the server responds 200 to GET /readyz.
func (c *Client) Ready(ctx context.Context) bool {
	resp, err := c.do(ctx, http.MethodGet, "/readyz", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("bacon: create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bacon: send request: %w", err)
	}
	return resp, nil
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Error{StatusCode: resp.StatusCode, Title: http.StatusText(resp.StatusCode)}
	}

	var apiErr Error
	if json.Unmarshal(data, &apiErr) == nil && apiErr.Title != "" {
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	return &Error{
		StatusCode: resp.StatusCode,
		Title:      http.StatusText(resp.StatusCode),
		Detail:     string(data),
	}
}
