package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const quoteTTL = 5 * time.Minute

type CreationMode struct {
	ID          string
	Code        string
	Name        string
	Description string
	Status      string
	SortOrder   int
}

type CreationCatalogModel struct {
	ID          string
	CarrierID   string
	CarrierCode string
	CarrierName string
	Code        string
	Name        string
	Description string
	Modes       []string
}

type CreationCatalog struct {
	Modes    []CreationMode
	Carriers []Carrier
	Models   []CreationCatalogModel
}

type CreationOptions struct {
	Durations              []int
	Resolutions            []string
	AspectRatios           []string
	SupportsGeneratedAudio bool
	MinImages              int
	MaxImages              int
	DefaultDuration        int
	DefaultResolution      string
}

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

type SupplierOffering struct {
	ID                      string
	ExecutionID             string
	LegacyOfferingID        string
	SupplierID              string
	SupplierCode            string
	SupplierName            string
	SupplierBaseURL         string
	SupplierAuthHeader      string
	SupplierTokenConfigured bool
	ModelID                 string
	ModelCode               string
	ModelName               string
	CarrierName             string
	ModeID                  string
	ModeCode                string
	ModeName                string
	Name                    string
	UpstreamWorkflowID      string
	ExecutionType           string
	APIModelCode            string
	APIVersion              string
	WorkflowCode            string
	AdapterCode             string
	Enabled                 bool
	HealthStatus            string
	QuotaStatus             string
	Priority                int
	MinDurationSec          int
	MaxDurationSec          int
	Resolutions             []string
	AspectRatios            []string
	MinImages               int
	MaxImages               int
	SupportsGeneratedAudio  bool
	ParameterMapping        map[string]any
	PeakCostPerSecond       float64
	OffpeakCostPerSecond    float64
}

// DirectRouteRequest is the client creation contract used for both estimates
// and task submission. Execution details are deliberately absent.
type DirectRouteRequest struct {
	ModeID          string
	CarrierID       string
	ModelID         string
	DurationSeconds int
	Resolution      string
	AspectRatio     string
	GenerateAudio   bool
	// ImageCount is derived from client source references and is used only for
	// capability filtering. The task snapshot remains the dispatch authority.
	ImageCount int
	// SupplierTokenFallback is an internal runtime credential hint. It is
	// populated by the handler from configuration and is never client input.
	SupplierTokenFallback string
}

// ResolvedRoute is an internal route selected from API and workflow
// executions. Never serialize this type directly to a client response.
type ResolvedRoute struct {
	Mode          *CreationMode
	Model         *CreationCatalogModel
	Offering      *SupplierOffering
	CostRuleID    string
	CostVersion   string
	CostPerSecond float64
	Retail        *RetailPriceRule
	Total         float64
}

type RetailPriceRule struct {
	ID             string
	ModeID         string
	ModeCode       string
	ModeName       string
	ModelID        string
	ModelCode      string
	ModelName      string
	CarrierName    string
	Resolution     string
	GenerateAudio  bool
	Name           string
	PricePerSecond float64
	Version        string
	Status         string
}

type SupplierCostRule struct {
	ID               string
	OfferingID       string
	ExecutionID      string
	Resolution       string
	GenerateAudio    bool
	PeakPerSecond    float64
	OffpeakPerSecond float64
	Currency         string
	Version          string
	Status           string
}

// ModelConfiguration is the administrator-facing ownership of a model's
// support for one fixed client workflow. The underlying relationship remains
// compatible with existing quote snapshots while the management surface is
// migrated away from workflow-owned policies.
type ModelConfiguration struct {
	ModelID            string
	ModelCode          string
	ModelName          string
	CarrierID          string
	CarrierName        string
	WorkflowID         string
	WorkflowName       string
	Enabled            bool
	RecommendationRank int
	DisplayBadge       string
	DefaultDuration    int
	DefaultResolution  string
}

type AdminCreationMode struct{ CreationMode }

