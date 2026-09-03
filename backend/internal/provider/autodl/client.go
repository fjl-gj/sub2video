package autodl

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

type ResultItem struct {
	URL      string
	Type     string
	FileType string
}

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
		return nil, fmt.Errorf("autodl token not configured")
	}
	url := fmt.Sprintf("%s/api/v1/comfyui/comfyui_workflow/%s", c.BaseURL, workflowID)
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
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
	rawBody, _ := io.ReadAll(res.Body)
	var envelope struct {
		Code string         `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return nil, fmt.Errorf("autodl submit decode: %w body=%s", err, truncate(string(rawBody), 300))
	}
	if res.StatusCode >= 300 || !strings.EqualFold(envelope.Code, "Success") {
		return nil, fmt.Errorf("autodl submit failed status=%d code=%s msg=%s", res.StatusCode, envelope.Code, envelope.Msg)
	}
	out := &SubmitResult{Raw: envelope.Data}
	if v, ok := envelope.Data["task_id"].(string); ok {
		out.TaskID = v
	}
	if v, ok := envelope.Data["status"].(string); ok {
		out.Status = v
	}
	if v, ok := envelope.Data["created_at"].(string); ok {
		out.CreatedAt = v
	}
	if out.TaskID == "" {
		return nil, fmt.Errorf("autodl submit missing task_id")
	}
	return out, nil
}

func (c *Client) Query(ctx context.Context, taskID string) (*QueryResult, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("autodl token not configured")
	}
	url := fmt.Sprintf("%s/api/v1/comfyui/comfyui_workflow/result/%s", c.BaseURL, taskID)
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
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
