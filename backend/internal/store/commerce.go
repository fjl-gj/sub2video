package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	// Canonical workflows/pricing are maintained by AlignSupplyChainH3.
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

func (s *Store) CreatePlatformWorkflow(ctx context.Context, code, name, description, modelRef, status, badge string) (*PlatformWorkflow, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code == "" || name == "" || modelRef == "" {
		return nil, errors.New("code, name and model required")
	}
	switch status {
	case "published", "draft", "disabled":
	default:
		status = "draft"
	}
	params, _ := json.Marshal(WorkflowParams{
		AspectRatios: []string{"16:9", "9:16", "1:1"},
		Durations:    []int{5, 10, 15}, DefaultDuration: 15,
		Resolutions: []string{"768P", "1080P"}, DefaultResolution: "768P",
	})
	tag, err := s.Pool.Exec(ctx, `
INSERT INTO platform_workflows (public_id, code, name, description, model_id, status, badge, sort_order, params)
SELECT $1, $2, $3, $4, m.id, $5, $6,
  COALESCE((SELECT MAX(sort_order)+1 FROM platform_workflows), 1), $7::jsonb
FROM models m WHERE m.public_id=$8 OR m.code=$8`,
		code, code, name, description, status, badge, string(params), modelRef)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, errors.New("model not found")
	}
	return s.GetPlatformWorkflow(ctx, code)
}

func (s *Store) UpdatePlatformWorkflow(ctx context.Context, publicID string, name, description, modelRef, status, badge *string) (*PlatformWorkflow, error) {
	w, err := s.GetPlatformWorkflow(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if name != nil {
		w.Name = strings.TrimSpace(*name)
	}
	if description != nil {
		w.Description = *description
	}
	if badge != nil {
		w.Badge = *badge
	}
	if status != nil {
		st := *status
		switch st {
		case "已上架", "published":
			st = "published"
		case "草稿", "draft":
			st = "draft"
		case "未开放", "disabled":
			st = "disabled"
		default:
			return nil, errors.New("invalid status")
		}
		w.Status = st
	}
	modelSel := w.ModelID
	if modelRef != nil && strings.TrimSpace(*modelRef) != "" {
		modelSel = strings.TrimSpace(*modelRef)
	}
	_, err = s.Pool.Exec(ctx, `
UPDATE platform_workflows w SET
  name=$1, description=$2, badge=$3, status=$4, updated_at=NOW(),
  model_id = (SELECT id FROM models WHERE public_id=$5 OR code=$5 LIMIT 1)
WHERE w.public_id=$6 OR w.code=$6`,
		w.Name, w.Description, w.Badge, w.Status, modelSel, publicID)
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

func (s *Store) CheapestActivePrice(ctx context.Context) (float64, error) {
	var price float64
	err := s.Pool.QueryRow(ctx, `SELECT MIN(price_per_sec::float8) FROM pricing_rules WHERE status='active'`).Scan(&price)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return price, err
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
