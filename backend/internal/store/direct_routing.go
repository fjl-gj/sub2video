package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ResolveDirectRoute evaluates all enabled API and workflow executions for a
// selected model. It is intentionally used by both estimation and submission;
// callers decide whether to persist the result.
func (s *Store) ResolveDirectRoute(ctx context.Context, in DirectRouteRequest) (*ResolvedRoute, error) {
	if in.DurationSeconds < 1 || in.DurationSeconds > 60 {
		return nil, errors.New("视频时长不合法")
	}
	if strings.TrimSpace(in.Resolution) == "" || strings.TrimSpace(in.AspectRatio) == "" {
		return nil, errors.New("请完整选择生成设置")
	}
	mode, err := s.getCreationMode(ctx, in.ModeID)
	if err != nil || mode.Status != "published" {
		return nil, errors.New("创作模式未开放")
	}
	if err := validateModeImageCount(mode.Code, in.ImageCount); err != nil {
		return nil, err
	}
	modelID := strings.TrimSpace(in.ModelID)
	if modelID == "" {
		modelID, err = s.recommendedModelIDWithFallback(ctx, mode.ID, in.SupplierTokenFallback)
		if err != nil {
			return nil, err
		}
	}
	model, err := s.getCatalogModel(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if in.CarrierID != "" && model.CarrierID != in.CarrierID {
		return nil, errors.New("模型不属于所选承运商")
	}
	if !s.modelSupportsMode(ctx, mode.ID, model.ID) {
		return nil, errors.New("该模型不支持当前创作模式")
	}
	retail, err := s.findDirectRetailPrice(ctx, mode.ID, in.Resolution, in.GenerateAudio)
	if err != nil {
		return nil, err
	}
	offerings, err := s.listDirectOfferingsWithFallback(ctx, mode.ID, model.ID, in.SupplierTokenFallback)
	if err != nil {
		return nil, err
	}
	var selected *SupplierOffering
	var selectedCost float64
	var costID, costVersion string
	for i := range offerings {
		o := &offerings[i]
		if !offeringMatches(*o, QuoteInput{DurationSeconds: in.DurationSeconds, Resolution: in.Resolution, AspectRatio: in.AspectRatio, GenerateAudio: in.GenerateAudio, ImageCount: in.ImageCount}) {
			continue
		}
		cost, id, version, costErr := s.findSupplierCost(ctx, o.ID, in.Resolution, in.GenerateAudio)
		if costErr != nil {
			continue
		}
		if o.QuotaStatus == "unavailable" || o.QuotaStatus == "exhausted" {
			continue
		}
		if selected == nil || cost < selectedCost || (cost == selectedCost && (o.Priority < selected.Priority || (o.Priority == selected.Priority && o.ID < selected.ID))) {
			selected, selectedCost, costID, costVersion = o, cost, id, version
		}
	}
	if selected == nil {
		return nil, errors.New("当前参数暂无可用执行配置")
	}
	return &ResolvedRoute{
		Mode: mode, Model: model, Offering: selected,
		CostRuleID: costID, CostVersion: costVersion, CostPerSecond: selectedCost,
		Retail: retail, Total: roundCurrency(retail.PricePerSecond * float64(in.DurationSeconds)),
	}, nil
}

// roundCurrency keeps the public charge, wallet mutation, order amount and
// refund amount aligned with NUMERIC(12,2) billing columns.
func roundCurrency(value float64) float64 {
	// NUMERIC(12,2) rounds half away from zero. Add a tiny, scale-relative
	// epsilon before rounding so decimal inputs such as 1.005, whose binary
	// representation is just below the half-cent boundary, match PostgreSQL
	// and wallet expectations. The epsilon is deliberately bounded so values
	// that are materially below a cent boundary are unaffected.
	scaled := value * 100
	epsilon := math.Abs(scaled) * 1e-12
	if epsilon < 1e-12 {
		epsilon = 1e-12
	}
	if scaled < 0 {
		scaled -= epsilon
	} else {
		scaled += epsilon
	}
	return math.Round(scaled) / 100
}

func (s *Store) listDirectOfferings(ctx context.Context, modeID, modelID string) ([]SupplierOffering, error) {
	return s.listDirectOfferingsWithFallback(ctx, modeID, modelID, "")
}

func (s *Store) listDirectOfferingsWithFallback(ctx context.Context, modeID, modelID, tokenFallback string) ([]SupplierOffering, error) {
	rows, err := s.routeDB().Query(ctx, `
SELECT e.public_id, COALESCE(o.public_id,''), supplier.public_id, supplier.code, supplier.name, supplier.base_url, supplier.auth_header,
       (supplier.token_encrypted <> ''),
       model.public_id, model.code, model.name, carrier.name, mode.public_id, mode.code, mode.name,
       COALESCE(o.name, e.workflow_code || ' ' || model.name), e.workflow_code, e.execution_type, e.api_model_code, e.api_version, e.workflow_code,
       e.adapter_code, e.enabled, e.health_status, e.quota_status, e.priority,
       e.min_duration_sec, e.max_duration_sec, e.resolutions, e.aspect_ratios, e.min_images, e.max_images,
       e.supports_generated_audio, e.parameter_mapping
FROM supplier_executions e
JOIN supplier_model_bindings binding ON binding.id=e.binding_id AND binding.enabled AND binding.status='active'
JOIN suppliers supplier ON supplier.id=binding.supplier_id AND supplier.enabled
  AND supplier.status NOT IN ('disabled', 'unavailable', 'degraded')
JOIN models model ON model.id=binding.model_id AND model.enabled
JOIN carriers carrier ON carrier.id=model.carrier_id AND carrier.enabled
LEFT JOIN supplier_offerings o ON o.id=e.legacy_offering_id
JOIN creation_modes mode ON mode.id=e.mode_id AND mode.status='published'
WHERE mode.public_id=$1 AND model.public_id=$2 AND e.enabled AND e.health_status NOT IN ('degraded', 'unavailable')
  AND e.quota_status NOT IN ('unavailable', 'exhausted')
  AND e.execution_type IN ('api', 'workflow')
ORDER BY e.priority, e.public_id`, modeID, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SupplierOffering, 0)
	for rows.Next() {
		var item SupplierOffering
		var resRaw, aspectRaw, parameterRaw []byte
		if err := rows.Scan(&item.ID, &item.LegacyOfferingID, &item.SupplierID, &item.SupplierCode, &item.SupplierName, &item.SupplierBaseURL, &item.SupplierAuthHeader,
			&item.SupplierTokenConfigured,
			&item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName, &item.ModeID, &item.ModeCode, &item.ModeName,
			&item.Name, &item.UpstreamWorkflowID, &item.ExecutionType, &item.APIModelCode, &item.APIVersion, &item.WorkflowCode,
			&item.AdapterCode, &item.Enabled, &item.HealthStatus, &item.QuotaStatus, &item.Priority,
			&item.MinDurationSec, &item.MaxDurationSec, &resRaw, &aspectRaw, &item.MinImages, &item.MaxImages, &item.SupportsGeneratedAudio, &parameterRaw); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(resRaw, &item.Resolutions)
		_ = json.Unmarshal(aspectRaw, &item.AspectRatios)
		_ = json.Unmarshal(parameterRaw, &item.ParameterMapping)
		item.ExecutionID = item.ID
		if item.LegacyOfferingID != "" {
			item.UpstreamWorkflowID = item.WorkflowCode
		}
		if item.WorkflowCode == "" {
			item.WorkflowCode = item.UpstreamWorkflowID
		}
		if !item.SupplierTokenConfigured && !(strings.EqualFold(item.SupplierCode, "autodl") && strings.TrimSpace(tokenFallback) != "") {
			continue
		}
		// Do not allow an administrator to publish a configuration that the
		// current worker cannot execute. This keeps estimate and submission
		// from charging a task that can only fail at dispatch time.
		if !supportedProviderAdapter(item.AdapterCode) {
			continue
		}
		// The AutoDL API contract is intentionally not enabled until its
		// submit/poll endpoints are verified. Keeping it out of route selection
		// prevents a charge that can only end in SUBMIT_UNKNOWN.
		if item.ExecutionType == "api" && item.AdapterCode == "autodl_h3_v1" {
			continue
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) findDirectRetailPrice(ctx context.Context, modeID, resolution string, generateAudio bool) (*RetailPriceRule, error) {
	var item RetailPriceRule
	err := s.routeDB().QueryRow(ctx, `
SELECT retail.public_id, mode.public_id, mode.code, mode.name, '', '', '', '',
       retail.resolution, retail.generate_audio, retail.name, retail.price_per_sec::float8, retail.version, retail.status
FROM retail_price_rules retail
JOIN creation_modes mode ON mode.id=retail.mode_id
	WHERE mode.public_id=$1 AND retail.model_id IS NULL AND retail.resolution=$2 AND retail.generate_audio=$3
  AND retail.status='active' AND (retail.effective_from IS NULL OR retail.effective_from <= NOW())
  AND (retail.effective_to IS NULL OR retail.effective_to > NOW())
ORDER BY (retail.model_id IS NULL) DESC, retail.updated_at DESC LIMIT 1`, modeID, resolution, generateAudio).Scan(
		&item.ID, &item.ModeID, &item.ModeCode, &item.ModeName, &item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName,
		&item.Resolution, &item.GenerateAudio, &item.Name, &item.PricePerSecond, &item.Version, &item.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("当前模式暂无该参数售价")
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

type DirectTaskInput struct {
	DirectRouteRequest
	ClientRequestID string
	Title           string
	Prompt          string
	SourceMeta      SourceMeta
	Preflight       func(context.Context, *Store, *ResolvedRoute) error
}

type DirectTaskResult struct {
	Task        *Task
	OrderID     string
	OrderStatus string
	LedgerID    string
	Balance     float64
	AlreadyDone bool
}

func (s *Store) CreateTaskDirect(ctx context.Context, userID int64, in DirectTaskInput) (*DirectTaskResult, error) {
	requestID := strings.TrimSpace(in.ClientRequestID)
	if requestID == "" || len(requestID) > 128 {
		return nil, errors.New("缺少有效幂等请求 ID")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// Serialize the idempotency key inside the transaction so concurrent
	// retries cannot both pass the pre-check and charge twice.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, fmt.Sprintf("sub2video:%d:%s", userID, requestID)); err != nil {
		return nil, err
	}
	var existingTaskID, existingOrderID string
	var existingAmount, existingBalance float64
	var existingTitle, existingWorkflow, existingStatus, existingResolution string
	var existingProgress, existingDuration int
	var existingCreatedAt, existingUpdatedAt time.Time
	var existingOrderStatus string
	err = tx.QueryRow(ctx, `SELECT o.public_id, o.amount::float8, o.status,
       COALESCE(t.public_id,''), COALESCE(t.title,''), COALESCE(t.workflow,''),
       COALESCE(t.status,''), COALESCE(t.progress,0), COALESCE(t.duration_sec,0),
       COALESCE(t.resolution,''), w.balance::float8,
       COALESCE(t.created_at,NOW()), COALESCE(t.updated_at,NOW())
FROM consumption_orders o LEFT JOIN generation_tasks t ON t.id=o.task_id JOIN wallets w ON w.user_id=o.user_id
WHERE o.user_id=$1 AND o.client_request_id=$2`, userID, requestID).Scan(
		&existingOrderID, &existingAmount, &existingOrderStatus, &existingTaskID, &existingTitle, &existingWorkflow,
		&existingStatus, &existingProgress, &existingDuration, &existingResolution, &existingBalance,
		&existingCreatedAt, &existingUpdatedAt)
	if err == nil {
		return &DirectTaskResult{Task: &Task{
			ID: existingTaskID, Title: existingTitle, Workflow: existingWorkflow,
			Status: statusCN(existingStatus), Progress: existingProgress,
			DurationSec: existingDuration, Resolution: existingResolution, Cost: existingAmount,
			CreatedAt: existingCreatedAt, UpdatedAt: existingUpdatedAt,
		}, OrderID: existingOrderID, OrderStatus: existingOrderStatus, Balance: existingBalance, AlreadyDone: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	in.DirectRouteRequest.ImageCount = in.SourceMeta.ImageCount()

	// Keep the advisory lock held while routing and validating the request.
	// A retry that arrives after the first transaction commits will therefore
	// return the original order without re-running route selection or charging.
	routingStore := *s
	routingStore.routeTx = tx
	route, err := routingStore.ResolveDirectRoute(ctx, in.DirectRouteRequest)
	if err != nil {
		return nil, err
	}
	tmpQuote := &TaskQuote{ModeCode: route.Mode.Code, ModeName: route.Mode.Name, MinImages: route.Offering.MinImages, MaxImages: route.Offering.MaxImages}
	if err := validateModeInput(ctx, &routingStore, userID, tmpQuote, in.Prompt, &in.SourceMeta); err != nil {
		return nil, err
	}
	if in.SourceMeta.Audio != nil && !in.GenerateAudio {
		return nil, errors.New("上传参考音频需要启用音频设置")
	}
	if in.SourceMeta.Audio != nil {
		if err := routingStore.validateReferenceAudioSource(ctx, userID, in.SourceMeta.Audio); err != nil {
			return nil, err
		}
	}
	if in.Preflight != nil {
		if err := in.Preflight(ctx, &routingStore, route); err != nil {
			return nil, err
		}
	}
	var balance float64
	if err = tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id=$1 FOR UPDATE`, userID).Scan(&balance); err != nil {
		return nil, err
	}
	if balance < route.Total {
		return nil, errors.New("insufficient balance")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSpace(in.Prompt)
		runes := []rune(title)
		if len(runes) > 20 {
			title = string(runes[:20])
		}
	}
	if title == "" {
		title = "未命名创作"
	}
	publicTaskID, orderID, ledgerID := NewTaskID(), NewOrderID(), NewLedgerID()
	sourceType, sourceRef := sourceFromMeta(route.Mode.Code, in.SourceMeta)
	metaRaw, _ := json.Marshal(in.SourceMeta)
	inputRaw, _ := json.Marshal(map[string]any{"modeId": route.Mode.ID, "modelId": route.Model.ID, "prompt": in.Prompt, "sourceMeta": in.SourceMeta, "durationSeconds": in.DurationSeconds, "resolution": in.Resolution, "aspectRatio": in.AspectRatio, "generateAudio": in.GenerateAudio})
	retailSnapshot := map[string]any{"ruleId": route.Retail.ID, "version": route.Retail.Version, "pricePerSecond": route.Retail.PricePerSecond, "total": route.Total, "currency": "CNY"}
	costSnapshot := map[string]any{"ruleId": route.CostRuleID, "version": route.CostVersion, "costPerSecond": route.CostPerSecond, "estimatedTotal": route.CostPerSecond * float64(in.DurationSeconds), "currency": "CNY"}
	executionSnapshot := map[string]any{"offeringId": route.Offering.ID, "supplierCode": route.Offering.SupplierCode, "executionType": route.Offering.ExecutionType, "workflowCode": route.Offering.WorkflowCode, "apiModelCode": route.Offering.APIModelCode, "apiVersion": route.Offering.APIVersion, "upstreamWorkflowId": route.Offering.UpstreamWorkflowID, "adapterCode": route.Offering.AdapterCode, "parameterMapping": route.Offering.ParameterMapping}
	retailRaw, _ := json.Marshal(retailSnapshot)
	costRaw, _ := json.Marshal(costSnapshot)
	executionRaw, _ := json.Marshal(executionSnapshot)
	var taskInternalID int64
	err = tx.QueryRow(ctx, `INSERT INTO generation_tasks (
public_id,user_id,title,workflow,status,progress,duration_sec,resolution,cost,thumb_url,source_type,source_ref,source_meta,prompt,aspect_ratio,
workflow_public_id,price_rule_id,price_version,price_per_sec,cost_per_sec,price_snapshot,route_snapshot,carrier_code,model_code,supplier_code,supplier_route_id,upstream_workflow_id,
mode_public_id,model_public_id,generate_audio,quote_public_id,retail_snapshot,cost_snapshot,offering_snapshot,input_snapshot,client_request_id,consumption_order_public_id,execution_type,execution_snapshot,execution_mode)
VALUES ($1,$2,$3,$4,'queued',0,$5,$6,$7,'',$8,$9,$10::jsonb,$11,$12,$13,$14,$15,$16,$17,$18::jsonb,$19::jsonb,$20,$21,$22,$23,$24,$25,$26,$27,'',$28::jsonb,$29::jsonb,$30::jsonb,$31::jsonb,$32,$33,$34,$35::jsonb,'provider') RETURNING id`,
		publicTaskID, userID, title, route.Mode.Name, in.DurationSeconds, in.Resolution, route.Total, sourceType, sourceRef, string(metaRaw), in.Prompt, in.AspectRatio,
		route.Mode.Code, route.Retail.ID, route.Retail.Version, route.Retail.PricePerSecond, route.CostPerSecond, string(retailRaw), string(executionRaw), route.Model.CarrierCode, route.Model.Code, route.Offering.SupplierCode, route.Offering.ID, route.Offering.UpstreamWorkflowID,
		route.Mode.ID, route.Model.ID, in.GenerateAudio, string(retailRaw), string(costRaw), string(executionRaw), string(inputRaw), requestID, orderID, route.Offering.ExecutionType, string(executionRaw)).Scan(&taskInternalID)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance=balance-$1, version=version+1, updated_at=NOW() WHERE user_id=$2`, route.Total, userID); err != nil {
		return nil, err
	}
	if err = tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id=$1`, userID).Scan(&balance); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO wallet_ledger (public_id,user_id,type,amount,balance_after,ref_type,ref_id,description) VALUES ($1,$2,'task_charge',$3,$4,'order',$5,$6)`, ledgerID, userID, -route.Total, balance, orderID, fmt.Sprintf("%s %s", orderID, title)); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE generation_tasks SET ledger_public_id=$1 WHERE id=$2`, ledgerID, taskInternalID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO consumption_orders (public_id,user_id,task_id,client_request_id,amount,currency,status,retail_snapshot,cost_snapshot,execution_snapshot,paid_at) VALUES ($1,$2,$3,$4,$5,'CNY','paid',$6::jsonb,$7::jsonb,$8::jsonb,NOW())`, orderID, userID, taskInternalID, requestID, route.Total, string(retailRaw), string(costRaw), string(executionRaw)); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE generation_tasks SET consumption_order_public_id=$1 WHERE id=$2`, orderID, taskInternalID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &DirectTaskResult{Task: &Task{ID: publicTaskID, Title: title, Workflow: route.Mode.Name, Status: "排队中", DurationSec: in.DurationSeconds, Resolution: in.Resolution, Cost: route.Total, CreatedAt: time.Now(), UpdatedAt: time.Now()}, OrderID: orderID, OrderStatus: "paid", LedgerID: ledgerID, Balance: balance}, nil
}