func (s *Store) SeedCreationCatalog(ctx context.Context) error {
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO carriers (public_id, code, name, description, enabled, sort_order)
VALUES ('car_minimax', 'minimax', 'MINMAX', 'MiniMax 视频生成承运商', TRUE, 1)
ON CONFLICT (code) DO NOTHING`); err != nil {
		return fmt.Errorf("seed minimax carrier: %w", err)
	}
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO models (public_id, carrier_id, code, name, description, client_workflow_id, enabled, sort_order)
SELECT 'mdl_h3', id, 'h3', 'H3', 'MiniMax H3 视频模型', '', TRUE, 1 FROM carriers WHERE code='minimax'
ON CONFLICT (carrier_id, code) DO NOTHING`); err != nil {
		return fmt.Errorf("seed h3 model: %w", err)
	}
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO suppliers (public_id, code, name, base_url, auth_header, token_encrypted, enabled, status)
VALUES ('sup_autodl', 'autodl', 'AutoDL', 'https://autodl.art', 'Authorization', '', TRUE, 'unknown')
ON CONFLICT (code) DO NOTHING`); err != nil {
		return fmt.Errorf("seed autodl supplier: %w", err)
	}
	modes := []struct {
		id, code, name, desc string
		order                int
		contract             map[string]any
	}{
		{"mode_text_to_video", "text_to_video", "文生视频", "通过文字描述生成视频", 1, map[string]any{"prompt": true, "images": 0}},
		{"mode_first_last", "first_last_frame_to_video", "首尾帧生成视频", "使用首帧、尾帧和提示词生成视频", 2, map[string]any{"prompt": true, "firstFrame": true, "lastFrame": true}},
		{"mode_multi_image", "multi_image_to_video", "多图生成视频", "使用多张参考图和提示词生成视频", 3, map[string]any{"prompt": true, "minImages": 1}},
	}
	for _, item := range modes {
		raw, _ := json.Marshal(item.contract)
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO creation_modes (public_id, code, name, description, status, sort_order, input_contract)
VALUES ($1, $2, $3, $4, 'published', $5, $6::jsonb)
ON CONFLICT (code) DO NOTHING`, item.id, item.code, item.name, item.desc, item.order, string(raw)); err != nil {
			return fmt.Errorf("seed mode %s: %w", item.code, err)
		}
	}

	for _, item := range modes {
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO mode_model_policies (mode_id, model_id, enabled, recommendation_rank, display_badge, default_duration, default_resolution)
SELECT mode.id, model.id, TRUE, 1, '推荐', 5, '768P'
FROM creation_modes mode
JOIN models model ON model.public_id='mdl_h3'
WHERE mode.code=$1
ON CONFLICT (mode_id, model_id) DO NOTHING`, item.code); err != nil {
			return fmt.Errorf("seed h3 policy %s: %w", item.code, err)
		}
	}

	type offeringSeed struct {
		id, mode, name, workflow string
		maxDuration              int
		resolutions              []string
		audio                    bool
	}
	// Keep the initial production catalog intentionally small. These are the
	// four AutoDL H3 workflows that have been selected for the first creator
	// acceptance pass; adding an execution remains an explicit admin action.
	seeds := []offeringSeed{
		{"off_h3_text_lightx", "text_to_video", "H3 文生视频（无图）", "minimax_h3_lightx2v_no_pic", 15, []string{"480P", "768P"}, false},
		{"off_h3_first_last_lightx", "first_last_frame_to_video", "H3 首尾帧生成视频（10秒）", "minimax_h3_lightx2v", 10, []string{"480P", "768P"}, false},
		{"off_h3_multi_zm_u24", "multi_image_to_video", "H3 多图多音频生视频（升级画质）", "minimax_h3_zm_u24", 15, []string{"480P", "768P"}, true},
		{"off_h3_multi_zm_u08", "multi_image_to_video", "H3 多图多音频生视频（高速版）", "minimax_h3_zm_u08", 15, []string{"480P", "768P"}, true},
	}
	for _, item := range seeds {
		resolutions, _ := json.Marshal(item.resolutions)
		aspects, _ := json.Marshal([]string{"16:9", "9:16", "1:1"})
		minImages, maxImages := 0, 0
		if item.mode == "multi_image_to_video" {
			minImages, maxImages = 1, 9
		} else if item.mode == "first_last_frame_to_video" {
			minImages, maxImages = 2, 2
		}
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO supplier_offerings (
  public_id, supplier_id, model_id, mode_id, name, upstream_workflow_id, adapter_code,
  enabled, health_status, priority, min_duration_sec, max_duration_sec, resolutions, aspect_ratios,
  min_images, max_images, supports_generated_audio, workflow_code, parameter_mapping
)
SELECT $1, supplier.id, model.id, mode.id, $2, $3, 'autodl_h3_v1', TRUE, 'unknown', 10, 1, $4, $5::jsonb, $6::jsonb, $7, $8, $10, $3, $11::jsonb
FROM suppliers supplier
JOIN models model ON model.public_id='mdl_h3'
JOIN creation_modes mode ON mode.code=$9
WHERE supplier.code='autodl'
ON CONFLICT (supplier_id, model_id, mode_id, upstream_workflow_id) DO NOTHING`,
			item.id, item.name, item.workflow, item.maxDuration, string(resolutions), string(aspects), minImages, maxImages, item.mode, item.audio, "{}"); err != nil {
			return fmt.Errorf("seed offering %s: %w", item.mode, err)
		}
		audioOptions := []bool{false}
		if item.audio {
			audioOptions = append(audioOptions, true)
		}
		for _, resolution := range item.resolutions {
			peak, offpeak := 0.02, 0.01
			if resolution == "768P" {
				peak, offpeak = 0.03, 0.02
			}
			for _, generateAudio := range audioOptions {
				// Startup seeds fill missing defaults only. Existing costs and
				// their activation/effective windows are administrator-owned.
				if _, err := s.Pool.Exec(ctx, `
INSERT INTO supplier_cost_rules (public_id, offering_id, resolution, generate_audio, price_per_sec_peak, price_per_sec_offpeak, version, status)
SELECT $1, offering.id, $3, $4, $5, $6, 'v1', 'active'
FROM supplier_offerings offering WHERE offering.public_id=$2
ON CONFLICT (offering_id, resolution, generate_audio) DO NOTHING`, fmt.Sprintf("cost_%s_%s_%t", item.id, resolution, generateAudio), item.id, resolution, generateAudio, peak, offpeak); err != nil {
					return fmt.Errorf("seed cost %s %s audio=%t: %w", item.mode, resolution, generateAudio, err)
				}
			}
		}
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO retail_price_rules (public_id, mode_id, model_id, resolution, generate_audio, name, price_per_sec, version, status)
SELECT $1, mode.id, NULL, '768P', FALSE, $2, 0.03, 'v1', 'active'
FROM creation_modes mode
WHERE mode.code=$3
			ON CONFLICT DO NOTHING`,
			"ret_"+item.id, item.name+" / 768P", item.mode); err != nil {
			return fmt.Errorf("seed retail %s: %w", item.mode, err)
		}
		for _, resolution := range item.resolutions {
			if resolution == "768P" {
				continue
			}
			price := 0.02
			if resolution == "1080P" {
				price = 0.09
			}
			if _, err := s.Pool.Exec(ctx, `
