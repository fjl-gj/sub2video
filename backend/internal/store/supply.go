package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sbu2video/backend/internal/crypto"
)

type Carrier struct {
	ID          string
	Code        string
	Name        string
	Description string
	Enabled     bool
	SortOrder   int
}

type Model struct {
	ID                string
	CarrierID         string
	CarrierCode       string
	Code              string
	Name              string
	Description       string
	ClientWorkflowID  string
	Enabled           bool
	SortOrder         int
}

type Supplier struct {
	ID             string
	Code           string
	Name           string
	BaseURL        string
	AuthHeader     string
	TokenEncrypted string
	TokenConfigured bool
	Enabled        bool
	Status         string
	UpdatedAt      time.Time
}

type RouteConditions struct {
	MaxDuration    int      `json:"max_duration,omitempty"`
	MinDuration    int      `json:"min_duration,omitempty"`
	Resolutions    []string `json:"resolutions,omitempty"`
	RequireSource  []string `json:"require_source,omitempty"` // text | upload | asset
	AspectRatios   []string `json:"aspect_ratios,omitempty"`
}

type SupplierRoute struct {
	ID                 string
	SupplierID         string
	SupplierCode       string
	SupplierName       string
	SupplierBaseURL    string
	SupplierAuthHeader string
	TokenEncrypted     string
	ModelID            string
	ModelCode          string
	ClientWorkflowID   string
	WorkflowID         string
	Name               string
	Enabled            bool
	CostPerSecPeak     float64
	CostPerSecOffpeak  float64
	Conditions         RouteConditions
	Priority           int
}

