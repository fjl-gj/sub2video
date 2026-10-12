package store

import (
	"context"
	"encoding/json"
	"fmt"
)

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