INSERT INTO retail_price_rules (public_id, mode_id, model_id, resolution, generate_audio, name, price_per_sec, version, status)
SELECT $1, mode.id, NULL, $3, FALSE, $2, $4, 'v1', 'active'
FROM creation_modes mode WHERE mode.code=$5 ON CONFLICT DO NOTHING`, "ret_"+item.id+"_"+resolution, item.name+" / "+resolution, resolution, price, item.mode); err != nil {
				return fmt.Errorf("seed retail %s %s: %w", item.mode, resolution, err)
			}
		}
	}
	// Migrations run before runtime seeds. Mirror the migration backfill here
	// so a brand-new database receives the same unified execution records as an
	// upgraded UAT database.
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO supplier_model_bindings (public_id, supplier_id, model_id, enabled, status)
SELECT 'bind_' || md5(supplier.id::text || ':' || model.id::text), supplier.id, model.id, TRUE, 'active'
FROM supplier_offerings offering
JOIN suppliers supplier ON supplier.id=offering.supplier_id
JOIN models model ON model.id=offering.model_id
ON CONFLICT (supplier_id, model_id) DO NOTHING`); err != nil {
		return fmt.Errorf("seed supplier model bindings: %w", err)
	}
	if _, err := s.Pool.Exec(ctx, `
INSERT INTO supplier_executions (
  public_id, binding_id, mode_id, execution_type, api_model_code, api_version,
  workflow_code, adapter_code, enabled, health_status, quota_status, priority,
  min_duration_sec, max_duration_sec, resolutions, aspect_ratios, min_images,
  max_images, supports_generated_audio, parameter_mapping, result_mapping, legacy_offering_id
)
SELECT 'exec_' || offering.public_id, binding.id, offering.mode_id, offering.execution_type,
       offering.api_model_code, offering.api_version, offering.workflow_code, offering.adapter_code,
       offering.enabled, offering.health_status, offering.quota_status, offering.priority,
       offering.min_duration_sec, offering.max_duration_sec, offering.resolutions, offering.aspect_ratios,
       offering.min_images, offering.max_images, offering.supports_generated_audio,
       offering.parameter_mapping, offering.result_mapping, offering.id
FROM supplier_offerings offering
JOIN supplier_model_bindings binding
  ON binding.supplier_id=offering.supplier_id AND binding.model_id=offering.model_id
WHERE offering.execution_type IN ('api', 'workflow')
ON CONFLICT (legacy_offering_id) DO NOTHING`); err != nil {
		return fmt.Errorf("seed supplier executions: %w", err)
	}
	if _, err := s.Pool.Exec(ctx, `
UPDATE supplier_cost_rules cost
SET execution_id=execution.id
FROM supplier_executions execution
WHERE cost.execution_id IS NULL AND execution.legacy_offering_id=cost.offering_id`); err != nil {
		return fmt.Errorf("seed supplier cost execution links: %w", err)
	}
	// Platform retail pricing is mode-level. Audio remains a separate billing
	// dimension even where it currently shares the same retail rate, so direct
	// routing can price an audio-capable execution without falling back to a
	// non-audio rule.
	for _, item := range []struct {
		id, mode, resolution, name string
		price                      float64
	}{
		{"ret_mode_multi_480_audio", "multi_image_to_video", "480P", "平台统一售价 / 480P / 自动音频", 0.02},
		{"ret_mode_multi_768_audio", "multi_image_to_video", "768P", "平台统一售价 / 768P / 自动音频", 0.03},
	} {
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO retail_price_rules (public_id, mode_id, model_id, resolution, generate_audio, name, price_per_sec, version, status)
SELECT $1::varchar, mode.id, NULL, $3::varchar, TRUE, $4::varchar, $5::numeric, 'v1', 'active'
FROM creation_modes mode
WHERE mode.code=$2
  AND NOT EXISTS (
    SELECT 1 FROM retail_price_rules existing
    WHERE existing.mode_id=mode.id AND existing.model_id IS NULL
      AND existing.resolution=$3::varchar AND existing.generate_audio=TRUE
      AND existing.version='v1'
  )`, item.id, item.mode, item.resolution, item.name, item.price); err != nil {
			return fmt.Errorf("seed audio retail %s %s: %w", item.mode, item.resolution, err)
		}
	}
	return nil
}

// ListClientCreationCatalog returns the client-safe creation catalog using
// only credentials persisted on suppliers. The runtime-aware variant below
// is used by the HTTP handler when a provider credential is supplied through
// process configuration (for example SUB2VIDEO_AUTODL_TOKEN).
func (s *Store) ListClientCreationCatalog(ctx context.Context) (*CreationCatalog, error) {
	return s.ListClientCreationCatalogWithFallback(ctx, "")
}

// ListClientCreationCatalogWithFallback keeps catalog availability aligned
// with direct routing. A non-empty fallback is an internal credential hint;
// it is never serialized to the client or exposed in the catalog DTO.
func (s *Store) ListClientCreationCatalogWithFallback(ctx context.Context, tokenFallback string) (*CreationCatalog, error) {
	tokenFallback = strings.TrimSpace(tokenFallback)
	catalog := &CreationCatalog{}
	rows, err := s.Pool.Query(ctx, `
SELECT public_id, code, name, description, status, sort_order
FROM creation_modes WHERE status='published' ORDER BY sort_order, code`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item CreationMode
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Status, &item.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		catalog.Modes = append(catalog.Modes, item)
	}
	rows.Close()

	rows, err = s.Pool.Query(ctx, `
SELECT DISTINCT c.public_id, c.code, c.name, c.description, c.enabled, c.sort_order
FROM carriers c
JOIN models m ON m.carrier_id=c.id AND m.enabled
JOIN mode_model_policies p ON p.model_id=m.id AND p.enabled
JOIN creation_modes mode ON mode.id=p.mode_id AND mode.status='published'
JOIN supplier_model_bindings binding ON binding.model_id=m.id AND binding.enabled AND binding.status='active'
JOIN supplier_executions execution ON execution.binding_id=binding.id AND execution.mode_id=p.mode_id
  AND execution.enabled AND execution.health_status NOT IN ('degraded', 'unavailable')
  AND execution.quota_status NOT IN ('unavailable','exhausted')
  AND execution.execution_type='workflow'
  AND execution.adapter_code='autodl_h3_v1'
JOIN suppliers s ON s.id=binding.supplier_id AND s.enabled
  AND s.status NOT IN ('disabled', 'unavailable', 'degraded')
  AND (s.token_encrypted <> '' OR ($1 <> '' AND lower(s.code)='autodl'))
JOIN retail_price_rules r ON r.model_id IS NULL AND r.mode_id=p.mode_id AND r.status='active'
  AND (r.effective_from IS NULL OR r.effective_from <= NOW())
  AND (r.effective_to IS NULL OR r.effective_to > NOW())
  AND execution.resolutions ? r.resolution
  AND (NOT r.generate_audio OR execution.supports_generated_audio)
JOIN supplier_cost_rules cost ON cost.execution_id=execution.id AND cost.status='active'
  AND cost.resolution=r.resolution AND cost.generate_audio=r.generate_audio
  AND (cost.effective_from IS NULL OR cost.effective_from <= NOW())
  AND (cost.effective_to IS NULL OR cost.effective_to > NOW())
WHERE c.enabled
ORDER BY c.sort_order, c.code`, tokenFallback)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item Carrier
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Enabled, &item.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		catalog.Carriers = append(catalog.Carriers, item)
	}
	rows.Close()

	rows, err = s.Pool.Query(ctx, `
SELECT m.public_id, c.public_id, c.code, c.name, m.code, m.name, m.description,
       array_agg(DISTINCT mode.code ORDER BY mode.code)
FROM models m
JOIN carriers c ON c.id=m.carrier_id AND c.enabled
JOIN mode_model_policies p ON p.model_id=m.id AND p.enabled
JOIN creation_modes mode ON mode.id=p.mode_id AND mode.status='published'
JOIN supplier_model_bindings binding ON binding.model_id=m.id AND binding.enabled AND binding.status='active'
JOIN supplier_executions execution ON execution.binding_id=binding.id AND execution.mode_id=mode.id
  AND execution.enabled AND execution.health_status NOT IN ('degraded', 'unavailable')
  AND execution.quota_status NOT IN ('unavailable','exhausted')
  AND execution.execution_type='workflow'
  AND execution.adapter_code='autodl_h3_v1'
JOIN suppliers supplier ON supplier.id=binding.supplier_id AND supplier.enabled
  AND supplier.status NOT IN ('disabled', 'unavailable', 'degraded')
  AND (supplier.token_encrypted <> '' OR ($1 <> '' AND lower(supplier.code)='autodl'))
JOIN retail_price_rules retail ON retail.model_id IS NULL AND retail.mode_id=mode.id AND retail.status='active'
  AND (retail.effective_from IS NULL OR retail.effective_from <= NOW())
  AND (retail.effective_to IS NULL OR retail.effective_to > NOW())
  AND execution.resolutions ? retail.resolution
  AND (NOT retail.generate_audio OR execution.supports_generated_audio)
JOIN supplier_cost_rules cost ON cost.execution_id=execution.id AND cost.status='active'
  AND cost.resolution=retail.resolution AND cost.generate_audio=retail.generate_audio
  AND (cost.effective_from IS NULL OR cost.effective_from <= NOW())
  AND (cost.effective_to IS NULL OR cost.effective_to > NOW())
WHERE m.enabled
GROUP BY m.public_id, c.public_id, c.code, c.name, m.code, m.name, m.description, m.sort_order
ORDER BY c.code, m.sort_order, m.code`, tokenFallback)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item CreationCatalogModel
		if err := rows.Scan(&item.ID, &item.CarrierID, &item.CarrierCode, &item.CarrierName, &item.Code, &item.Name, &item.Description, &item.Modes); err != nil {
			rows.Close()
			return nil, err
		}
		catalog.Models = append(catalog.Models, item)
	}
	return catalog, rows.Err()
}

func (s *Store) ResolveCreationOptions(ctx context.Context, modeID, modelID string) (*CreationOptions, error) {
	return s.ResolveCreationOptionsWithFallback(ctx, modeID, modelID, "")
}

// ResolveCreationOptionsWithFallback uses the same internal credential
// fallback as direct estimate/submission routing. The returned options remain
// client-safe capability data only.
func (s *Store) ResolveCreationOptionsWithFallback(ctx context.Context, modeID, modelID, tokenFallback string) (*CreationOptions, error) {
	mode, err := s.getCreationMode(ctx, modeID)
	if err != nil {
		return nil, err
	}
	if mode.Status != "published" {
		return nil, errors.New("创作模式未开放")
	}
	if modelID == "" {
		modelID, err = s.recommendedModelIDWithFallback(ctx, mode.ID, tokenFallback)
		if err != nil {
			return nil, err
		}
	}
	if !s.modelSupportsMode(ctx, mode.ID, modelID) {
		return nil, errors.New("该模型不支持当前创作模式")
	}
	offerings, err := s.listDirectOfferingsWithFallback(ctx, mode.ID, modelID, tokenFallback)
	if err != nil {
		return nil, err
	}
	if len(offerings) == 0 {
		return nil, errors.New("当前模型暂无可用供给")
	}
	options := &CreationOptions{DefaultDuration: 5, DefaultResolution: "768P"}
	durations := map[int]bool{}
	resolutions := map[string]bool{}
	aspects := map[string]bool{}
	for _, offering := range offerings {
		nonAudioAvailable := false
		for _, value := range offering.Resolutions {
			withoutAudio := s.hasBillableOfferingPair(ctx, mode.ID, modelID, offering.ID, value, false)
			withAudio := offering.SupportsGeneratedAudio && s.hasBillableOfferingPair(ctx, mode.ID, modelID, offering.ID, value, true)
			if !withoutAudio && !withAudio {
				continue
			}
			resolutions[value], nonAudioAvailable = true, true
			if withAudio {
				options.SupportsGeneratedAudio = true
			}
		}
		if !nonAudioAvailable {
			continue
		}
		for n := offering.MinDurationSec; n <= offering.MaxDurationSec && n <= 60; n++ {
			durations[n] = true
		}
		for _, value := range offering.AspectRatios {
			aspects[value] = true
		}
		if offering.MinImages > 0 && (options.MinImages == 0 || offering.MinImages < options.MinImages) {
			options.MinImages = offering.MinImages
		}
		if offering.MaxImages > options.MaxImages {
			options.MaxImages = offering.MaxImages
		}
	}
	for value := range durations {
		options.Durations = append(options.Durations, value)
	}
	for value := range resolutions {
		options.Resolutions = append(options.Resolutions, value)
	}
	for value := range aspects {
		options.AspectRatios = append(options.AspectRatios, value)
	}
	sort.Ints(options.Durations)
	sort.Strings(options.Resolutions)
	sort.Strings(options.AspectRatios)
	if mode.Code == "first_last_frame_to_video" {
		options.MinImages, options.MaxImages = 2, 2
	}
	if len(options.Resolutions) == 0 || len(options.Durations) == 0 || len(options.AspectRatios) == 0 {
		return nil, errors.New("当前模型暂无可用供给")
	}
	return options, nil
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
	if !s.modelSupportsMode(ctx, mode.ID, model.ID) {
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
		if !offeringMatches(*offering, in) {
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
	if err := validateModeInput(ctx, s, userID, quote, prompt, &sourceMeta); err != nil {
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

func validateModeInput(ctx context.Context, s *Store, userID int64, quote *TaskQuote, prompt string, meta *SourceMeta) error {
	if strings.TrimSpace(prompt) == "" {
		return errors.New("请先填写画面描述")
	}
	if meta == nil {
		meta = &SourceMeta{}
	}
	if err := ValidateCreationSourceShape(quote.ModeCode, *meta); err != nil {
		return err
	}
	if meta.Audio != nil {
		if err := s.validateReferenceAudioSource(ctx, userID, meta.Audio); err != nil {
			return err
		}
	}
	switch quote.ModeCode {
	case "text_to_video":
		return nil
	case "first_last_frame_to_video":
		if meta.First == nil || meta.Last == nil {
			return errors.New("首尾帧生成视频需要首帧和尾帧")
		}
		if err := s.validateSourceRefItem(ctx, userID, meta.First); err != nil {
			return err
		}
		return s.validateSourceRefItem(ctx, userID, meta.Last)
	case "multi_image_to_video":
		minimum := quote.MinImages
		if minimum < 1 {
			minimum = 1
		}
		if len(meta.Refs) < minimum {
			return fmt.Errorf("多图生成视频至少需要 %d 张参考图", minimum)
		}
		if quote.MaxImages > 0 && len(meta.Refs) > quote.MaxImages {
			return fmt.Errorf("参考图最多 %d 张", quote.MaxImages)
		}
		for i := range meta.Refs {
			if err := s.validateSourceRefItem(ctx, userID, &meta.Refs[i]); err != nil {
				return err
			}
		}
		return nil
	default:
		return errors.New("不支持的创作模式")
	}
}

func sourceFromMeta(modeCode string, meta SourceMeta) (string, string) {
	if modeCode == "first_last_frame_to_video" && meta.First != nil {
		return meta.First.Type, meta.First.ID
	}
	if modeCode == "multi_image_to_video" && len(meta.Refs) > 0 {
		return meta.Refs[0].Type, meta.Refs[0].ID
	}
	return "text", ""
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
	if !offeringMatches(currentOffering, QuoteInput{DurationSeconds: quote.DurationSeconds, Resolution: quote.Resolution, AspectRatio: quote.AspectRatio, GenerateAudio: quote.GenerateAudio}) {
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

func (s *Store) getCreationMode(ctx context.Context, publicID string) (*CreationMode, error) {
	var item CreationMode
	err := s.routeDB().QueryRow(ctx, `SELECT public_id, code, name, description, status, sort_order FROM creation_modes WHERE public_id=$1`, publicID).Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Status, &item.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) getCatalogModel(ctx context.Context, publicID string) (*CreationCatalogModel, error) {
	var item CreationCatalogModel
	err := s.routeDB().QueryRow(ctx, `
SELECT m.public_id, c.public_id, c.code, c.name, m.code, m.name, m.description
FROM models m JOIN carriers c ON c.id=m.carrier_id
WHERE m.public_id=$1 AND m.enabled AND c.enabled`, publicID).Scan(&item.ID, &item.CarrierID, &item.CarrierCode, &item.CarrierName, &item.Code, &item.Name, &item.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("模型不存在或未启用")
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) modelSupportsMode(ctx context.Context, modeID, modelID string) bool {
	var ok bool
	_ = s.routeDB().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mode_model_policies WHERE mode_id=(SELECT id FROM creation_modes WHERE public_id=$1) AND model_id=(SELECT id FROM models WHERE public_id=$2) AND enabled)`, modeID, modelID).Scan(&ok)
	return ok
}

