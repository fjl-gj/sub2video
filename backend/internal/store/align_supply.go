package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// AlignSupplyChainH3 idempotently reshapes UAT/prod seeds to:
// Carrier minimax → Model h3 → 3 platform workflows → AutoDL routes from 供应商autodl.md
func (s *Store) AlignSupplyChainH3(ctx context.Context) error {
	// Carrier MINMAX
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO carriers (public_id, code, name, description, enabled, sort_order)
VALUES ('car_minimax', 'minimax', 'MINMAX', 'MiniMax 视频生成承运商', TRUE, 1)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  enabled = TRUE,
  updated_at = NOW()`); err != nil {
		return fmt.Errorf("carrier: %w", err)
	}

	// Model H3 under minimax
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO models (public_id, carrier_id, code, name, description, client_workflow_id, enabled, sort_order)
SELECT 'mdl_h3', c.id, 'h3', 'H3', 'MiniMax H3 视频模型', '', TRUE, 1
FROM carriers c WHERE c.code = 'minimax'
ON CONFLICT (carrier_id, code) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  client_workflow_id = '',
  enabled = TRUE,
  updated_at = NOW()`); err != nil {
		return fmt.Errorf("model: %w", err)
	}

	// Disable legacy modality models (text / image-keyframes / image-multi)
	if _, err := s.Pool.Exec(ctx, `
UPDATE models SET enabled=FALSE, updated_at=NOW()
WHERE code IN ('text','image-keyframes','image-multi')`); err != nil {
		return err
	}

	type wfSeed struct {
		code, name, desc, status, badge string
		order                           int
		params                          WorkflowParams
	}
	wfs := []wfSeed{
		{
			code: "wf-h3-text", name: "文生视频", desc: "纯文字描述生成视频",
			status: "published", order: 1,
			params: WorkflowParams{
				AspectRatios: []string{"16:9", "9:16", "1:1"},
				Durations:    []int{5, 10, 15}, DefaultDuration: 5,
				Resolutions: []string{"768P"}, DefaultResolution: "768P",
			},
		},
		{
			code: "wf-h3-image", name: "首尾帧图生视频", desc: "首帧 / 尾帧图片 + 提示词",
			status: "published", order: 2,
			params: WorkflowParams{
				AspectRatios: []string{"16:9", "9:16", "1:1"},
				Durations:    []int{5, 10, 15}, DefaultDuration: 5,
				Resolutions: []string{"768P"}, DefaultResolution: "768P",
			},
		},
		{
			code: "wf-h3-multi", name: "多图参考生视频", desc: "多张参考图 + 提示词（最长 12 秒）",
			status: "published", order: 3,
			params: WorkflowParams{
				AspectRatios: []string{"16:9", "9:16", "1:1"},
				Durations:    []int{5, 10, 12}, DefaultDuration: 5,
				Resolutions: []string{"768P"}, DefaultResolution: "768P",
			},
		},
	}
	for _, w := range wfs {
		raw, _ := json.Marshal(w.params)
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO platform_workflows (public_id, code, name, description, model_id, status, badge, sort_order, params)
SELECT $1, $2, $3, $4, m.id, $5, $6, $7, $8::jsonb
FROM models m
JOIN carriers c ON c.id = m.carrier_id
WHERE m.code = 'h3' AND c.code = 'minimax'
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  model_id = EXCLUDED.model_id,
  status = EXCLUDED.status,
  badge = EXCLUDED.badge,
  sort_order = EXCLUDED.sort_order,
  params = EXCLUDED.params,
  updated_at = NOW()`,
			w.code, w.code, w.name, w.desc, w.status, w.badge, w.order, string(raw)); err != nil {
			return fmt.Errorf("workflow %s: %w", w.code, err)
		}
	}

	// Pricing: keep 768P active for all three; deactivate orphan 1080P if no route
	prices := []struct {
		wf, res, name string
		price         float64
	}{
		{"wf-h3-text", "768P", "H3 文生 / 768P", 0.50},
		{"wf-h3-image", "768P", "H3 首尾帧 / 768P", 0.50},
		{"wf-h3-multi", "768P", "H3 多图 / 768P", 0.50},
	}
	for _, p := range prices {
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO pricing_rules (public_id, workflow_id, name, resolution, price_per_sec, cost_hint_per_sec, version, status)
SELECT $1, w.id, $2, $3, $4, 0.03, 'v2.0', 'active'
FROM platform_workflows w WHERE w.code = $5
ON CONFLICT (workflow_id, resolution, version) DO UPDATE SET
  name = EXCLUDED.name,
  price_per_sec = EXCLUDED.price_per_sec,
  cost_hint_per_sec = EXCLUDED.cost_hint_per_sec,
  status = 'active',
  updated_at = NOW()`,
			"pr_"+shortID(), p.name, p.res, p.price, p.wf); err != nil {
			return fmt.Errorf("price %s: %w", p.wf, err)
		}
		// Activate any existing 768P rules for this workflow
		if _, err := s.Pool.Exec(ctx, `
UPDATE pricing_rules r SET status='active', updated_at=NOW()
FROM platform_workflows w
WHERE r.workflow_id = w.id AND w.code = $1 AND r.resolution = '768P'`, p.wf); err != nil {
			return err
		}
	}

	// Supplier autodl
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO suppliers (public_id, code, name, base_url, auth_header, token_encrypted, enabled, status)
VALUES ('sup_autodl', 'autodl', 'AutoDL', 'https://autodl.art', 'Authorization', '', TRUE, 'unknown')
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  base_url = EXCLUDED.base_url,
  enabled = TRUE,
  updated_at = NOW()`); err != nil {
		return fmt.Errorf("supplier: %w", err)
	}

	type routeSeed struct {
		workflowID, clientWF, name string
		peak, off                  float64
		cond                       RouteConditions
		priority                   int
	}
	// From D:\download\download\供应商autodl.md — H3 三工作流
	routes := []routeSeed{
		{
			workflowID: "minimax_h3_b99_001", clientWF: "wf-h3-text", name: "H3文生视频(b99)",
			peak: 0.03, off: 0.02, priority: 10,
			cond: RouteConditions{MaxDuration: 15, Resolutions: []string{"768P"}, RequireSource: []string{"text"}, AspectRatios: []string{"16:9", "9:16", "1:1"}},
		},
		{
			workflowID: "minimax_h3_b99_002", clientWF: "wf-h3-image", name: "H3首尾帧(b99)",
			peak: 0.03, off: 0.02, priority: 10,
			cond: RouteConditions{MaxDuration: 15, Resolutions: []string{"768P"}, RequireSource: []string{"upload", "asset"}, AspectRatios: []string{"16:9", "9:16", "1:1"}},
		},
		{
			workflowID: "minimax_h3_b99_003_12s", clientWF: "wf-h3-multi", name: "H3多图参考12s(b99)",
			peak: 0.03, off: 0.02, priority: 10,
			cond: RouteConditions{MaxDuration: 12, Resolutions: []string{"768P"}, RequireSource: []string{"upload", "asset"}, AspectRatios: []string{"16:9", "9:16", "1:1"}},
		},
	}

	for _, r := range routes {
		condRaw, _ := json.Marshal(r.cond)
		var existing string
		err := s.Pool.QueryRow(ctx, `
