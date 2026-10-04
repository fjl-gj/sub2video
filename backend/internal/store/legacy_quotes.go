package store

// Historical quote creation/consumption is isolated here for compatibility
// maintenance. The public quote endpoint returns 410; current clients must use
// estimate followed by direct, idempotent submission. Do not call these
// functions from the formal creation path or remove historical tables as part
// of this source-only separation.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const quoteTTL = 5 * time.Minute

type QuoteInput struct {
	ModeID          string
	ModelID         string
	DurationSeconds int
	Resolution      string
	AspectRatio     string
	GenerateAudio   bool
	ImageCount      int
}

type TaskQuote struct {
	ID               string
	ExpiresAt        time.Time
	ModeID           string
	ModeCode         string
	ModeName         string
	ModelID          string
	ModelCode        string
	ModelName        string
	CarrierID        string
	CarrierCode      string
	CarrierName      string
	OfferingID       string
	OfferingName     string
	SupplierCode     string
	SupplierBaseURL  string
	SupplierAuth     string
	UpstreamWorkflow string
	AdapterCode      string
	CostRuleID       string
	RetailRuleID     string
	DurationSeconds  int
	Resolution       string
	AspectRatio      string
	GenerateAudio    bool
	PricePerSecond   float64
	CostPerSecond    float64
	Total            float64
	MinImages        int
	MaxImages        int
	RetailSnapshot   map[string]any
	CostSnapshot     map[string]any
	OfferingSnapshot map[string]any
}

func (s *Store) CreateQuote(ctx context.Context, userID int64, in QuoteInput) (*TaskQuote, error) {
	if in.DurationSeconds <= 0 || in.DurationSeconds > 60 {
		return nil, errors.New("视频时长不合法")
	}
	if in.Resolution == "" || in.AspectRatio == "" {
		return nil, errors.New("请完整选择生成设置")
	}
	mode, err := s.getCreationMode(ctx, in.ModeID)
	if err != nil || mode.Status != "published" {
		return nil, errors.New("创作模式未开放")
	}
	modelIDs := []string{in.ModelID}
	if in.ModelID == "" {
		modelIDs, err = s.recommendedModelIDs(ctx, mode.ID)
		if err != nil {
			return nil, err
		}
	}
	for _, modelID := range modelIDs {
		quote, err := s.createQuoteForModel(ctx, userID, mode, modelID, in)
		if err == nil {
			return quote, nil
		}
		if in.ModelID != "" {
			return nil, err
		}
	}
	return nil, errors.New("当前参数暂无可用模型或供给")
}

