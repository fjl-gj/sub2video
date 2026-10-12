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

var ErrPromptRequired = errors.New("请先填写画面描述")

var errSupplierCostUnavailable = errors.New("当前供给暂无成本规则")

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
	supported, err := s.modelSupportsMode(ctx, mode.ID, modelID)
	if err != nil {
		return nil, err
	}
	if !supported {
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
			withoutAudio, err := s.hasBillableOfferingPair(ctx, mode.ID, modelID, offering.ID, value, false)
			if err != nil {
				return nil, err
			}
			withAudio := false
			if offering.SupportsGeneratedAudio {
				withAudio, err = s.hasBillableOfferingPair(ctx, mode.ID, modelID, offering.ID, value, true)
				if err != nil {
					return nil, err
				}
			}
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

// creationSourceConstraints is the material-validation boundary shared by
// direct submission and historical quote consumption. It has no price,
// expiry, or persisted quote identity.
type creationSourceConstraints struct {
	ModeCode  string
	MinImages int
	MaxImages int
}

func validateModeInput(ctx context.Context, s *Store, userID int64, constraints creationSourceConstraints, prompt string, meta *SourceMeta) error {
	if strings.TrimSpace(prompt) == "" {
		return ErrPromptRequired
	}
	if meta == nil {
		meta = &SourceMeta{}
	}
	if err := ValidateCreationSourceShape(constraints.ModeCode, *meta); err != nil {
		return err
	}
	if meta.Audio != nil {
		if err := s.validateReferenceAudioSource(ctx, userID, meta.Audio); err != nil {
			return err
		}
	}
	switch constraints.ModeCode {
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
		minimum := constraints.MinImages
		if minimum < 1 {
			minimum = 1
		}
		if len(meta.Refs) < minimum {
			return fmt.Errorf("多图生成视频至少需要 %d 张参考图", minimum)
		}
		if constraints.MaxImages > 0 && len(meta.Refs) > constraints.MaxImages {
			return fmt.Errorf("参考图最多 %d 张", constraints.MaxImages)
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

func (s *Store) modelSupportsMode(ctx context.Context, modeID, modelID string) (bool, error) {
	var ok bool
	err := s.routeDB().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mode_model_policies WHERE mode_id=(SELECT id FROM creation_modes WHERE public_id=$1) AND model_id=(SELECT id FROM models WHERE public_id=$2) AND enabled)`, modeID, modelID).Scan(&ok)
	return ok, err
}

func (s *Store) recommendedModelID(ctx context.Context, modeID string) (string, error) {
	return s.recommendedModelIDWithFallback(ctx, modeID, "")
}

func (s *Store) recommendedModelIDWithFallback(ctx context.Context, modeID, tokenFallback string) (string, error) {
	ids, err := s.recommendedModelIDsWithFallback(ctx, modeID, tokenFallback)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
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

// GetSupplierOfferingExecution returns the supply configuration locked onto a task.
// It is store-internal and must never be serialized to a client response.
func (s *Store) GetSupplierOfferingExecution(ctx context.Context, offeringID string) (*SupplierOffering, error) {
	return getSupplierOfferingExecution(ctx, s.routeDB(), offeringID)
}

func getSupplierOfferingExecution(ctx context.Context, q routeQuerier, offeringID string) (*SupplierOffering, error) {
	// Direct tasks store supplier_executions.public_id. Resolve that immutable
	// snapshot first, while retaining the legacy offering lookup below.
	var direct SupplierOffering
	var resRaw, aspectRaw []byte
	err := q.QueryRow(ctx, `
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
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	var item SupplierOffering
	var resolutions, aspects []byte
	err = q.QueryRow(ctx, `
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
		&item.Name, &item.UpstreamWorkflowID, &item.ExecutionType, &item.APIModelCode, &item.APIVersion, &item.WorkflowCode,
		&item.AdapterCode, &item.Enabled, &item.HealthStatus, &item.QuotaStatus, &item.Priority,
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
	item.LegacyOfferingID = item.ID
	return &item, nil
}

func offeringMatches(offering SupplierOffering, in DirectRouteRequest) bool {
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
func (s *Store) hasBillableOfferingPair(ctx context.Context, modeID, modelID, offeringID, resolution string, generateAudio bool) (bool, error) {
	var available bool
	err := s.routeDB().QueryRow(ctx, `
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
	return available, err
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
		return 0, "", "", errSupplierCostUnavailable
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