SELECT r.public_id FROM supplier_routes r
JOIN suppliers s ON s.id = r.supplier_id
WHERE s.code = 'autodl' AND r.workflow_id = $1
LIMIT 1`, r.workflowID).Scan(&existing)
		if err == nil && existing != "" {
			if _, err := s.Pool.Exec(ctx, `
UPDATE supplier_routes r SET
  model_id = m.id,
  name = $1,
  enabled = TRUE,
  cost_per_sec_peak = $2,
  cost_per_sec_offpeak = $3,
  conditions = $4::jsonb,
  client_workflow_id = $5,
  priority = $6,
  updated_at = NOW()
FROM models m
JOIN carriers c ON c.id = m.carrier_id
WHERE r.public_id = $7 AND m.code = 'h3' AND c.code = 'minimax'`,
				r.name, r.peak, r.off, string(condRaw), r.clientWF, r.priority, existing); err != nil {
				return fmt.Errorf("update route %s: %w", r.workflowID, err)
			}
			continue
		}
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO supplier_routes (
  public_id, supplier_id, model_id, workflow_id, name, enabled,
  cost_per_sec_peak, cost_per_sec_offpeak, conditions, priority, client_workflow_id
) SELECT $1, s.id, m.id, $2, $3, TRUE, $4, $5, $6::jsonb, $7, $8
FROM suppliers s, models m
JOIN carriers c ON c.id = m.carrier_id
WHERE s.code = 'autodl' AND m.code = 'h3' AND c.code = 'minimax'`,
			"rte_"+shortID(), r.workflowID, r.name, r.peak, r.off, string(condRaw), r.priority, r.clientWF); err != nil {
			return fmt.Errorf("insert route %s: %w", r.workflowID, err)
		}
	}

	// Disable obsolete AutoDL H3 routes not in the allowlist
	if _, err := s.Pool.Exec(ctx, `
UPDATE supplier_routes r SET enabled=FALSE, updated_at=NOW()
FROM suppliers s
WHERE r.supplier_id = s.id AND s.code = 'autodl'
  AND r.workflow_id LIKE 'minimax_h3%'
  AND r.workflow_id NOT IN ('minimax_h3_b99_001','minimax_h3_b99_002','minimax_h3_b99_003_12s')`); err != nil {
		return err
	}

	return nil
}
