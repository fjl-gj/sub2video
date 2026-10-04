package autodl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	AuthHeader string
	HTTP       *http.Client
}

type SubmitResult struct {
	TaskID    string
	Status    string
	CreatedAt string
	Raw       map[string]any
}

type QueryResult struct {
	TaskID    string
	Status    string
	Duration  int
	StartedAt string
	Results   []ResultItem
	Raw       map[string]any
}

// SubmitError records whether a provider submission may have crossed the
// provider boundary. Transport failures, 5xx responses, and malformed success
// responses remain uncertain and must be reconciled instead of refunded
// automatically. Only failures proven to occur before a request is sent are
// definite here. HTTP 4xx (including timeout/conflict) is not proof of rejection;
// no provider rejection-code taxonomy has yet been verified for this adapter.
type SubmitError struct {
	Err               error
	Uncertain         bool
	ProviderRequestID string
}

func (e *SubmitError) Error() string { return e.Err.Error() }
func (e *SubmitError) Unwrap() error { return e.Err }

func IsSubmitUncertain(err error) bool {
	if err == nil {
		return false
	}
	var submitErr *SubmitError
	if errors.As(err, &submitErr) {
		return submitErr.Uncertain
	}
	// Unknown adapter errors are conservatively treated as uncertain. A caller
	// must opt into a definite failure by returning SubmitError explicitly.
	return true
}

type ResultItem struct {
	URL      string
	Type     string
	FileType string
}

// ErrAPIExecutionUnsupported is returned until the provider's API submit and
// poll contract is verified from an authoritative adapter specification. The
// worker must not invent an endpoint and risk charging a task it cannot track.
var ErrAPIExecutionUnsupported = errors.New("autodl API execution contract is not verified")

