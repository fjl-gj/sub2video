package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sbu2video/backend/internal/crypto"
)

// ProviderTask is the dispatch/poll view of a generation task.
type ProviderTask struct {
	ID                 int64
	PublicID           string
	UserID             int64
	Status             string
	Progress           int
	DurationSec        int
	Resolution         string
	AspectRatio        string
	Prompt             string
	SourceType         string
	SourceRef          string
	ClientWorkflow     string // derived from workflow label / stored later
	WorkflowLabel      string
	CarrierCode        string
	ModelCode          string
	SupplierCode       string
	SupplierRouteID    string
	UpstreamWorkflowID string
	UpstreamTaskID     string
	UpstreamStatus     string
	ResultURL          string
	ResultOSSURL       string
	FailReason         string
	CreatedAt          time.Time
}

func (s *Store) ResolveSupplierToken(ctx context.Context, secretKey, supplierCode, envFallback string) (string, error) {
	sp, err := s.GetSupplierByCode(ctx, supplierCode)
	if err != nil {
		if envFallback != "" {
			return envFallback, nil
		}
		return "", err
	}
	if sp.TokenEncrypted != "" {
		plain, err := crypto.DecryptAESGCM(secretKey, sp.TokenEncrypted)
		if err != nil {
			return "", err
		}
		if plain != "" {
			return plain, nil
		}
	}
	return envFallback, nil
}

func (s *Store) ListPendingProviderTasks(ctx context.Context, limit int) ([]ProviderTask, error) {
	if limit < 1 {
		limit = 20
	}
	rows, err := s.Pool.Query(ctx, `
SELECT id, public_id, user_id, status, progress, duration_sec, resolution, COALESCE(aspect_ratio,''),
       COALESCE(prompt,''), COALESCE(source_type,''), COALESCE(source_ref,''), workflow,
       carrier_code, model_code, supplier_code, supplier_route_id, upstream_workflow_id,
       upstream_task_id, upstream_status, result_url, result_oss_url, fail_reason, created_at
FROM generation_tasks
WHERE status IN ('queued', 'running')
ORDER BY created_at ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ProviderTask
	for rows.Next() {
		var t ProviderTask
		if err := rows.Scan(
			&t.ID, &t.PublicID, &t.UserID, &t.Status, &t.Progress, &t.DurationSec, &t.Resolution, &t.AspectRatio,
			&t.Prompt, &t.SourceType, &t.SourceRef, &t.WorkflowLabel,
			&t.CarrierCode, &t.ModelCode, &t.SupplierCode, &t.SupplierRouteID, &t.UpstreamWorkflowID,
			&t.UpstreamTaskID, &t.UpstreamStatus, &t.ResultURL, &t.ResultOSSURL, &t.FailReason, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		t.ClientWorkflow = workflowIDFromLabel(t.WorkflowLabel)
		items = append(items, t)
	}
	return items, rows.Err()
}

func workflowIDFromLabel(label string) string {
	switch label {
	case "H3 文生视频":
		return "wf-h3-text"
	case "H3 多图生视频":
		return "wf-h3-multi"
	default:
		return "wf-h3-image"
	}
}

type RouteAttach struct {
	CarrierCode        string
	ModelCode          string
	SupplierCode       string
	SupplierRouteID    string
	UpstreamWorkflowID string
	Snapshot           map[string]any
}

func (s *Store) AttachTaskRoute(ctx context.Context, taskID int64, a RouteAttach) error {
	raw, _ := json.Marshal(a.Snapshot)
	_, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  carrier_code=$1, model_code=$2, supplier_code=$3, supplier_route_id=$4,
  upstream_workflow_id=$5, route_snapshot=$6::jsonb, updated_at=NOW()
WHERE id=$7`, a.CarrierCode, a.ModelCode, a.SupplierCode, a.SupplierRouteID, a.UpstreamWorkflowID, string(raw), taskID)
	return err
}

func (s *Store) MarkTaskSubmitted(ctx context.Context, taskID int64, upstreamTaskID, upstreamStatus string) error {
	_, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  status='running', progress=GREATEST(progress, 8), upstream_task_id=$1, upstream_status=$2,
  started_at=COALESCE(started_at, NOW()), updated_at=NOW()
WHERE id=$3`, upstreamTaskID, upstreamStatus, taskID)
	return err
}

func (s *Store) UpdateTaskUpstream(ctx context.Context, taskID int64, upstreamStatus string, progress int) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 99 {
		progress = 99
	}
	_, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET status='running', upstream_status=$1, progress=$2, updated_at=NOW()
WHERE id=$3 AND status IN ('queued','running')`, upstreamStatus, progress, taskID)
	return err
}

func (s *Store) CompleteTaskUpstream(ctx context.Context, taskID int64, resultURL, ossURL string, upstreamDuration int) error {
	_, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  status='completed', progress=100, upstream_status='SUCCESS',
  result_url=$1, result_oss_url=$2, upstream_duration_sec=$3,
  thumb_url=CASE WHEN $1 <> '' THEN $1 ELSE thumb_url END,
  completed_at=NOW(), updated_at=NOW()
WHERE id=$4`, resultURL, ossURL, upstreamDuration, taskID)
	return err
}

func (s *Store) FailTaskUpstream(ctx context.Context, taskID int64, reason string) error {
	_, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  status='failed', progress=0, upstream_status='FAILED', fail_reason=$1, updated_at=NOW()
WHERE id=$2`, reason, taskID)
	return err
}

func (s *Store) MarkTaskSimulated(ctx context.Context, taskID int64) error {
	_, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET supplier_code='simulator', upstream_status='SIMULATED', updated_at=NOW()
WHERE id=$1 AND supplier_code=''`, taskID)
	return err
}

func (s *Store) AdvanceSimulatedTasks(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `
SELECT id, status, progress FROM generation_tasks
WHERE status IN ('queued', 'running')
  AND (upstream_task_id = '' OR supplier_code = 'simulator' OR supplier_code = '')`)
	if err != nil {
		return err
	}
	type item struct {
		id       int64
		status   string
		progress int
	}
	var batch []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.status, &it.progress); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, it)
	}
	rows.Close()

	for _, it := range batch {
		_ = s.MarkTaskSimulated(ctx, it.id)
		switch it.status {
		case "queued":
			_, err = s.Pool.Exec(ctx, `UPDATE generation_tasks SET status='running', progress=5, updated_at=NOW() WHERE id=$1 AND upstream_task_id=''`, it.id)
		case "running":
			next := it.progress + 12
			if next >= 100 {
				_, err = s.Pool.Exec(ctx, `UPDATE generation_tasks SET status='completed', progress=100, completed_at=NOW(), updated_at=NOW() WHERE id=$1 AND upstream_task_id=''`, it.id)
			} else {
				_, err = s.Pool.Exec(ctx, `UPDATE generation_tasks SET progress=$1, updated_at=NOW() WHERE id=$2 AND upstream_task_id=''`, next, it.id)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