func (s *Store) createQuoteForModel(ctx context.Context, userID int64, mode *CreationMode, modelID string, in QuoteInput) (*TaskQuote, error) {
	model, err := s.getCatalogModel(ctx, modelID)
	if err != nil {
		return nil, err
	}
	supported, err := s.modelSupportsMode(ctx, mode.ID, model.ID)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, errors.New("该模型不支持当前创作模式")
	}
	retail, err := s.findRetailPrice(ctx, mode.ID, model.ID, in.Resolution, in.GenerateAudio)
	if err != nil {
		return nil, err
	}
	offerings, err := s.listMatchingOfferings(ctx, mode.ID, model.ID)
	if err != nil {
		return nil, err
	}
	var selected *SupplierOffering
	var cost float64
	var costRuleID, costVersion string
	for i := range offerings {
		offering := &offerings[i]
		if !offeringMatches(*offering, DirectRouteRequest{DurationSeconds: in.DurationSeconds, Resolution: in.Resolution, AspectRatio: in.AspectRatio, GenerateAudio: in.GenerateAudio, ImageCount: in.ImageCount}) {
			continue
		}
		unitCost, ruleID, version, err := s.findSupplierCost(ctx, offering.ID, in.Resolution, in.GenerateAudio)
		if err != nil {
			continue
		}
		if selected == nil || unitCost < cost || (unitCost == cost && offering.Priority < selected.Priority) || (unitCost == cost && offering.Priority == selected.Priority && offering.ID < selected.ID) {
			selected, cost, costRuleID, costVersion = offering, unitCost, ruleID, version
		}
	}
	if selected == nil {
		return nil, errors.New("当前参数暂无可用供给")
	}
	quote := &TaskQuote{
		ID:     NewQuoteID(),
		ModeID: mode.ID, ModeCode: mode.Code, ModeName: mode.Name,
		ModelID: model.ID, ModelCode: model.Code, ModelName: model.Name,
		CarrierID: model.CarrierID, CarrierCode: model.CarrierCode, CarrierName: model.CarrierName,
		OfferingID: selected.ID, OfferingName: selected.Name, SupplierCode: selected.SupplierCode,
		SupplierBaseURL: selected.SupplierBaseURL, SupplierAuth: selected.SupplierAuthHeader,
		UpstreamWorkflow: selected.UpstreamWorkflowID, AdapterCode: selected.AdapterCode,
		CostRuleID: costRuleID, RetailRuleID: retail.ID,
		DurationSeconds: in.DurationSeconds, Resolution: in.Resolution, AspectRatio: in.AspectRatio,
		GenerateAudio: in.GenerateAudio, PricePerSecond: retail.PricePerSecond, CostPerSecond: cost,
		Total: retail.PricePerSecond * float64(in.DurationSeconds), MinImages: selected.MinImages, MaxImages: selected.MaxImages,
	}
	quote.RetailSnapshot = map[string]any{"ruleId": retail.ID, "version": retail.Version, "pricePerSecond": quote.PricePerSecond, "total": quote.Total, "currency": "CNY"}
	quote.CostSnapshot = map[string]any{"ruleId": costRuleID, "version": costVersion, "costPerSecond": cost, "estimatedTotal": cost * float64(in.DurationSeconds), "currency": "CNY"}
	quote.OfferingSnapshot = map[string]any{"offeringId": selected.ID, "supplierCode": selected.SupplierCode, "upstreamWorkflowId": selected.UpstreamWorkflowID, "adapterCode": selected.AdapterCode, "maxImages": selected.MaxImages}
	request := map[string]any{"modeId": mode.ID, "modeCode": mode.Code, "modelId": model.ID, "durationSeconds": in.DurationSeconds, "resolution": in.Resolution, "aspectRatio": in.AspectRatio, "generateAudio": in.GenerateAudio}
	requestRaw, _ := json.Marshal(request)
	retailRaw, _ := json.Marshal(quote.RetailSnapshot)
	costRaw, _ := json.Marshal(quote.CostSnapshot)
	offeringRaw, _ := json.Marshal(quote.OfferingSnapshot)
	err = s.Pool.QueryRow(ctx, `
INSERT INTO task_quotes (public_id, user_id, mode_id, model_id, offering_id, retail_rule_id, cost_rule_id, request_snapshot, retail_snapshot, cost_snapshot, offering_snapshot, expires_at)
SELECT $1, $2, mode.id, model.id, offering.id, retail.id, cost.id, $3::jsonb, $4::jsonb, $5::jsonb, $6::jsonb, NOW() + $7 * INTERVAL '1 second'
FROM creation_modes mode, models model, supplier_offerings offering, retail_price_rules retail, supplier_cost_rules cost
WHERE mode.public_id=$8 AND model.public_id=$9 AND offering.public_id=$10 AND retail.public_id=$11 AND cost.public_id=$12
RETURNING expires_at`,
		quote.ID, userID, string(requestRaw), string(retailRaw), string(costRaw), string(offeringRaw), int64(quoteTTL/time.Second),
		mode.ID, model.ID, selected.ID, retail.ID, costRuleID).Scan(&quote.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return quote, nil
}

func (s *Store) CreateTaskFromQuote(ctx context.Context, userID int64, quoteID, title, prompt string, sourceMeta SourceMeta) (*Task, string, float64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, "", 0, err
	}
	defer tx.Rollback(ctx)
	quote, err := s.getQuoteForConsume(ctx, tx, userID, quoteID)
	if err != nil {
		return nil, "", 0, err
	}
	constraints := creationSourceConstraints{ModeCode: quote.ModeCode, MinImages: quote.MinImages, MaxImages: quote.MaxImages}
	if err := validateModeInput(ctx, s, userID, constraints, prompt, &sourceMeta); err != nil {
		return nil, "", 0, err
	}
	var balance float64
	if err := tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id=$1 FOR UPDATE`, userID).Scan(&balance); err != nil {
		return nil, "", 0, err
	}
	if balance < quote.Total {
		return nil, "", 0, errors.New("insufficient balance")
	}
	if title = strings.TrimSpace(title); title == "" {
		title = strings.TrimSpace(prompt)
		if runes := []rune(title); len(runes) > 20 {
			title = string(runes[:20])
		}
		if title == "" {
			title = "未命名创作"
		}
	}
	sourceType, sourceRef := sourceFromMeta(quote.ModeCode, sourceMeta)
	metaRaw, _ := json.Marshal(sourceMeta)
	inputRaw, _ := json.Marshal(map[string]any{"prompt": prompt, "sourceMeta": sourceMeta})
	retailRaw, _ := json.Marshal(quote.RetailSnapshot)
	costRaw, _ := json.Marshal(quote.CostSnapshot)
	offeringRaw, _ := json.Marshal(quote.OfferingSnapshot)
	taskID := NewTaskID()
	var internalID int64
	err = tx.QueryRow(ctx, `
