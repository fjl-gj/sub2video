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

func (s *Store) ListAdminCreationModes(ctx context.Context) ([]AdminCreationMode, error) {
	rows, err := s.Pool.Query(ctx, `SELECT public_id, code, name, description, status, sort_order FROM creation_modes ORDER BY sort_order, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AdminCreationMode{}
	for rows.Next() {
		var item AdminCreationMode
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Status, &item.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) UpdateCreationMode(ctx context.Context, modeID string, name, description, status *string, sortOrder *int) (*CreationMode, error) {
	mode, err := s.getCreationMode(ctx, modeID)
	if err != nil {
		return nil, err
	}
	if name != nil && strings.TrimSpace(*name) != "" {
		mode.Name = strings.TrimSpace(*name)
	}
	if description != nil {
		mode.Description = strings.TrimSpace(*description)
	}
	if status != nil {
		if *status != "published" && *status != "disabled" {
			return nil, errors.New("创作模式状态不合法")
		}
		mode.Status = *status
	}
	if sortOrder != nil {
		mode.SortOrder = *sortOrder
	}
	_, err = s.Pool.Exec(ctx, `UPDATE creation_modes SET name=$1, description=$2, status=$3, sort_order=$4, updated_at=NOW() WHERE public_id=$5`, mode.Name, mode.Description, mode.Status, mode.SortOrder, mode.ID)
	if err != nil {
		return nil, err
	}
	return mode, nil
}

func (s *Store) UpsertModeModelPolicy(ctx context.Context, modeID, modelID string, enabled bool, rank int, badge string, defaultDuration int, defaultResolution string) error {
	if rank < 1 {
		rank = 1
	}
	if defaultDuration < 1 {
		defaultDuration = 5
	}
	if defaultResolution == "" {
		defaultResolution = "768P"
	}
	tag, err := s.Pool.Exec(ctx, `
INSERT INTO mode_model_policies (mode_id, model_id, enabled, recommendation_rank, display_badge, default_duration, default_resolution)
SELECT mode.id, model.id, $3, $4, $5, $6, $7 FROM creation_modes mode, models model WHERE mode.public_id=$1 AND model.public_id=$2
ON CONFLICT (mode_id, model_id) DO UPDATE SET enabled=EXCLUDED.enabled, recommendation_rank=EXCLUDED.recommendation_rank,
  display_badge=EXCLUDED.display_badge, default_duration=EXCLUDED.default_duration, default_resolution=EXCLUDED.default_resolution, updated_at=NOW()`,
		modeID, modelID, enabled, rank, badge, defaultDuration, defaultResolution)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListModelConfigurations(ctx context.Context) ([]ModelConfiguration, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT model.public_id, model.code, model.name, carrier.public_id, carrier.name,
       mode.public_id, mode.name, policy.enabled, policy.recommendation_rank,
       policy.display_badge, policy.default_duration, policy.default_resolution
FROM mode_model_policies policy
JOIN models model ON model.id=policy.model_id
JOIN carriers carrier ON carrier.id=model.carrier_id
JOIN creation_modes mode ON mode.id=policy.mode_id
ORDER BY carrier.sort_order, model.sort_order, mode.sort_order, policy.recommendation_rank`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ModelConfiguration{}
	for rows.Next() {
		var item ModelConfiguration
		if err := rows.Scan(&item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierID, &item.CarrierName,
			&item.WorkflowID, &item.WorkflowName, &item.Enabled, &item.RecommendationRank,
			&item.DisplayBadge, &item.DefaultDuration, &item.DefaultResolution); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListSupplierOfferings(ctx context.Context) ([]SupplierOffering, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT execution.public_id, COALESCE(o.public_id,''), supplier.public_id, supplier.code, supplier.name, supplier.base_url, supplier.auth_header,
       model.public_id, model.code, model.name, carrier.name, mode.public_id, mode.code, mode.name,
       COALESCE(o.name, NULLIF(execution.workflow_code,''), NULLIF(execution.api_model_code,''), model.name),
       COALESCE(o.upstream_workflow_id, execution.workflow_code), execution.execution_type, execution.api_model_code, execution.api_version, execution.workflow_code, execution.adapter_code, execution.enabled, execution.health_status, execution.quota_status, execution.priority,
       execution.min_duration_sec, execution.max_duration_sec, execution.resolutions, execution.aspect_ratios, execution.min_images, execution.max_images,
       execution.supports_generated_audio
FROM supplier_executions execution
JOIN supplier_model_bindings binding ON binding.id=execution.binding_id
JOIN suppliers supplier ON supplier.id=binding.supplier_id
JOIN models model ON model.id=binding.model_id
JOIN carriers carrier ON carrier.id=model.carrier_id
JOIN creation_modes mode ON mode.id=execution.mode_id
LEFT JOIN supplier_offerings o ON o.id=execution.legacy_offering_id
ORDER BY supplier.code, model.code, mode.sort_order, execution.priority, execution.public_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SupplierOffering{}
	for rows.Next() {
		var item SupplierOffering
		var resRaw, aspectRaw []byte
		if err := rows.Scan(&item.ID, &item.LegacyOfferingID, &item.SupplierID, &item.SupplierCode, &item.SupplierName, &item.SupplierBaseURL, &item.SupplierAuthHeader,
			&item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName, &item.ModeID, &item.ModeCode, &item.ModeName,
			&item.Name, &item.UpstreamWorkflowID, &item.ExecutionType, &item.APIModelCode, &item.APIVersion, &item.WorkflowCode, &item.AdapterCode, &item.Enabled, &item.HealthStatus, &item.QuotaStatus, &item.Priority,
			&item.MinDurationSec, &item.MaxDurationSec, &resRaw, &aspectRaw, &item.MinImages, &item.MaxImages, &item.SupportsGeneratedAudio); err != nil {
			return nil, err
		}
		item.ExecutionID = item.ID
		_ = json.Unmarshal(resRaw, &item.Resolutions)
		_ = json.Unmarshal(aspectRaw, &item.AspectRatios)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateSupplierOffering(ctx context.Context, in SupplierOffering) (*SupplierOffering, error) {
	if in.SupplierID == "" || in.ModelID == "" || in.ModeID == "" {
		return nil, errors.New("供应商、模型和创作模式均为必填")
	}
	if in.ExecutionType == "" {
		in.ExecutionType = "workflow"
	}
	if in.ExecutionType != "api" && in.ExecutionType != "workflow" {
		return nil, errors.New("执行类型不合法")
	}
	if in.ExecutionType == "workflow" && strings.TrimSpace(in.WorkflowCode) == "" {
		in.WorkflowCode = strings.TrimSpace(in.UpstreamWorkflowID)
	}
	if in.ExecutionType == "workflow" && in.WorkflowCode == "" {
		return nil, errors.New("工作流编码为必填")
	}
	if in.ExecutionType == "api" && strings.TrimSpace(in.APIModelCode) == "" {
		return nil, errors.New("API 模型编码为必填")
	}
	if in.AdapterCode == "" {
		in.AdapterCode = "autodl_h3_v1"
	}
	if !supportedProviderAdapter(in.AdapterCode) {
		return nil, fmt.Errorf("当前部署未启用供给适配器: %s", in.AdapterCode)
	}
	if in.MinDurationSec < 1 {
		in.MinDurationSec = 1
	}
	if in.MaxDurationSec < in.MinDurationSec {
		return nil, errors.New("最大时长不能小于最小时长")
	}
	resRaw, _ := json.Marshal(in.Resolutions)
	aspectRaw, _ := json.Marshal(in.AspectRatios)
	in.ID = NewOfferingID()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
INSERT INTO supplier_offerings (public_id, supplier_id, model_id, mode_id, name, upstream_workflow_id, adapter_code, enabled,
 health_status, priority, min_duration_sec, max_duration_sec, resolutions, aspect_ratios, min_images, max_images, supports_generated_audio,
 execution_type, api_model_code, api_version, workflow_code)
SELECT $1, supplier.id, model.id, mode.id, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14::jsonb, $15, $16, $17, $18, $19, $20, $21
FROM suppliers supplier, models model, creation_modes mode
WHERE supplier.public_id=$2 AND model.public_id=$3 AND mode.public_id=$4`,
		in.ID, in.SupplierID, in.ModelID, in.ModeID, strings.TrimSpace(in.Name), strings.TrimSpace(in.UpstreamWorkflowID), in.AdapterCode,
		in.Enabled, in.HealthStatus, in.Priority, in.MinDurationSec, in.MaxDurationSec, string(resRaw), string(aspectRaw), in.MinImages, in.MaxImages, in.SupportsGeneratedAudio,
		in.ExecutionType, in.APIModelCode, in.APIVersion, in.WorkflowCode)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrNotFound
	}
	// New execution tables are the source for direct routing. Keep a legacy
	// offering row for existing worker/task references during migration.
	_, err = tx.Exec(ctx, `
INSERT INTO supplier_model_bindings (public_id, supplier_id, model_id)
SELECT 'bind_' || md5(supplier.id::text || ':' || model.id::text), supplier.id, model.id
FROM suppliers supplier, models model WHERE supplier.public_id=$1 AND model.public_id=$2
ON CONFLICT (supplier_id, model_id) DO NOTHING`, in.SupplierID, in.ModelID)
	if err != nil {
		return nil, err
	}
	tag, err = tx.Exec(ctx, `
INSERT INTO supplier_executions (public_id, binding_id, mode_id, execution_type, api_model_code, api_version, workflow_code,
 adapter_code, enabled, health_status, quota_status, priority, min_duration_sec, max_duration_sec, resolutions, aspect_ratios,
 min_images, max_images, supports_generated_audio, legacy_offering_id)
SELECT 'exec_' || $1, b.id, mode.id, $5, $6, $7, $8, $9, $10, $11, 'available', $12, $13, $14, $15::jsonb, $16::jsonb,
 $17, $18, $19, offering.id
FROM supplier_offerings offering
JOIN supplier_model_bindings b ON b.supplier_id=offering.supplier_id AND b.model_id=offering.model_id
JOIN creation_modes mode ON mode.public_id=$4
WHERE offering.public_id=$1
  AND b.supplier_id=(SELECT id FROM suppliers WHERE public_id=$2)
  AND b.model_id=(SELECT id FROM models WHERE public_id=$3)
ON CONFLICT (legacy_offering_id) DO UPDATE SET execution_type=EXCLUDED.execution_type, api_model_code=EXCLUDED.api_model_code,
 api_version=EXCLUDED.api_version, workflow_code=EXCLUDED.workflow_code, adapter_code=EXCLUDED.adapter_code, enabled=EXCLUDED.enabled,
 health_status=EXCLUDED.health_status, priority=EXCLUDED.priority, min_duration_sec=EXCLUDED.min_duration_sec, max_duration_sec=EXCLUDED.max_duration_sec,
 resolutions=EXCLUDED.resolutions, aspect_ratios=EXCLUDED.aspect_ratios, min_images=EXCLUDED.min_images, max_images=EXCLUDED.max_images,
 supports_generated_audio=EXCLUDED.supports_generated_audio, updated_at=NOW()`,
		in.ID, in.SupplierID, in.ModelID, in.ModeID, in.ExecutionType, in.APIModelCode, in.APIVersion, in.WorkflowCode, in.AdapterCode,
		in.Enabled, in.HealthStatus, in.Priority, in.MinDurationSec, in.MaxDurationSec, string(resRaw), string(aspectRaw), in.MinImages, in.MaxImages, in.SupportsGeneratedAudio)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrNotFound
	}
	created, err := getSupplierOfferingExecution(ctx, tx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Store) UpdateSupplierOffering(ctx context.Context, offeringID string, in SupplierOffering) (*SupplierOffering, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	current, err := getSupplierOfferingExecution(ctx, tx, offeringID)
	if err != nil {
		return nil, err
	}
	if in.Name != "" {
		current.Name = in.Name
	}
	if in.UpstreamWorkflowID != "" {
		current.UpstreamWorkflowID = in.UpstreamWorkflowID
	}
	if in.ExecutionType != "" {
		current.ExecutionType = in.ExecutionType
	}
	if in.APIModelCode != "" {
		current.APIModelCode = in.APIModelCode
	}
	if in.APIVersion != "" {
		current.APIVersion = in.APIVersion
	}
	if in.WorkflowCode != "" {
		current.WorkflowCode = in.WorkflowCode
		current.UpstreamWorkflowID = in.WorkflowCode
	}
	if in.AdapterCode != "" {
		current.AdapterCode = in.AdapterCode
	}
	if !supportedProviderAdapter(current.AdapterCode) {
		return nil, fmt.Errorf("当前部署未启用供给适配器: %s", current.AdapterCode)
	}
	if in.HealthStatus != "" {
		current.HealthStatus = in.HealthStatus
	}
	if in.MaxDurationSec > 0 {
		current.MaxDurationSec = in.MaxDurationSec
	}
	if in.MinDurationSec > 0 {
		current.MinDurationSec = in.MinDurationSec
	}
	if current.MaxDurationSec < current.MinDurationSec {
		return nil, errors.New("最大时长不能小于最小时长")
	}
	if in.Resolutions != nil {
		current.Resolutions = in.Resolutions
	}
	if in.AspectRatios != nil {
		current.AspectRatios = in.AspectRatios
	}
	current.Enabled, current.Priority, current.MinImages, current.MaxImages, current.SupportsGeneratedAudio = in.Enabled, in.Priority, in.MinImages, in.MaxImages, in.SupportsGeneratedAudio
	resRaw, _ := json.Marshal(current.Resolutions)
	aspectRaw, _ := json.Marshal(current.AspectRatios)
	legacyID := current.LegacyOfferingID
	if legacyID == "" {
		tag, err := tx.Exec(ctx, `UPDATE supplier_executions SET execution_type=$1, api_model_code=$2, api_version=$3, workflow_code=$4, adapter_code=$5, enabled=$6,
	health_status=$7, priority=$8, min_duration_sec=$9, max_duration_sec=$10, resolutions=$11::jsonb, aspect_ratios=$12::jsonb,
	min_images=$13, max_images=$14, supports_generated_audio=$15, updated_at=NOW() WHERE public_id=$16`,
			current.ExecutionType, current.APIModelCode, current.APIVersion, current.WorkflowCode, current.AdapterCode, current.Enabled, current.HealthStatus, current.Priority,
			current.MinDurationSec, current.MaxDurationSec, string(resRaw), string(aspectRaw), current.MinImages, current.MaxImages, current.SupportsGeneratedAudio, current.ID)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != 1 {
			return nil, ErrNotFound
		}
	} else {
		tag, err := tx.Exec(ctx, `UPDATE supplier_offerings SET name=$1, upstream_workflow_id=$2, execution_type=$3, api_model_code=$4, api_version=$5, workflow_code=$6, adapter_code=$7, enabled=$8,
health_status=$9, priority=$10, min_duration_sec=$11, max_duration_sec=$12, resolutions=$13::jsonb, aspect_ratios=$14::jsonb,
min_images=$15, max_images=$16, supports_generated_audio=$17, updated_at=NOW() WHERE public_id=$18`,
			current.Name, current.UpstreamWorkflowID, current.ExecutionType, current.APIModelCode, current.APIVersion, current.WorkflowCode, current.AdapterCode, current.Enabled, current.HealthStatus, current.Priority,
			current.MinDurationSec, current.MaxDurationSec, string(resRaw), string(aspectRaw), current.MinImages, current.MaxImages, current.SupportsGeneratedAudio, legacyID)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != 1 {
			return nil, ErrNotFound
		}
		tag, err = tx.Exec(ctx, `UPDATE supplier_executions e SET execution_type=o.execution_type, api_model_code=o.api_model_code, api_version=o.api_version,
workflow_code=o.workflow_code, adapter_code=o.adapter_code, enabled=o.enabled, health_status=o.health_status, priority=o.priority,
min_duration_sec=o.min_duration_sec, max_duration_sec=o.max_duration_sec, resolutions=o.resolutions, aspect_ratios=o.aspect_ratios,
min_images=o.min_images, max_images=o.max_images, supports_generated_audio=o.supports_generated_audio, updated_at=NOW()
FROM supplier_offerings o WHERE e.legacy_offering_id=o.id AND o.public_id=$1`, legacyID)
		if err != nil {
			return nil, err
		}
		// An old-only offering is still editable, but a linked execution must
		// never disappear silently partway through a successful update.
		if current.ExecutionID != "" && tag.RowsAffected() != 1 {
			return nil, ErrNotFound
		}
	}
	updated, err := getSupplierOfferingExecution(ctx, tx, current.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Store) ListSupplierCostRules(ctx context.Context) ([]SupplierCostRule, error) {
	rows, err := s.Pool.Query(ctx, `SELECT cost.public_id, COALESCE(offering.public_id,''), COALESCE(execution.public_id,''), cost.resolution, cost.generate_audio,
cost.price_per_sec_peak::float8, cost.price_per_sec_offpeak::float8, cost.currency, cost.version, cost.status
FROM supplier_cost_rules cost
LEFT JOIN supplier_offerings offering ON offering.id=cost.offering_id
LEFT JOIN supplier_executions execution ON execution.id=cost.execution_id
ORDER BY COALESCE(execution.public_id, offering.public_id), cost.resolution, cost.generate_audio`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SupplierCostRule{}
	for rows.Next() {
		var item SupplierCostRule
		if err := rows.Scan(&item.ID, &item.OfferingID, &item.ExecutionID, &item.Resolution, &item.GenerateAudio, &item.PeakPerSecond, &item.OffpeakPerSecond, &item.Currency, &item.Version, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertSupplierCostRule(ctx context.Context, in SupplierCostRule) (*SupplierCostRule, error) {
	if in.ExecutionID == "" && in.OfferingID == "" || in.Resolution == "" || in.PeakPerSecond < 0 || in.OffpeakPerSecond < 0 {
		return nil, errors.New("成本规则参数不合法")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.Currency == "" {
		in.Currency = "CNY"
	}
	if in.ID == "" {
		in.ID = NewPriceID()
	}
	in.Version = fmt.Sprintf("v%d", time.Now().UnixNano())
	var executionInternalID, offeringInternalID int64
	if in.ExecutionID != "" {
		if err := s.Pool.QueryRow(ctx, `SELECT execution.id, COALESCE(execution.legacy_offering_id,0) FROM supplier_executions execution WHERE execution.public_id=$1`, in.ExecutionID).Scan(&executionInternalID, &offeringInternalID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		if offeringInternalID > 0 {
			_ = s.Pool.QueryRow(ctx, `SELECT public_id FROM supplier_offerings WHERE id=$1`, offeringInternalID).Scan(&in.OfferingID)
		}
	} else {
		if err := s.Pool.QueryRow(ctx, `SELECT offering.id, COALESCE(execution.id,0), COALESCE(execution.public_id,'') FROM supplier_offerings offering LEFT JOIN supplier_executions execution ON execution.legacy_offering_id=offering.id WHERE offering.public_id=$1`, in.OfferingID).Scan(&offeringInternalID, &executionInternalID, &in.ExecutionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}
	var existingID string
	if executionInternalID > 0 {
		err := s.Pool.QueryRow(ctx, `SELECT public_id FROM supplier_cost_rules WHERE execution_id=$1 AND resolution=$2 AND generate_audio=$3`, executionInternalID, in.Resolution, in.GenerateAudio).Scan(&existingID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	} else {
		err := s.Pool.QueryRow(ctx, `SELECT public_id FROM supplier_cost_rules WHERE offering_id=$1 AND resolution=$2 AND generate_audio=$3`, offeringInternalID, in.Resolution, in.GenerateAudio).Scan(&existingID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	var err error
	if existingID != "" {
		_, err = s.Pool.Exec(ctx, `UPDATE supplier_cost_rules SET price_per_sec_peak=$1, price_per_sec_offpeak=$2, execution_id=NULLIF($3,0), currency=$4, version=$5, status=$6, updated_at=NOW() WHERE public_id=$7`, in.PeakPerSecond, in.OffpeakPerSecond, executionInternalID, in.Currency, in.Version, in.Status, existingID)
		in.ID = existingID
	} else {
		_, err = s.Pool.Exec(ctx, `INSERT INTO supplier_cost_rules (public_id, offering_id, execution_id, resolution, generate_audio, price_per_sec_peak, price_per_sec_offpeak, currency, version, status) VALUES ($1, NULLIF($2,0), NULLIF($3,0), $4, $5, $6, $7, $8, $9, $10)`, in.ID, offeringInternalID, executionInternalID, in.Resolution, in.GenerateAudio, in.PeakPerSecond, in.OffpeakPerSecond, in.Currency, in.Version, in.Status)
	}
	if err != nil {
		return nil, err
	}
	items, err := s.ListSupplierCostRules(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if ((in.ExecutionID != "" && items[i].ExecutionID == in.ExecutionID) || (in.ExecutionID == "" && items[i].OfferingID == in.OfferingID)) && items[i].Resolution == in.Resolution && items[i].GenerateAudio == in.GenerateAudio {
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) ListRetailPriceRules(ctx context.Context) ([]RetailPriceRule, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT retail.public_id, mode.public_id, mode.code, mode.name,
COALESCE(model.public_id, ''), COALESCE(model.code, ''), COALESCE(model.name, ''), COALESCE(carrier.name, ''),
retail.resolution, retail.generate_audio, retail.name, retail.price_per_sec::float8, retail.version, retail.status
FROM retail_price_rules retail
JOIN creation_modes mode ON mode.id=retail.mode_id
LEFT JOIN models model ON model.id=retail.model_id
LEFT JOIN carriers carrier ON carrier.id=model.carrier_id
ORDER BY mode.sort_order, COALESCE(carrier.name,''), COALESCE(model.name,''), retail.resolution, retail.generate_audio`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RetailPriceRule{}
	for rows.Next() {
		var item RetailPriceRule
		if err := rows.Scan(&item.ID, &item.ModeID, &item.ModeCode, &item.ModeName, &item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName,
			&item.Resolution, &item.GenerateAudio, &item.Name, &item.PricePerSecond, &item.Version, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertRetailPriceRule(ctx context.Context, in RetailPriceRule) (*RetailPriceRule, error) {
	if in.ModeID == "" || in.Resolution == "" || in.PricePerSecond < 0 {
		return nil, errors.New("售价规则参数不合法")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	// New pricing is mode-level. Ignore any legacy model selector sent by old
	// admin clients so the direct route cannot accidentally use model pricing.
	in.ModelID, in.ModelCode, in.ModelName, in.CarrierName = "", "", "", ""
	in.Version = fmt.Sprintf("v%d", time.Now().UnixNano())
	var existingID string
	err := s.Pool.QueryRow(ctx, `
SELECT retail.public_id FROM retail_price_rules retail
JOIN creation_modes mode ON mode.id=retail.mode_id
	WHERE mode.public_id=$1 AND retail.model_id IS NULL AND retail.resolution=$2 AND retail.generate_audio=$3
ORDER BY retail.updated_at DESC LIMIT 1`, in.ModeID, in.Resolution, in.GenerateAudio).Scan(&existingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if existingID != "" {
		_, err = s.Pool.Exec(ctx, `UPDATE retail_price_rules SET name=$1, price_per_sec=$2, version=$3, status=$4, updated_at=NOW() WHERE public_id=$5`, in.Name, in.PricePerSecond, in.Version, in.Status, existingID)
	} else {
		in.ID = NewPriceID()
		_, err = s.Pool.Exec(ctx, `
		INSERT INTO retail_price_rules (public_id, mode_id, model_id, resolution, generate_audio, name, price_per_sec, version, status)
SELECT $1, mode.id, NULL::bigint, $3, $4, $5, $6, $7, $8 FROM creation_modes mode
WHERE mode.public_id=$2`,
			in.ID, in.ModeID, in.Resolution, in.GenerateAudio, in.Name, in.PricePerSecond, in.Version, in.Status)
	}
	if err != nil {
		return nil, err
	}
	items, err := s.ListRetailPriceRules(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ModeID == in.ModeID && items[i].ModelID == in.ModelID && items[i].Resolution == in.Resolution && items[i].GenerateAudio == in.GenerateAudio && items[i].Version == in.Version {
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}
