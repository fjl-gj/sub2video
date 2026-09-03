package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type WorkflowParams struct {
	AspectRatios    []string `json:"aspectRatios,omitempty"`
	Durations       []int    `json:"durations,omitempty"`
	DefaultDuration int      `json:"defaultDuration,omitempty"`
	Resolutions     []string `json:"resolutions,omitempty"`
	DefaultResolution string `json:"defaultResolution,omitempty"`
}

type PlatformWorkflow struct {
	ID          string
	Code        string
	Name        string
	Description string
	ModelID     string
	ModelCode   string
	CarrierCode string
	CarrierName string
	Status      string
	Badge       string
	SortOrder   int
	Params      WorkflowParams
	UpdatedAt   time.Time
}

type PricingRule struct {
	ID              string
	WorkflowID      string
	WorkflowCode    string
	WorkflowName    string
	Name            string
	Resolution      string
	PricePerSec     float64
	CostHintPerSec  float64
	Version         string
	Status          string
	UpdatedAt       time.Time
}

func (s *Store) SeedPlatformCommerce(ctx context.Context) error {
	var n int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM platform_workflows`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	type wfSeed struct {
		code, name, desc, modelCode, status, badge string
		order                                      int
		enabled                                    bool
	}
	seeds := []wfSeed{
		{"wf-h3-image", "图生视频", "一张图片 + 文字描述", "image-keyframes", "published", "", 1, true},
		{"wf-h3-text", "文生视频", "纯文字描述生成视频", "text", "published", "", 2, true},
		{"wf-h3-multi", "多图生视频", "多张参考图", "image-multi", "draft", "即将开放", 3, false},
	}
	params, _ := json.Marshal(WorkflowParams{
		AspectRatios: []string{"16:9", "9:16", "1:1"},
		Durations: []int{5, 10, 15}, DefaultDuration: 15,
		Resolutions: []string{"768P", "1080P"}, DefaultResolution: "768P",
	})

	for _, w := range seeds {
		status := w.status
		if !w.enabled {
			status = "draft"
		}
		_, err := s.Pool.Exec(ctx, `
INSERT INTO platform_workflows (public_id, code, name, description, model_id, status, badge, sort_order, params)
SELECT $1, $2, $3, $4, m.id, $5, $6, $7, $8::jsonb
FROM models m WHERE m.code = $9
ON CONFLICT (code) DO NOTHING`,
			w.code, w.code, w.name, w.desc, status, w.badge, w.order, string(params), w.modelCode)
		if err != nil {
			return err
		}
	}

	type priceSeed struct {
		wf, resolution, name string
		price, costHint      float64
		version              string
	}
	prices := []priceSeed{
		{"wf-h3-image", "768P", "H3 图生 / 768P", 0.50, 0.03, "v1.2"},
		{"wf-h3-image", "1080P", "H3 图生 / 1080P", 0.80, 0.04, "v1.0"},
		{"wf-h3-text", "768P", "H3 文生 / 768P", 0.50, 0.03, "v1.0"},
		{"wf-h3-text", "1080P", "H3 文生 / 1080P", 0.80, 0.04, "v1.0"},
		{"wf-h3-multi", "768P", "H3 多图 / 768P", 0.50, 0.03, "v0.1"},
	}
	for _, p := range prices {
		status := "active"
		if p.wf == "wf-h3-multi" {
			status = "draft"
		}
		pid := "pr_" + shortID()
		_, err := s.Pool.Exec(ctx, `
INSERT INTO pricing_rules (public_id, workflow_id, name, resolution, price_per_sec, cost_hint_per_sec, version, status)
SELECT $1, w.id, $2, $3, $4, $5, $6, $7
FROM platform_workflows w WHERE w.code = $8
ON CONFLICT (workflow_id, resolution, version) DO NOTHING`,
			pid, p.name, p.resolution, p.price, p.costHint, p.version, status, p.wf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListPlatformWorkflows(ctx context.Context, publishedOnly bool) ([]PlatformWorkflow, error) {
	q := `
SELECT w.public_id, w.code, w.name, w.description, COALESCE(m.public_id,''), COALESCE(m.code,''),
       COALESCE(c.code,''), COALESCE(c.name,''), w.status, w.badge, w.sort_order, w.params, w.updated_at
FROM platform_workflows w
LEFT JOIN models m ON m.id = w.model_id
LEFT JOIN carriers c ON c.id = m.carrier_id`
	if publishedOnly {
		q += ` WHERE w.status = 'published'`
	}
	q += ` ORDER BY w.sort_order, w.code`
	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PlatformWorkflow
	for rows.Next() {
		var w PlatformWorkflow
		var raw []byte
		if err := rows.Scan(&w.ID, &w.Code, &w.Name, &w.Description, &w.ModelID, &w.ModelCode,
			&w.CarrierCode, &w.CarrierName, &w.Status, &w.Badge, &w.SortOrder, &raw, &w.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &w.Params)
		items = append(items, w)
	}
	return items, rows.Err()
}

func (s *Store) GetPlatformWorkflow(ctx context.Context, code string) (*PlatformWorkflow, error) {
	var w PlatformWorkflow
	var raw []byte
	err := s.Pool.QueryRow(ctx, `
SELECT w.public_id, w.code, w.name, w.description, COALESCE(m.public_id,''), COALESCE(m.code,''),
       COALESCE(c.code,''), COALESCE(c.name,''), w.status, w.badge, w.sort_order, w.params, w.updated_at
FROM platform_workflows w
LEFT JOIN models m ON m.id = w.model_id
LEFT JOIN carriers c ON c.id = m.carrier_id
WHERE w.code = $1 OR w.public_id = $1`, code).Scan(
		&w.ID, &w.Code, &w.Name, &w.Description, &w.ModelID, &w.ModelCode,
		&w.CarrierCode, &w.CarrierName, &w.Status, &w.Badge, &w.SortOrder, &raw, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(raw, &w.Params)
	return &w, nil
}

func (s *Store) UpdatePlatformWorkflowStatus(ctx context.Context, publicID, status string) (*PlatformWorkflow, error) {
	_, err := s.Pool.Exec(ctx, `UPDATE platform_workflows SET status=$1, updated_at=NOW() WHERE public_id=$2 OR code=$2`, status, publicID)
	if err != nil {
		return nil, err
	}
	return s.GetPlatformWorkflow(ctx, publicID)
}

func (s *Store) ListPricingRules(ctx context.Context) ([]PricingRule, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT r.public_id, w.public_id, w.code, w.name, r.name, r.resolution,
       r.price_per_sec::float8, r.cost_hint_per_sec::float8, r.version, r.status, r.updated_at
FROM pricing_rules r
JOIN platform_workflows w ON w.id = r.workflow_id
ORDER BY w.sort_order, r.resolution, r.version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PricingRule
	for rows.Next() {
		var r PricingRule
		if err := rows.Scan(&r.ID, &r.WorkflowID, &r.WorkflowCode, &r.WorkflowName, &r.Name, &r.Resolution,
			&r.PricePerSec, &r.CostHintPerSec, &r.Version, &r.Status, &r.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func (s *Store) FindActivePrice(ctx context.Context, workflowCode, resolution string) (*PricingRule, error) {
	var r PricingRule
	err := s.Pool.QueryRow(ctx, `
SELECT r.public_id, w.public_id, w.code, w.name, r.name, r.resolution,
       r.price_per_sec::float8, r.cost_hint_per_sec::float8, r.version, r.status, r.updated_at
FROM pricing_rules r
JOIN platform_workflows w ON w.id = r.workflow_id
WHERE (w.code = $1 OR w.public_id = $1) AND r.resolution = $2 AND r.status = 'active'
ORDER BY r.updated_at DESC
LIMIT 1`, workflowCode, resolution).Scan(
		&r.ID, &r.WorkflowID, &r.WorkflowCode, &r.WorkflowName, &r.Name, &r.Resolution,
		&r.PricePerSec, &r.CostHintPerSec, &r.Version, &r.Status, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) UpdatePricingRulePrice(ctx context.Context, publicID string, price float64) (*PricingRule, error) {
	_, err := s.Pool.Exec(ctx, `UPDATE pricing_rules SET price_per_sec=$1, updated_at=NOW() WHERE public_id=$2`, price, publicID)
	if err != nil {
		return nil, err
	}
	rows, err := s.ListPricingRules(ctx)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].ID == publicID {
			return &rows[i], nil
		}
	}
	return nil, ErrNotFound
}

type AdminTaskRow struct {
	ID               string
	Title            string
	Workflow         string
	WorkflowPublicID string
	Status           string
	Progress         int
	DurationSec      int
	Resolution       string
	Cost             float64
	UserEmail        string
	UserName         string
	SupplierCode     string
	UpstreamTaskID   string
	UpstreamStatus   string
	FailReason       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (s *Store) ListAdminTasks(ctx context.Context, statusCNFilter string, page, pageSize int) ([]AdminTaskRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	where := `WHERE 1=1`
	args := []any{}
	argN := 1
	if statusCNFilter != "" && statusCNFilter != "全部" {
		where += fmt.Sprintf(` AND t.status = $%d`, argN)
		args = append(args, statusDB(statusCNFilter))
		argN++
	}
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM generation_tasks t `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := fmt.Sprintf(`
SELECT t.public_id, t.title, t.workflow, t.workflow_public_id, t.status, t.progress, t.duration_sec, t.resolution,
       t.cost::float8, u.email, u.display_name, t.supplier_code, t.upstream_task_id, t.upstream_status, t.fail_reason,
       t.created_at, t.updated_at
FROM generation_tasks t
JOIN users u ON u.id = t.user_id
%s ORDER BY t.updated_at DESC
LIMIT $%d OFFSET $%d`, where, argN, argN+1)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []AdminTaskRow
	for rows.Next() {
		var r AdminTaskRow
		var st string
		if err := rows.Scan(&r.ID, &r.Title, &r.Workflow, &r.WorkflowPublicID, &st, &r.Progress, &r.DurationSec, &r.Resolution,
			&r.Cost, &r.UserEmail, &r.UserName, &r.SupplierCode, &r.UpstreamTaskID, &r.UpstreamStatus, &r.FailReason,
			&r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, 0, err
		}
		r.Status = statusCN(st)
		items = append(items, r)
	}
	return items, total, rows.Err()
}