INSERT INTO generation_tasks (
  public_id, user_id, title, workflow, status, progress, duration_sec, resolution, cost, thumb_url,
  source_type, source_ref, source_meta, prompt, aspect_ratio, workflow_public_id, price_rule_id, price_version,
  price_per_sec, cost_per_sec, price_snapshot, route_snapshot, carrier_code, model_code, supplier_code,
  supplier_route_id, upstream_workflow_id, mode_public_id, model_public_id, generate_audio, quote_public_id,
  retail_snapshot, cost_snapshot, offering_snapshot, input_snapshot
) VALUES (
  $1,$2,$3,$4,'queued',0,$5,$6,$7,'',$8,$9,$10::jsonb,$11,$12,$13,$14,$15,$16,$17,$18::jsonb,$19::jsonb,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30::jsonb,$31::jsonb,$32::jsonb
) RETURNING id`,
		taskID, userID, title, quote.ModeName, quote.DurationSeconds, quote.Resolution, quote.Total,
		sourceType, sourceRef, string(metaRaw), prompt, quote.AspectRatio, quote.ModeCode,
		quote.RetailRuleID, quote.RetailSnapshot["version"], quote.PricePerSecond, quote.CostPerSecond,
		string(retailRaw), string(offeringRaw), quote.CarrierCode, quote.ModelCode, quote.SupplierCode,
		quote.OfferingID, quote.UpstreamWorkflow, quote.ModeID, quote.ModelID, quote.GenerateAudio, quote.ID,
		string(retailRaw), string(costRaw), string(offeringRaw), string(inputRaw),
	).Scan(&internalID)
	if err != nil {
		return nil, "", 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance=balance-$1, version=version+1, updated_at=NOW() WHERE user_id=$2`, quote.Total, userID); err != nil {
		return nil, "", 0, err
	}
	if err = tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id=$1`, userID).Scan(&balance); err != nil {
		return nil, "", 0, err
	}
	ledgerID := NewLedgerID()
	if _, err = tx.Exec(ctx, `INSERT INTO wallet_ledger (public_id, user_id, type, amount, balance_after, ref_type, ref_id, description) VALUES ($1,$2,'task_charge',$3,$4,'task',$5,$6)`, ledgerID, userID, -quote.Total, balance, taskID, fmt.Sprintf("%s %s", taskID, title)); err != nil {
		return nil, "", 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE generation_tasks SET ledger_public_id=$1 WHERE id=$2`, ledgerID, internalID); err != nil {
		return nil, "", 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE task_quotes SET consumed_at=NOW() WHERE public_id=$1`, quote.ID); err != nil {
		return nil, "", 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, "", 0, err
	}
	return &Task{ID: taskID, Title: title, Workflow: quote.ModeName, Status: "排队中", DurationSec: quote.DurationSeconds, Resolution: quote.Resolution, Cost: quote.Total, CreatedAt: time.Now(), UpdatedAt: time.Now()}, ledgerID, balance, nil
}

func (s *Store) getQuoteForConsume(ctx context.Context, tx pgx.Tx, userID int64, quoteID string) (*TaskQuote, error) {
	var quote TaskQuote
	var requestRaw, retailRaw, costRaw, offeringRaw, resolutionsRaw, aspectsRaw []byte
	var retailVersion, costVersion string
	var consumedAt *time.Time
	var minDuration, maxDuration, minImages, maxImages int
	var supportsGeneratedAudio bool
	err := tx.QueryRow(ctx, `
SELECT q.public_id, q.expires_at, q.consumed_at, mode.public_id, mode.code, mode.name,
       model.public_id, model.code, model.name, carrier.public_id, carrier.code, carrier.name,
       offering.public_id, offering.name, supplier.code, supplier.base_url, supplier.auth_header,
       offering.upstream_workflow_id, offering.adapter_code, offering.min_duration_sec, offering.max_duration_sec,
       offering.resolutions, offering.aspect_ratios, offering.min_images, offering.max_images, offering.supports_generated_audio,
       cost.public_id, cost.version, retail.public_id, retail.version,
       q.request_snapshot, q.retail_snapshot, q.cost_snapshot, q.offering_snapshot
FROM task_quotes q
JOIN creation_modes mode ON mode.id=q.mode_id
JOIN models model ON model.id=q.model_id
JOIN carriers carrier ON carrier.id=model.carrier_id
JOIN supplier_offerings offering ON offering.id=q.offering_id AND offering.adapter_code='autodl_h3_v1'
JOIN suppliers supplier ON supplier.id=offering.supplier_id
JOIN supplier_cost_rules cost ON cost.id=q.cost_rule_id
JOIN retail_price_rules retail ON retail.id=q.retail_rule_id
WHERE q.public_id=$1 AND q.user_id=$2
	  AND q.expires_at > NOW()
  AND mode.status='published' AND model.enabled AND carrier.enabled
  AND offering.enabled AND offering.health_status <> 'unavailable' AND supplier.enabled AND supplier.token_encrypted <> ''
  AND cost.status='active' AND (cost.effective_from IS NULL OR cost.effective_from <= NOW())
  AND (cost.effective_to IS NULL OR cost.effective_to > NOW())
  AND retail.status='active' AND (retail.effective_from IS NULL OR retail.effective_from <= NOW())
  AND (retail.effective_to IS NULL OR retail.effective_to > NOW())
FOR UPDATE`, quoteID, userID).Scan(
		&quote.ID, &quote.ExpiresAt, &consumedAt, &quote.ModeID, &quote.ModeCode, &quote.ModeName,
		&quote.ModelID, &quote.ModelCode, &quote.ModelName, &quote.CarrierID, &quote.CarrierCode, &quote.CarrierName,
		&quote.OfferingID, &quote.OfferingName, &quote.SupplierCode, &quote.SupplierBaseURL, &quote.SupplierAuth,
		&quote.UpstreamWorkflow, &quote.AdapterCode, &minDuration, &maxDuration, &resolutionsRaw, &aspectsRaw, &minImages, &maxImages, &supportsGeneratedAudio,
		&quote.CostRuleID, &costVersion, &quote.RetailRuleID, &retailVersion,
		&requestRaw, &retailRaw, &costRaw, &offeringRaw,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("报价已过期或不可用")
	}
	if err != nil {
		return nil, err
	}
	if consumedAt != nil {
		return nil, errors.New("报价已使用")
	}
	_ = json.Unmarshal(retailRaw, &quote.RetailSnapshot)
	_ = json.Unmarshal(costRaw, &quote.CostSnapshot)
	_ = json.Unmarshal(offeringRaw, &quote.OfferingSnapshot)
	if quote.RetailSnapshot["version"] != retailVersion || quote.CostSnapshot["version"] != costVersion {
		return nil, errors.New("报价已过期或不可用")
	}
	var req struct {
		DurationSeconds int    `json:"durationSeconds"`
		Resolution      string `json:"resolution"`
		AspectRatio     string `json:"aspectRatio"`
		GenerateAudio   bool   `json:"generateAudio"`
	}
	_ = json.Unmarshal(requestRaw, &req)
	quote.DurationSeconds, quote.Resolution, quote.AspectRatio, quote.GenerateAudio = req.DurationSeconds, req.Resolution, req.AspectRatio, req.GenerateAudio
	currentOffering := SupplierOffering{MinDurationSec: minDuration, MaxDurationSec: maxDuration, MinImages: minImages, MaxImages: maxImages, SupportsGeneratedAudio: supportsGeneratedAudio}
	_ = json.Unmarshal(resolutionsRaw, &currentOffering.Resolutions)
	_ = json.Unmarshal(aspectsRaw, &currentOffering.AspectRatios)
	if !offeringMatches(currentOffering, DirectRouteRequest{DurationSeconds: quote.DurationSeconds, Resolution: quote.Resolution, AspectRatio: quote.AspectRatio, GenerateAudio: quote.GenerateAudio}) {
		return nil, errors.New("报价已过期或不可用")
	}
	quote.MinImages, quote.MaxImages = minImages, maxImages
	quote.PricePerSecond, _ = quote.RetailSnapshot["pricePerSecond"].(float64)
	quote.Total, _ = quote.RetailSnapshot["total"].(float64)
	quote.CostPerSecond, _ = quote.CostSnapshot["costPerSecond"].(float64)
	if max, ok := quote.OfferingSnapshot["maxImages"].(float64); ok {
		quote.MaxImages = int(max)
	}
	return &quote, nil
}

func (s *Store) listMatchingOfferings(ctx context.Context, modeID, modelID string) ([]SupplierOffering, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT o.public_id, supplier.public_id, supplier.code, supplier.name, supplier.base_url, supplier.auth_header,
       model.public_id, model.code, model.name, carrier.name, mode.public_id, mode.code, mode.name,
       o.name, o.upstream_workflow_id, o.execution_type, o.api_model_code, o.api_version, o.workflow_code, o.adapter_code, o.enabled, o.health_status, o.quota_status, o.priority,
       o.min_duration_sec, o.max_duration_sec, o.resolutions, o.aspect_ratios, o.min_images, o.max_images,
       o.supports_generated_audio
FROM supplier_offerings o
JOIN suppliers supplier ON supplier.id=o.supplier_id AND supplier.enabled AND supplier.token_encrypted <> ''
JOIN models model ON model.id=o.model_id AND model.enabled
JOIN carriers carrier ON carrier.id=model.carrier_id AND carrier.enabled
JOIN creation_modes mode ON mode.id=o.mode_id AND mode.status='published'
WHERE mode.public_id=$1 AND model.public_id=$2 AND o.enabled AND o.health_status <> 'unavailable'
  AND o.adapter_code='autodl_h3_v1'
ORDER BY o.priority, o.public_id`, modeID, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SupplierOffering
	for rows.Next() {
		var item SupplierOffering
		var resRaw, aspectRaw []byte
		if err := rows.Scan(&item.ID, &item.SupplierID, &item.SupplierCode, &item.SupplierName, &item.SupplierBaseURL, &item.SupplierAuthHeader,
			&item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName, &item.ModeID, &item.ModeCode, &item.ModeName,
			&item.Name, &item.UpstreamWorkflowID, &item.ExecutionType, &item.APIModelCode, &item.APIVersion, &item.WorkflowCode, &item.AdapterCode, &item.Enabled, &item.HealthStatus, &item.QuotaStatus, &item.Priority,
			&item.MinDurationSec, &item.MaxDurationSec, &resRaw, &aspectRaw, &item.MinImages, &item.MaxImages, &item.SupportsGeneratedAudio); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(resRaw, &item.Resolutions)
		_ = json.Unmarshal(aspectRaw, &item.AspectRatios)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) findRetailPrice(ctx context.Context, modeID, modelID, resolution string, generateAudio bool) (*RetailPriceRule, error) {
	var item RetailPriceRule
	err := s.Pool.QueryRow(ctx, `
SELECT retail.public_id, mode.public_id, mode.code, mode.name,
COALESCE(model.public_id, ''), COALESCE(model.code, ''), COALESCE(model.name, ''), COALESCE(carrier.name, ''),
       retail.resolution, retail.generate_audio, retail.name, retail.price_per_sec::float8, retail.version, retail.status
FROM retail_price_rules retail
JOIN creation_modes mode ON mode.id=retail.mode_id
JOIN models model ON model.id=retail.model_id
JOIN carriers carrier ON carrier.id=model.carrier_id
WHERE mode.public_id=$1 AND model.public_id=$2 AND retail.resolution=$3 AND retail.generate_audio=$4
  AND retail.status='active' AND (retail.effective_from IS NULL OR retail.effective_from <= NOW())
  AND (retail.effective_to IS NULL OR retail.effective_to > NOW())
ORDER BY retail.updated_at DESC LIMIT 1`, modeID, modelID, resolution, generateAudio).Scan(
		&item.ID, &item.ModeID, &item.ModeCode, &item.ModeName, &item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName,
		&item.Resolution, &item.GenerateAudio, &item.Name, &item.PricePerSecond, &item.Version, &item.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("当前模型暂无该参数售价")
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}