func New(baseURL, token, authHeader string) *Client {
	if baseURL == "" {
		baseURL = "https://autodl.art"
	}
	if authHeader == "" {
		authHeader = "Authorization"
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		AuthHeader: authHeader,
		HTTP:       &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Submit(ctx context.Context, workflowID string, body map[string]any) (*SubmitResult, error) {
	if c.Token == "" {
		return nil, &SubmitError{Err: errors.New("autodl token not configured")}
	}
	url := fmt.Sprintf("%s/api/v1/comfyui/comfyui_workflow/%s", c.BaseURL, workflowID)
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &SubmitError{Err: fmt.Errorf("autodl submit encode: %w", err)}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, &SubmitError{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(c.AuthHeader, c.Token)

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &SubmitError{Err: err, Uncertain: true}
	}
	defer res.Body.Close()
	rawBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, &SubmitError{Err: fmt.Errorf("autodl submit read: %w", err), Uncertain: true}
	}
	var envelope struct {
		Code string         `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return nil, &SubmitError{Err: fmt.Errorf("autodl submit decode: %w body=%s", err, truncate(string(rawBody), 300)), Uncertain: true}
	}
	if res.StatusCode >= 300 || !strings.EqualFold(envelope.Code, "Success") {
		providerRequestID, _ := envelope.Data["task_id"].(string)
		return nil, &SubmitError{
			Err:               fmt.Errorf("autodl submit failed status=%d code=%s msg=%s", res.StatusCode, envelope.Code, envelope.Msg),
			Uncertain:         true,
			ProviderRequestID: providerRequestID,
		}
	}
	out := &SubmitResult{Raw: envelope.Data}
	if v, ok := envelope.Data["task_id"].(string); ok {
		out.TaskID = v
	}
	if v, ok := envelope.Data["status"].(string); ok {
		out.Status = v
	}
	if out.Status == "" {
		if v, ok := envelope.Data["state"].(string); ok {
			out.Status = v
		}
	}
	if v, ok := envelope.Data["created_at"].(string); ok {
		out.CreatedAt = v
	}
	if out.TaskID == "" {
		return nil, &SubmitError{Err: errors.New("autodl submit missing task_id"), Uncertain: true}
	}
	return out, nil
}

// SubmitAPI is the provider API execution path. It intentionally shares the
// response contract with workflow submission so the worker can route either
// execution type from an immutable task snapshot.
func (c *Client) SubmitAPI(ctx context.Context, modelCode, version string, body map[string]any) (*SubmitResult, error) {
	return nil, ErrAPIExecutionUnsupported
}

func (c *Client) submitJSON(ctx context.Context, url string, payload map[string]any) (*SubmitResult, error) {
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(c.AuthHeader, c.Token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	var envelope struct {
		Code string         `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("api submit decode: %w", err)
	}
	if res.StatusCode >= 300 || (envelope.Code != "" && !strings.EqualFold(envelope.Code, "Success")) {
		return nil, fmt.Errorf("api submit failed status=%d code=%s msg=%s", res.StatusCode, envelope.Code, envelope.Msg)
	}
	out := &SubmitResult{Raw: envelope.Data}
	if v, ok := envelope.Data["task_id"].(string); ok {
		out.TaskID = v
	}
	if out.TaskID == "" {
		if v, ok := envelope.Data["id"].(string); ok {
			out.TaskID = v
		}
	}
	if v, ok := envelope.Data["status"].(string); ok {
		out.Status = v
	}
	if out.TaskID == "" {
		return nil, fmt.Errorf("api submit missing task id")
	}
	return out, nil
}

func (c *Client) Query(ctx context.Context, taskID string) (*QueryResult, error) {
	return c.queryJSON(ctx, fmt.Sprintf("%s/api/v1/comfyui/comfyui_workflow/result/%s", c.BaseURL, taskID))
}

// QueryAPI is the API execution poll endpoint. API executions do not share
// the workflow result URL; keeping the adapter methods separate prevents a
// workflow path from being silently used for a direct model execution.
func (c *Client) QueryAPI(ctx context.Context, taskID string) (*QueryResult, error) {
	return nil, ErrAPIExecutionUnsupported
}

func (c *Client) queryJSON(ctx context.Context, url string) (*QueryResult, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("autodl token not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(c.AuthHeader, c.Token)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	rawBody, _ := io.ReadAll(res.Body)
	var envelope struct {
		Code string         `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return nil, fmt.Errorf("autodl query decode: %w", err)
	}
	if res.StatusCode >= 300 || !strings.EqualFold(envelope.Code, "Success") {
		return nil, fmt.Errorf("autodl query failed status=%d code=%s msg=%s", res.StatusCode, envelope.Code, envelope.Msg)
	}
	out := &QueryResult{Raw: envelope.Data}
	if v, ok := envelope.Data["task_id"].(string); ok {
		out.TaskID = v
	}
	if v, ok := envelope.Data["status"].(string); ok {
		out.Status = v
	}
	if v, ok := envelope.Data["started_at"].(string); ok {
		out.StartedAt = v
	}
	switch d := envelope.Data["duration"].(type) {
	case float64:
		out.Duration = int(d)
	case int:
		out.Duration = d
	}
	if arr, ok := envelope.Data["results"].([]any); ok {
		for _, item := range arr {
			m, _ := item.(map[string]any)
			if m == nil {
				continue
			}
			ri := ResultItem{}
			if v, ok := m["url"].(string); ok {
				ri.URL = v
			}
			if v, ok := m["type"].(string); ok {
				ri.Type = v
			}
			if v, ok := m["file_type"].(string); ok {
				ri.FileType = v
			}
			out.Results = append(out.Results, ri)
		}
	}
	// Direct API providers commonly return one URL or an output object rather
	// than the workflow results array. Normalize those forms to ResultItem so
	// delivery does not depend on the execution type.
	if len(out.Results) == 0 {
		for _, key := range []string{"result_url", "video_url", "output_url"} {
			if url, ok := envelope.Data[key].(string); ok && url != "" {
				out.Results = append(out.Results, ResultItem{URL: url, Type: "video", FileType: "mp4"})
				break
			}
		}
	}
	if len(out.Results) == 0 {
		for _, key := range []string{"output", "result"} {
			if obj, ok := envelope.Data[key].(map[string]any); ok {
				for _, urlKey := range []string{"url", "result_url", "video_url"} {
					if url, ok := obj[urlKey].(string); ok && url != "" {
						out.Results = append(out.Results, ResultItem{URL: url, Type: "video", FileType: "mp4"})
						break
					}
				}
				if len(out.Results) > 0 {
					break
				}
			}
		}
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