func (s *Store) SeedSupplyChain(ctx context.Context, secretKey string) error {
	var n int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM carriers`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	carrierID := "car_" + shortID()
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO carriers (public_id, code, name, description, enabled, sort_order)
VALUES ($1, 'minimax-h3', 'MiniMax H3', 'MiniMax H3 视频生成能力', TRUE, 1)`, carrierID); err != nil {
		return err
	}

	type modelSeed struct {
		code, name, desc, wf string
		order                int
	}
	models := []modelSeed{
		{"text", "文生视频", "纯文字描述生成视频", "wf-h3-text", 1},
		{"image-keyframes", "首尾帧图生视频", "首帧/尾帧图片 + 提示词", "wf-h3-image", 2},
		{"image-multi", "多图参考生视频", "多张参考图 + 提示词", "wf-h3-multi", 3},
	}
	modelIDs := map[string]string{}
	for _, m := range models {
		id := "mdl_" + shortID()
		modelIDs[m.code] = id
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO models (public_id, carrier_id, code, name, description, client_workflow_id, enabled, sort_order)
SELECT $1, id, $2, $3, $4, $5, TRUE, $6 FROM carriers WHERE public_id = $7`,
			id, m.code, m.name, m.desc, m.wf, m.order, carrierID); err != nil {
			return err
		}
	}

	supplierID := "sup_" + shortID()
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO suppliers (public_id, code, name, base_url, auth_header, token_encrypted, enabled, status)
VALUES ($1, 'autodl', 'AutoDL', 'https://autodl.art', 'Authorization', '', TRUE, 'unknown')`, supplierID); err != nil {
		return err
	}

	type routeSeed struct {
		modelCode, workflowID, name string
		peak, off                   float64
		cond                        RouteConditions
		enabled                     bool
	}
	routes := []routeSeed{
		{"text", "minimax_h3_lightx2v_no_pic", "H3文生(lightx2v)", 0.03, 0.02, RouteConditions{MaxDuration: 15, Resolutions: []string{"480P", "768P"}, RequireSource: []string{"text"}}, true},
		{"text", "minimax_h3_b99_001", "H3文生(b99)", 0.03, 0.02, RouteConditions{MaxDuration: 15, Resolutions: []string{"768P"}, RequireSource: []string{"text"}}, true},
		{"image-keyframes", "minimax_h3_lightx2v", "H3首尾帧(lightx2v)", 0.03, 0.02, RouteConditions{MaxDuration: 10, Resolutions: []string{"480P", "768P"}, RequireSource: []string{"upload", "asset"}}, true},
		{"image-keyframes", "minimax_h3_b99_002", "H3首尾帧(b99)", 0.03, 0.02, RouteConditions{MaxDuration: 15, Resolutions: []string{"768P"}, RequireSource: []string{"upload", "asset"}}, true},
		{"image-multi", "minimax_h3_lightx2v_v5_15s", "H3多图15s", 0.03, 0.02, RouteConditions{MaxDuration: 15, Resolutions: []string{"480P", "768P"}, RequireSource: []string{"upload", "asset"}}, false},
	}
	for i, r := range routes {
		cond, _ := json.Marshal(r.cond)
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO supplier_routes (
  public_id, supplier_id, model_id, workflow_id, name, enabled,
  cost_per_sec_peak, cost_per_sec_offpeak, conditions, priority
) SELECT $1, s.id, m.id, $2, $3, $4, $5, $6, $7::jsonb, $8
FROM suppliers s, models m
WHERE s.public_id = $9 AND m.public_id = $10`,
			"rte_"+shortID(), r.workflowID, r.name, r.enabled, r.peak, r.off, string(cond), 100+i,
			supplierID, modelIDs[r.modelCode]); err != nil {
			return err
		}
	}
	_ = secretKey
	return nil
}

func (s *Store) ListSuppliers(ctx context.Context) ([]Supplier, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT public_id, code, name, base_url, auth_header, token_encrypted, enabled, status, updated_at
FROM suppliers ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Supplier
	for rows.Next() {
		var sp Supplier
		if err := rows.Scan(&sp.ID, &sp.Code, &sp.Name, &sp.BaseURL, &sp.AuthHeader, &sp.TokenEncrypted, &sp.Enabled, &sp.Status, &sp.UpdatedAt); err != nil {
			return nil, err
		}
		sp.TokenConfigured = sp.TokenEncrypted != ""
		items = append(items, sp)
	}
	return items, rows.Err()
}

func (s *Store) GetSupplierByCode(ctx context.Context, code string) (*Supplier, error) {
	var sp Supplier
	err := s.Pool.QueryRow(ctx, `
SELECT public_id, code, name, base_url, auth_header, token_encrypted, enabled, status, updated_at
FROM suppliers WHERE code = $1`, code).Scan(
		&sp.ID, &sp.Code, &sp.Name, &sp.BaseURL, &sp.AuthHeader, &sp.TokenEncrypted, &sp.Enabled, &sp.Status, &sp.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sp.TokenConfigured = sp.TokenEncrypted != ""
	return &sp, nil
}

func (s *Store) UpdateSupplier(ctx context.Context, publicID string, name, baseURL *string, enabled *bool, tokenPlain, secretKey string) (*Supplier, error) {
	sp, err := s.getSupplierByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if name != nil {
		sp.Name = *name
	}
	if baseURL != nil {
		sp.BaseURL = *baseURL
	}
	if enabled != nil {
		sp.Enabled = *enabled
	}
	if tokenPlain != "" {
		enc, err := crypto.EncryptAESGCM(secretKey, tokenPlain)
		if err != nil {
			return nil, err
		}
		sp.TokenEncrypted = enc
	}
	_, err = s.Pool.Exec(ctx, `
UPDATE suppliers SET name=$1, base_url=$2, enabled=$3, token_encrypted=$4, updated_at=NOW()
WHERE public_id=$5`, sp.Name, sp.BaseURL, sp.Enabled, sp.TokenEncrypted, publicID)
	if err != nil {
		return nil, err
	}
	sp.TokenConfigured = sp.TokenEncrypted != ""
	return sp, nil
}

func (s *Store) getSupplierByPublicID(ctx context.Context, publicID string) (*Supplier, error) {
	var sp Supplier
	err := s.Pool.QueryRow(ctx, `
SELECT public_id, code, name, base_url, auth_header, token_encrypted, enabled, status, updated_at
FROM suppliers WHERE public_id = $1`, publicID).Scan(
		&sp.ID, &sp.Code, &sp.Name, &sp.BaseURL, &sp.AuthHeader, &sp.TokenEncrypted, &sp.Enabled, &sp.Status, &sp.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sp.TokenConfigured = sp.TokenEncrypted != ""
	return &sp, nil
}

func (s *Store) ListRoutes(ctx context.Context) ([]SupplierRoute, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT r.public_id, s.public_id, s.code, s.name, s.base_url, s.auth_header, s.token_encrypted,
       m.public_id, m.code, m.client_workflow_id, r.workflow_id, r.name, r.enabled,
       r.cost_per_sec_peak::float8, r.cost_per_sec_offpeak::float8, r.conditions, r.priority
FROM supplier_routes r
JOIN suppliers s ON s.id = r.supplier_id
JOIN models m ON m.id = r.model_id
ORDER BY m.sort_order, r.priority, r.cost_per_sec_peak`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SupplierRoute
	for rows.Next() {
		var r SupplierRoute
		var condRaw []byte
		if err := rows.Scan(
			&r.ID, &r.SupplierID, &r.SupplierCode, &r.SupplierName, &r.SupplierBaseURL, &r.SupplierAuthHeader, &r.TokenEncrypted,
			&r.ModelID, &r.ModelCode, &r.ClientWorkflowID, &r.WorkflowID, &r.Name, &r.Enabled,
			&r.CostPerSecPeak, &r.CostPerSecOffpeak, &condRaw, &r.Priority,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(condRaw, &r.Conditions)
		items = append(items, r)
	}
	return items, rows.Err()
}

func (s *Store) UpdateRoute(ctx context.Context, publicID string, enabled *bool, peak, off *float64, conditions *RouteConditions, workflowID, name *string) (*SupplierRoute, error) {
	routes, err := s.ListRoutes(ctx)
	if err != nil {
		return nil, err
	}
	var found *SupplierRoute
	for i := range routes {
		if routes[i].ID == publicID {
			found = &routes[i]
			break
		}
	}
	if found == nil {
		return nil, ErrNotFound
	}
	if enabled != nil {
		found.Enabled = *enabled
	}
	if peak != nil {
		found.CostPerSecPeak = *peak
	}
	if off != nil {
		found.CostPerSecOffpeak = *off
	}
	if conditions != nil {
		found.Conditions = *conditions
	}
	if workflowID != nil {
		found.WorkflowID = *workflowID
	}
	if name != nil {
		found.Name = *name
	}
	cond, _ := json.Marshal(found.Conditions)
	_, err = s.Pool.Exec(ctx, `
UPDATE supplier_routes SET enabled=$1, cost_per_sec_peak=$2, cost_per_sec_offpeak=$3,
  conditions=$4::jsonb, workflow_id=$5, name=$6, updated_at=NOW()
WHERE public_id=$7`, found.Enabled, found.CostPerSecPeak, found.CostPerSecOffpeak, string(cond), found.WorkflowID, found.Name, publicID)
	if err != nil {
		return nil, err
	}
	return found, nil
}

func (s *Store) ListCarriersWithModels(ctx context.Context) ([]Carrier, []Model, error) {
	crows, err := s.Pool.Query(ctx, `SELECT public_id, code, name, description, enabled, sort_order FROM carriers ORDER BY sort_order`)
	if err != nil {
		return nil, nil, err
	}
	defer crows.Close()
	var carriers []Carrier
	for crows.Next() {
		var c Carrier
		if err := crows.Scan(&c.ID, &c.Code, &c.Name, &c.Description, &c.Enabled, &c.SortOrder); err != nil {
			return nil, nil, err
		}
		carriers = append(carriers, c)
	}
	mrows, err := s.Pool.Query(ctx, `
SELECT m.public_id, c.public_id, c.code, m.code, m.name, m.description, m.client_workflow_id, m.enabled, m.sort_order
FROM models m JOIN carriers c ON c.id = m.carrier_id ORDER BY c.sort_order, m.sort_order`)
	if err != nil {
		return nil, nil, err
	}
	defer mrows.Close()
	var models []Model
	for mrows.Next() {
		var m Model
		if err := mrows.Scan(&m.ID, &m.CarrierID, &m.CarrierCode, &m.Code, &m.Name, &m.Description, &m.ClientWorkflowID, &m.Enabled, &m.SortOrder); err != nil {
			return nil, nil, err
		}
		models = append(models, m)
	}
	return carriers, models, nil
}

func (s *Store) PromoteUserRole(ctx context.Context, email, role string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET role=$1, updated_at=NOW() WHERE email=$2 AND deleted_at IS NULL`, role, email)
	return err
}

func (s *Store) CreateSupplier(ctx context.Context, code, name, baseURL, tokenPlain, secretKey string, enabled bool) (*Supplier, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return nil, errors.New("code and name required")
	}
	if baseURL == "" {
		baseURL = "https://autodl.art"
	}
	enc := ""
	if tokenPlain != "" {
		var err error
		enc, err = crypto.EncryptAESGCM(secretKey, tokenPlain)
		if err != nil {
			return nil, err
		}
	}
	id := "sup_" + shortID()
	_, err := s.Pool.Exec(ctx, `
INSERT INTO suppliers (public_id, code, name, base_url, auth_header, token_encrypted, enabled, status)
VALUES ($1,$2,$3,$4,'Authorization',$5,$6,'unknown')`, id, code, name, baseURL, enc, enabled)
	if err != nil {
		return nil, err
	}
	return s.getSupplierByPublicID(ctx, id)
}

func (s *Store) CreateModel(ctx context.Context, carrierPublicID, code, name, description, clientWorkflowID string) (*Model, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if carrierPublicID == "" || code == "" || name == "" {
		return nil, errors.New("carrier, code and name required")
	}
	id := "mdl_" + shortID()
	tag, err := s.Pool.Exec(ctx, `
INSERT INTO models (public_id, carrier_id, code, name, description, client_workflow_id, enabled, sort_order)
SELECT $1, c.id, $2, $3, $4, $5, TRUE, COALESCE((SELECT MAX(sort_order)+1 FROM models WHERE carrier_id=c.id), 1)
FROM carriers c WHERE c.public_id=$6 OR c.code=$6`,
		id, code, name, description, clientWorkflowID, carrierPublicID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, errors.New("carrier not found")
	}
	_, models, err := s.ListCarriersWithModels(ctx)
	if err != nil {
		return nil, err
	}
	for i := range models {
		if models[i].ID == id {
			return &models[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) CreateRoute(ctx context.Context, supplierPublicID, modelPublicID, workflowID, name string, peak, off float64, enabled bool, cond RouteConditions) (*SupplierRoute, error) {
	workflowID = strings.TrimSpace(workflowID)
	if supplierPublicID == "" || modelPublicID == "" || workflowID == "" {
		return nil, errors.New("supplier, model and upstream workflowId required")
	}
	if name == "" {
		name = workflowID
	}
	if peak <= 0 {
		peak = 0.03
	}
	if off <= 0 {
		off = 0.02
	}
	condRaw, _ := json.Marshal(cond)
	id := "rte_" + shortID()
	tag, err := s.Pool.Exec(ctx, `
INSERT INTO supplier_routes (
  public_id, supplier_id, model_id, workflow_id, name, enabled,
  cost_per_sec_peak, cost_per_sec_offpeak, conditions, priority
) SELECT $1, s.id, m.id, $2, $3, $4, $5, $6, $7::jsonb,
  COALESCE((SELECT MAX(priority)+1 FROM supplier_routes), 100)
FROM suppliers s, models m
WHERE (s.public_id=$8 OR s.code=$8) AND (m.public_id=$9 OR m.code=$9)`,
		id, workflowID, name, enabled, peak, off, string(condRaw), supplierPublicID, modelPublicID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, errors.New("supplier or model not found")
	}
	routes, err := s.ListRoutes(ctx)
	if err != nil {
		return nil, err
	}
	for i := range routes {
		if routes[i].ID == id {
			return &routes[i], nil
		}
	}
	return nil, ErrNotFound
}