func (s *Store) recommendedModelID(ctx context.Context, modeID string) (string, error) {
	return s.recommendedModelIDWithFallback(ctx, modeID, "")
}

func (s *Store) recommendedModelIDWithFallback(ctx context.Context, modeID, tokenFallback string) (string, error) {
	ids, err := s.recommendedModelIDsWithFallback(ctx, modeID, tokenFallback)
	if err != nil || len(ids) == 0 {
		return "", errors.New("当前模式暂无可用模型")
	}
	return ids[0], nil
}

func (s *Store) recommendedModelIDs(ctx context.Context, modeID string) ([]string, error) {
	return s.recommendedModelIDsWithFallback(ctx, modeID, "")
}

func (s *Store) recommendedModelIDsWithFallback(ctx context.Context, modeID, tokenFallback string) ([]string, error) {
	tokenFallback = strings.TrimSpace(tokenFallback)
	rows, err := s.routeDB().Query(ctx, `
SELECT m.public_id
FROM mode_model_policies p
JOIN models m ON m.id=p.model_id AND m.enabled
JOIN carriers c ON c.id=m.carrier_id AND c.enabled
WHERE p.mode_id=(SELECT id FROM creation_modes WHERE public_id=$1 AND status='published') AND p.enabled
  AND EXISTS (
    SELECT 1
    FROM supplier_executions execution
    JOIN supplier_model_bindings binding ON binding.id=execution.binding_id
      AND binding.model_id=m.id AND binding.enabled AND binding.status='active'
    JOIN suppliers supplier ON supplier.id=binding.supplier_id
      AND supplier.enabled
      AND supplier.status NOT IN ('disabled', 'unavailable', 'degraded')
      AND (supplier.token_encrypted <> '' OR (LOWER(supplier.code)='autodl' AND $2 <> ''))
    JOIN supplier_cost_rules cost ON cost.execution_id=execution.id
      AND cost.status='active'
      AND (cost.effective_from IS NULL OR cost.effective_from <= NOW())
      AND (cost.effective_to IS NULL OR cost.effective_to > NOW())
    JOIN retail_price_rules retail ON retail.mode_id=execution.mode_id AND retail.model_id IS NULL
      AND retail.resolution=cost.resolution AND retail.generate_audio=cost.generate_audio
      AND retail.status='active'
      AND (retail.effective_from IS NULL OR retail.effective_from <= NOW())
      AND (retail.effective_to IS NULL OR retail.effective_to > NOW())
      AND execution.resolutions ? retail.resolution
      AND (NOT retail.generate_audio OR execution.supports_generated_audio)
    WHERE execution.mode_id=p.mode_id
      AND execution.enabled AND execution.health_status NOT IN ('degraded', 'unavailable')
      AND execution.quota_status NOT IN ('unavailable', 'exhausted')
      AND execution.execution_type='workflow'
      AND execution.adapter_code='autodl_h3_v1'
  )
  AND EXISTS (
    SELECT 1 FROM retail_price_rules retail
    WHERE retail.mode_id=p.mode_id AND retail.model_id IS NULL AND retail.status='active'
      AND (retail.effective_from IS NULL OR retail.effective_from <= NOW())
      AND (retail.effective_to IS NULL OR retail.effective_to > NOW())
  )
GROUP BY m.public_id
ORDER BY MIN(p.recommendation_rank), m.public_id`, modeID, tokenFallback)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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

// GetSupplierOfferingExecution returns the supply configuration locked onto a task.
// It is store-internal and must never be serialized to a client response.
func (s *Store) GetSupplierOfferingExecution(ctx context.Context, offeringID string) (*SupplierOffering, error) {
	// Direct tasks store supplier_executions.public_id. Resolve that immutable
	// snapshot first, while retaining the legacy offering lookup below.
	var direct SupplierOffering
	var resRaw, aspectRaw []byte
	err := s.Pool.QueryRow(ctx, `
SELECT e.public_id, COALESCE(o.public_id,''), supplier.public_id, supplier.code, supplier.name, supplier.base_url, supplier.auth_header,
       model.public_id, model.code, model.name, carrier.name, mode.public_id, mode.code, mode.name,
       COALESCE(o.name, e.workflow_code || ' ' || model.name), e.workflow_code, e.execution_type, e.api_model_code, e.api_version, e.workflow_code,
       e.adapter_code, e.enabled, e.health_status, e.quota_status, e.priority, e.min_duration_sec, e.max_duration_sec,
       e.resolutions, e.aspect_ratios, e.min_images, e.max_images, e.supports_generated_audio
FROM supplier_executions e
JOIN supplier_model_bindings b ON b.id=e.binding_id
JOIN suppliers supplier ON supplier.id=b.supplier_id
JOIN models model ON model.id=b.model_id
JOIN carriers carrier ON carrier.id=model.carrier_id
JOIN creation_modes mode ON mode.id=e.mode_id
LEFT JOIN supplier_offerings o ON o.id=e.legacy_offering_id
WHERE e.public_id=$1 OR o.public_id=$1
ORDER BY (e.public_id=$1) DESC LIMIT 1`, offeringID).Scan(
		&direct.ID, &direct.LegacyOfferingID, &direct.SupplierID, &direct.SupplierCode, &direct.SupplierName, &direct.SupplierBaseURL, &direct.SupplierAuthHeader,
		&direct.ModelID, &direct.ModelCode, &direct.ModelName, &direct.CarrierName, &direct.ModeID, &direct.ModeCode, &direct.ModeName,
		&direct.Name, &direct.UpstreamWorkflowID, &direct.ExecutionType, &direct.APIModelCode, &direct.APIVersion, &direct.WorkflowCode,
		&direct.AdapterCode, &direct.Enabled, &direct.HealthStatus, &direct.QuotaStatus, &direct.Priority, &direct.MinDurationSec, &direct.MaxDurationSec,
		&resRaw, &aspectRaw, &direct.MinImages, &direct.MaxImages, &direct.SupportsGeneratedAudio)
	if err == nil {
		direct.ExecutionID = direct.ID
		_ = json.Unmarshal(resRaw, &direct.Resolutions)
		_ = json.Unmarshal(aspectRaw, &direct.AspectRatios)
		return &direct, nil
	}
	var item SupplierOffering
	var resolutions, aspects []byte
	err = s.Pool.QueryRow(ctx, `
SELECT o.public_id, supplier.public_id, supplier.code, supplier.name, supplier.base_url, supplier.auth_header,
       model.public_id, model.code, model.name, carrier.name, mode.public_id, mode.code, mode.name,
       o.name, o.upstream_workflow_id, o.execution_type, o.api_model_code, o.api_version, o.workflow_code, o.adapter_code, o.enabled, o.health_status, o.quota_status, o.priority,
       o.min_duration_sec, o.max_duration_sec, o.resolutions, o.aspect_ratios, o.min_images, o.max_images,
       o.supports_generated_audio
FROM supplier_offerings o
JOIN suppliers supplier ON supplier.id=o.supplier_id
JOIN models model ON model.id=o.model_id
JOIN carriers carrier ON carrier.id=model.carrier_id
JOIN creation_modes mode ON mode.id=o.mode_id
WHERE o.public_id=$1`, offeringID).Scan(
		&item.ID, &item.SupplierID, &item.SupplierCode, &item.SupplierName, &item.SupplierBaseURL, &item.SupplierAuthHeader,
		&item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName, &item.ModeID, &item.ModeCode, &item.ModeName,
		&item.Name, &item.UpstreamWorkflowID, &item.AdapterCode, &item.Enabled, &item.HealthStatus, &item.Priority,
		&item.MinDurationSec, &item.MaxDurationSec, &resolutions, &aspects, &item.MinImages, &item.MaxImages, &item.SupportsGeneratedAudio,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(resolutions, &item.Resolutions)
	_ = json.Unmarshal(aspects, &item.AspectRatios)
	return &item, nil
}

func offeringMatches(offering SupplierOffering, in QuoteInput) bool {
	if in.DurationSeconds < offering.MinDurationSec || in.DurationSeconds > offering.MaxDurationSec {
		return false
	}
	if in.GenerateAudio && !offering.SupportsGeneratedAudio {
		return false
	}
	// A configured minimum is a hard capability requirement. Treat an absent
	// reference list as zero so estimates cannot route a multi-image request to
	// an execution that will reject it later during submission.
	if offering.MinImages > 0 && in.ImageCount < offering.MinImages {
		return false
	}
	if offering.MaxImages > 0 && in.ImageCount > offering.MaxImages {
		return false
	}
	return includes(offering.Resolutions, in.Resolution) && includes(offering.AspectRatios, in.AspectRatio)
}

// hasBillableOfferingPair keeps the catalog from advertising a resolution that
// cannot be both procured and sold for the selected mode and model.
func (s *Store) hasBillableOfferingPair(ctx context.Context, modeID, modelID, offeringID, resolution string, generateAudio bool) bool {
	var available bool
	err := s.Pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1
  FROM supplier_cost_rules cost
	JOIN supplier_executions execution ON execution.id=cost.execution_id
	JOIN supplier_model_bindings binding ON binding.id=execution.binding_id
	JOIN models model ON model.id=binding.model_id AND model.public_id=$2
  JOIN retail_price_rules retail ON retail.mode_id=(SELECT id FROM creation_modes WHERE public_id=$1)
    AND retail.model_id IS NULL
    AND retail.resolution=$4 AND retail.generate_audio=$5
	WHERE execution.public_id=$3 AND execution.mode_id=retail.mode_id
    AND cost.resolution=$4 AND cost.generate_audio=$5
    AND cost.status='active' AND (cost.effective_from IS NULL OR cost.effective_from <= NOW())
    AND (cost.effective_to IS NULL OR cost.effective_to > NOW())
    AND retail.status='active' AND (retail.effective_from IS NULL OR retail.effective_from <= NOW())
    AND (retail.effective_to IS NULL OR retail.effective_to > NOW())
)`, modeID, modelID, offeringID, resolution, generateAudio).Scan(&available)
	return err == nil && available
}

// CheapestAvailableRetailPrice is the overview-only reference price. Unlike
// the historical pricing table, it only considers combinations that can be
// quoted and executed by the current creation catalog.
func (s *Store) CheapestAvailableRetailPrice(ctx context.Context) (float64, bool, error) {
	return s.CheapestAvailableRetailPriceWithFallback(ctx, "")
}

// CheapestAvailableRetailPriceWithFallback is the overview-only reference
// price for the same execution catalog used by direct routing. The fallback
// token is intentionally limited to the configured AutoDL environment token;
// it is never returned to the client.
func (s *Store) CheapestAvailableRetailPriceWithFallback(ctx context.Context, tokenFallback string) (float64, bool, error) {
	tokenFallback = strings.TrimSpace(tokenFallback)
	var price *float64
	err := s.Pool.QueryRow(ctx, `
SELECT MIN(retail.price_per_sec::float8)
FROM retail_price_rules retail
JOIN creation_modes mode ON mode.id=retail.mode_id AND mode.status='published'
JOIN supplier_executions execution ON execution.mode_id=mode.id
  AND execution.enabled AND execution.health_status NOT IN ('degraded', 'unavailable')
  AND execution.quota_status NOT IN ('unavailable', 'exhausted')
  AND execution.adapter_code='autodl_h3_v1'
  AND execution.execution_type='workflow'
  AND execution.resolutions ? retail.resolution
  AND (NOT retail.generate_audio OR execution.supports_generated_audio)
JOIN supplier_model_bindings binding ON binding.id=execution.binding_id AND binding.enabled AND binding.status='active'
JOIN models model ON model.id=binding.model_id AND model.enabled
JOIN carriers carrier ON carrier.id=model.carrier_id AND carrier.enabled
JOIN mode_model_policies policy ON policy.mode_id=mode.id AND policy.model_id=model.id AND policy.enabled
JOIN suppliers supplier ON supplier.id=binding.supplier_id AND supplier.enabled
  AND supplier.status NOT IN ('disabled', 'unavailable', 'degraded')
	JOIN supplier_cost_rules cost ON cost.execution_id=execution.id AND cost.resolution=retail.resolution
  AND cost.generate_audio=retail.generate_audio AND cost.status='active'
  AND (cost.effective_from IS NULL OR cost.effective_from <= NOW())
  AND (cost.effective_to IS NULL OR cost.effective_to > NOW())
WHERE retail.model_id IS NULL AND retail.status='active'
  AND (retail.effective_from IS NULL OR retail.effective_from <= NOW())
  AND (retail.effective_to IS NULL OR retail.effective_to > NOW())
  AND (supplier.token_encrypted <> '' OR (LOWER(supplier.code)='autodl' AND $1 <> ''))`, tokenFallback).Scan(&price)
	if err != nil {
		return 0, false, err
	}
	if price == nil {
		return 0, false, nil
	}
	return *price, true, nil
}

func includes(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func supportedProviderAdapter(adapterCode string) bool {
	return adapterCode == "autodl_h3_v1"
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

func (s *Store) findSupplierCost(ctx context.Context, offeringID, resolution string, generateAudio bool) (float64, string, string, error) {
	var id, version string
	var peak, off float64
	var databaseNow time.Time
	err := s.routeDB().QueryRow(ctx, `
SELECT public_id, version, price_per_sec_peak::float8, price_per_sec_offpeak::float8, NOW()
FROM supplier_cost_rules
WHERE (execution_id=(SELECT id FROM supplier_executions WHERE public_id=$1)
       OR (execution_id IS NULL AND offering_id=(SELECT id FROM supplier_offerings WHERE public_id=$1)))
  AND resolution=$2 AND generate_audio=$3
  AND status='active' AND (effective_from IS NULL OR effective_from <= NOW()) AND (effective_to IS NULL OR effective_to > NOW())
ORDER BY updated_at DESC LIMIT 1`, offeringID, resolution, generateAudio).Scan(&id, &version, &peak, &off, &databaseNow)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", "", errors.New("当前供给暂无成本规则")
	}
	if err != nil {
		return 0, "", "", err
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	hour := databaseNow.In(loc).Hour()
	if hour >= 8 && hour < 24 {
		return peak, id, version, nil
	}
	return off, id, version, nil
}

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
	_, err := s.Pool.Exec(ctx, `
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
	// New execution tables are the source for direct routing. Keep a legacy
	// offering row for existing worker/task references during migration.
	_, err = s.Pool.Exec(ctx, `
INSERT INTO supplier_model_bindings (public_id, supplier_id, model_id)
SELECT 'bind_' || md5(supplier.id::text || ':' || model.id::text), supplier.id, model.id
FROM suppliers supplier, models model WHERE supplier.public_id=$1 AND model.public_id=$2
ON CONFLICT (supplier_id, model_id) DO NOTHING`, in.SupplierID, in.ModelID)
	if err != nil {
		return nil, err
	}
	_, err = s.Pool.Exec(ctx, `
INSERT INTO supplier_executions (public_id, binding_id, mode_id, execution_type, api_model_code, api_version, workflow_code,
 adapter_code, enabled, health_status, quota_status, priority, min_duration_sec, max_duration_sec, resolutions, aspect_ratios,
 min_images, max_images, supports_generated_audio, legacy_offering_id)
SELECT 'exec_' || $1, b.id, mode.id, $5, $6, $7, $8, $9, $10, $11, 'available', $12, $13, $14, $15::jsonb, $16::jsonb,
 $17, $18, $19, offering.id
FROM supplier_offerings offering
JOIN supplier_model_bindings b ON b.supplier_id=offering.supplier_id AND b.model_id=offering.model_id
JOIN creation_modes mode ON mode.public_id=$4
WHERE offering.public_id=$1
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
	return s.GetSupplierOfferingExecution(ctx, in.ID)
}

func (s *Store) UpdateSupplierOffering(ctx context.Context, offeringID string, in SupplierOffering) (*SupplierOffering, error) {
	current, err := s.GetSupplierOfferingExecution(ctx, offeringID)
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
		_, err = s.Pool.Exec(ctx, `UPDATE supplier_executions SET execution_type=$1, api_model_code=$2, api_version=$3, workflow_code=$4, adapter_code=$5, enabled=$6,
	health_status=$7, priority=$8, min_duration_sec=$9, max_duration_sec=$10, resolutions=$11::jsonb, aspect_ratios=$12::jsonb,
	min_images=$13, max_images=$14, supports_generated_audio=$15, updated_at=NOW() WHERE public_id=$16`,
			current.ExecutionType, current.APIModelCode, current.APIVersion, current.WorkflowCode, current.AdapterCode, current.Enabled, current.HealthStatus, current.Priority,
			current.MinDurationSec, current.MaxDurationSec, string(resRaw), string(aspectRaw), current.MinImages, current.MaxImages, current.SupportsGeneratedAudio, current.ID)
		if err != nil {
			return nil, err
		}
		return s.GetSupplierOfferingExecution(ctx, current.ID)
	}
	_, err = s.Pool.Exec(ctx, `UPDATE supplier_offerings SET name=$1, upstream_workflow_id=$2, execution_type=$3, api_model_code=$4, api_version=$5, workflow_code=$6, adapter_code=$7, enabled=$8,
health_status=$9, priority=$10, min_duration_sec=$11, max_duration_sec=$12, resolutions=$13::jsonb, aspect_ratios=$14::jsonb,
min_images=$15, max_images=$16, supports_generated_audio=$17, updated_at=NOW() WHERE public_id=$18`,
		current.Name, current.UpstreamWorkflowID, current.ExecutionType, current.APIModelCode, current.APIVersion, current.WorkflowCode, current.AdapterCode, current.Enabled, current.HealthStatus, current.Priority,
		current.MinDurationSec, current.MaxDurationSec, string(resRaw), string(aspectRaw), current.MinImages, current.MaxImages, current.SupportsGeneratedAudio, legacyID)
	if err != nil {
		return nil, err
	}
	_, err = s.Pool.Exec(ctx, `UPDATE supplier_executions e SET execution_type=o.execution_type, api_model_code=o.api_model_code, api_version=o.api_version,
workflow_code=o.workflow_code, adapter_code=o.adapter_code, enabled=o.enabled, health_status=o.health_status, priority=o.priority,
min_duration_sec=o.min_duration_sec, max_duration_sec=o.max_duration_sec, resolutions=o.resolutions, aspect_ratios=o.aspect_ratios,
min_images=o.min_images, max_images=o.max_images, supports_generated_audio=o.supports_generated_audio, updated_at=NOW()
FROM supplier_offerings o WHERE e.legacy_offering_id=o.id AND o.public_id=$1`, legacyID)
	if err != nil {
		return nil, err
	}
	return s.GetSupplierOfferingExecution(ctx, legacyID)
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
