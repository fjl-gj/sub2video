package store

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
	SourceMeta         SourceMeta
	ClientWorkflow     string // derived from workflow label / stored later
	WorkflowLabel      string
	CarrierCode        string
	ModelCode          string
	SupplierCode       string
	SupplierRouteID    string
	ExecutionType      string
	APIModelCode       string
	APIVersion         string
	WorkflowCode       string
	UpstreamWorkflowID string
	QuotePublicID      string
	GenerateAudio      bool
	ParameterMapping   map[string]any
	AdapterCode        string
	UpstreamTaskID     string
	UpstreamStatus     string
	ResultURL          string
	ResultOSSURL       string
	FailReason         string
	ClaimedAt          *time.Time
	ClaimExpiresAt     *time.Time
	ClaimToken         string
	SubmitStarted      bool
	SubmitStartedAt    *time.Time
	SubmitUncertain    bool
	ExecutionDeadline  *time.Time
	PollAttempts       int
	NextPollAt         *time.Time
	ProviderRequestID  string
	CreatedAt          time.Time
}

func (s *Store) ResolveSupplierToken(ctx context.Context, secretKey, supplierCode, envFallback string) (string, error) {
	sp, err := s.GetSupplierByCode(ctx, supplierCode)
	if err != nil {
		if strings.EqualFold(strings.TrimSpace(supplierCode), "autodl") && envFallback != "" {
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
	if strings.EqualFold(strings.TrimSpace(supplierCode), "autodl") {
		return envFallback, nil
	}
	return "", nil
}

func (s *Store) ListPendingProviderTasks(ctx context.Context, limit int) ([]ProviderTask, error) {
	if limit < 1 {
		limit = 20
	}
	rows, err := s.Pool.Query(ctx, `
SELECT id, public_id, user_id, status, progress, duration_sec, resolution, COALESCE(aspect_ratio,''),
       COALESCE(prompt,''), COALESCE(source_type,''), COALESCE(source_ref,''), COALESCE(source_meta,'{}'::jsonb), workflow,
	       COALESCE(NULLIF(workflow_public_id,''), ''), carrier_code, model_code, supplier_code, supplier_route_id, upstream_workflow_id,
	       quote_public_id, generate_audio, offering_snapshot, execution_type, execution_snapshot,
       upstream_task_id, upstream_status, result_url, result_oss_url, fail_reason,
       claimed_at, claim_expires_at, claim_token, submit_started, submit_started_at, submit_uncertain,
       execution_deadline, poll_attempts, next_poll_at,
       provider_request_id, created_at
FROM generation_tasks
WHERE execution_mode='provider'
  AND (
    (status='queued' AND NOT submit_uncertain AND (next_poll_at IS NULL OR next_poll_at <= NOW()))
    OR (status='running' AND upstream_task_id<>'' AND NOT submit_uncertain
        AND (next_poll_at IS NULL OR next_poll_at <= NOW()))
    OR (status='running' AND upstream_task_id='' AND upstream_status='SUBMITTING'
        AND NOT submit_started AND NOT submit_uncertain
        AND (next_poll_at IS NULL OR next_poll_at <= NOW()))
  )
ORDER BY created_at ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ProviderTask
	for rows.Next() {
		var t ProviderTask
		var wfPublic string
		var metaRaw, offeringRaw, executionRaw []byte
		if err := rows.Scan(
			&t.ID, &t.PublicID, &t.UserID, &t.Status, &t.Progress, &t.DurationSec, &t.Resolution, &t.AspectRatio,
			&t.Prompt, &t.SourceType, &t.SourceRef, &metaRaw, &t.WorkflowLabel, &wfPublic,
			&t.CarrierCode, &t.ModelCode, &t.SupplierCode, &t.SupplierRouteID, &t.UpstreamWorkflowID,
			&t.QuotePublicID, &t.GenerateAudio, &offeringRaw, &t.ExecutionType, &executionRaw,
			&t.UpstreamTaskID, &t.UpstreamStatus, &t.ResultURL, &t.ResultOSSURL, &t.FailReason,
			&t.ClaimedAt, &t.ClaimExpiresAt, &t.ClaimToken, &t.SubmitStarted, &t.SubmitStartedAt, &t.SubmitUncertain,
			&t.ExecutionDeadline, &t.PollAttempts, &t.NextPollAt,
			&t.ProviderRequestID, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metaRaw, &t.SourceMeta)
		if len(offeringRaw) > 0 {
			var offeringSnapshot struct {
				AdapterCode string `json:"adapterCode"`
			}
			_ = json.Unmarshal(offeringRaw, &offeringSnapshot)
			t.AdapterCode = offeringSnapshot.AdapterCode
		}
		if len(executionRaw) > 0 {
			var snapshot struct {
				APIModelCode     string         `json:"apiModelCode"`
				APIVersion       string         `json:"apiVersion"`
				WorkflowCode     string         `json:"workflowCode"`
				ExecutionType    string         `json:"executionType"`
				AdapterCode      string         `json:"adapterCode"`
				ParameterMapping map[string]any `json:"parameterMapping"`
			}
			_ = json.Unmarshal(executionRaw, &snapshot)
			if t.ExecutionType == "" {
				t.ExecutionType = snapshot.ExecutionType
			}
			t.APIModelCode, t.APIVersion, t.WorkflowCode = snapshot.APIModelCode, snapshot.APIVersion, snapshot.WorkflowCode
			t.ParameterMapping = snapshot.ParameterMapping
			if t.AdapterCode == "" {
				t.AdapterCode = snapshot.AdapterCode
			}
		}
		if wfPublic != "" {
			t.ClientWorkflow = wfPublic
		} else {
			t.ClientWorkflow = workflowIDFromLabel(t.WorkflowLabel)
		}
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

// ClaimTaskForDispatch gives exactly one worker the right to submit a provider
// task. A stale SUBMITTING lease is reclaimable after a worker crash. The
// external call still uses the immutable execution snapshot on the task.
func (s *Store) ClaimTaskForDispatch(ctx context.Context, taskID int64) (bool, error) {
	claimed, _, err := s.ClaimTaskForDispatchWithToken(ctx, taskID)
	return claimed, err
}

// ClaimTaskForDispatchWithToken claims a task before any provider request is
// sent. A stale claim is reclaimable only while submit_started is false. Once
// the request may have crossed the provider boundary, it is never retried
// automatically because the response might have been lost.
func (s *Store) ClaimTaskForDispatchWithToken(ctx context.Context, taskID int64) (bool, string, error) {
	claimToken := NewPublicID()
	var returnedToken string
	err := s.Pool.QueryRow(ctx, `
UPDATE generation_tasks SET
  status='running', progress=GREATEST(progress, 1), upstream_status='SUBMITTING',
  started_at=COALESCE(started_at, NOW()),
  claim_token=$2,
  claimed_at=NOW(), claim_expires_at=NOW()+INTERVAL '5 minutes',
  next_poll_at=NOW()+INTERVAL '5 minutes',
  execution_deadline=COALESCE(execution_deadline, NOW()+INTERVAL '30 minutes'),
  submit_started=FALSE, submit_started_at=NULL, submit_uncertain=FALSE, updated_at=NOW()
WHERE id=$1 AND execution_mode='provider' AND upstream_task_id='' AND NOT submit_uncertain
  AND (
    (status='queued' AND (next_poll_at IS NULL OR next_poll_at <= NOW()))
    OR (status='running' AND upstream_status='SUBMITTING' AND submit_started=FALSE
        AND (claim_expires_at IS NULL OR claim_expires_at < NOW()))
  )
RETURNING claim_token`, taskID, claimToken).Scan(&returnedToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, returnedToken, nil
}

// ClaimTaskForPoll serializes polling across worker instances. A stale lease
// is reclaimed so a crashed worker cannot leave a task permanently running.
func (s *Store) ClaimTaskForPoll(ctx context.Context, taskID int64) (bool, error) {
	claimed, _, err := s.ClaimTaskForPollWithToken(ctx, taskID)
	return claimed, err
}

func (s *Store) ClaimTaskForPollWithToken(ctx context.Context, taskID int64) (bool, string, error) {
	claimToken := NewPublicID()
	var returnedToken string
	err := s.Pool.QueryRow(ctx, `
UPDATE generation_tasks SET
  claim_token=$2,
  claimed_at=NOW(), claim_expires_at=NOW()+INTERVAL '2 minutes',
  execution_deadline=COALESCE(execution_deadline, NOW()+INTERVAL '30 minutes'),
  updated_at=NOW()
WHERE id=$1 AND status='running' AND execution_mode='provider' AND NOT submit_uncertain
  AND upstream_task_id<>''
	AND (claimed_at IS NULL OR claim_expires_at IS NULL OR claim_expires_at < NOW())
RETURNING claim_token`, taskID, claimToken).Scan(&returnedToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, returnedToken, nil
}

// MarkTaskSubmitStarted is the commit point before crossing the provider
// boundary. After this flag is set, a worker crash must not blindly resubmit.
func (s *Store) MarkTaskSubmitStarted(ctx context.Context, taskID int64, claimToken string) error {
	var executionType string
	err := s.Pool.QueryRow(ctx, `
UPDATE generation_tasks SET
  submit_started=TRUE, submit_started_at=NOW(), submit_uncertain=FALSE, updated_at=NOW()
WHERE id=$1 AND status='running' AND execution_mode='provider'
  AND upstream_task_id='' AND upstream_status='SUBMITTING'
  AND NOT submit_started AND NOT submit_uncertain
  AND claim_expires_at > NOW()
  AND ($2='' OR claim_token=$2)
RETURNING execution_type`, taskID, claimToken).Scan(&executionType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := s.Pool.Exec(ctx, `INSERT INTO provider_task_events (task_id,event_type,execution_type,upstream_status,detail,elapsed_ms)
VALUES ($1,'submit_started',$2,'SUBMITTING','{}'::jsonb,0)`, taskID, executionType); err != nil {
		// Best-effort telemetry; the submit fence itself is already committed.
		log.Printf("provider event submit_started task=%d: %v", taskID, err)
	}
	return nil
}

// MarkTaskSubmissionUnknown fences a submission whose provider response was
// lost or failed after the request may have been sent. It intentionally keeps
// the task running and does not refund; an operator must reconcile provider
// state before deciding whether to complete or cancel it.
func (s *Store) MarkTaskSubmissionUnknown(ctx context.Context, taskID int64, reason, claimToken string) error {
	return s.MarkTaskSubmissionUnknownWithRequestID(ctx, taskID, reason, claimToken, "")
}

func (s *Store) MarkTaskSubmissionUnknownWithRequestID(ctx context.Context, taskID int64, reason, claimToken, providerRequestID string) error {
	tag, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  status='running', upstream_status='SUBMIT_UNKNOWN', submit_uncertain=TRUE,
  provider_request_id=COALESCE(NULLIF($4,''),provider_request_id),
  fail_reason=$1, claim_token='', claimed_at=NULL, claim_expires_at=NULL, next_poll_at=NULL, updated_at=NOW()
WHERE id=$2 AND status='running' AND execution_mode='provider'
  AND upstream_task_id='' AND submit_started=TRUE
  AND ($3='' OR claim_token=$3)`, reason, taskID, claimToken, providerRequestID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_ = s.RecordProviderEvent(ctx, taskID, "submit_unknown", "", "", "SUBMIT_UNKNOWN", map[string]any{"reason": reason}, 0)
	return nil
}

// RecoverExpiredProviderSubmissions makes a lost in-flight request visible for
// reconciliation after a crash or unavailable database. It never resubmits or
// refunds. Clearing the token fences a late response from the old worker.
func (s *Store) RecoverExpiredProviderSubmissions(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `
WITH recovered AS (
  UPDATE generation_tasks SET upstream_status='SUBMIT_UNKNOWN', submit_uncertain=TRUE,
    fail_reason='提交响应待核对：Worker 提交租约已过期',
    claimed_at=NULL, claim_expires_at=NULL, claim_token='', next_poll_at=NULL, updated_at=NOW()
  WHERE execution_mode='provider' AND status='running' AND upstream_status='SUBMITTING'
    AND upstream_task_id='' AND submit_started AND NOT submit_uncertain
    AND (claim_expires_at < NOW() OR (claim_expires_at IS NULL AND COALESCE(submit_started_at,updated_at,created_at) < NOW()-INTERVAL '5 minutes'))
  RETURNING id, execution_type, provider_request_id
)
INSERT INTO provider_task_events (task_id,event_type,execution_type,provider_request_id,upstream_status,detail)
SELECT id,'submit_unknown',execution_type,provider_request_id,'SUBMIT_UNKNOWN',
  '{"reason":"expired_submit_lease"}'::jsonb FROM recovered`)
	return err
}

func (s *Store) MarkTaskSubmitted(ctx context.Context, taskID int64, upstreamTaskID, upstreamStatus string) error {
	return s.MarkTaskSubmittedWithToken(ctx, taskID, upstreamTaskID, upstreamStatus, "")
}

func (s *Store) MarkTaskSubmittedWithToken(ctx context.Context, taskID int64, upstreamTaskID, upstreamStatus, claimToken string) error {
	var executionType string
	var elapsedMs int64
	err := s.Pool.QueryRow(ctx, `
WITH claimed AS (
  SELECT id, claimed_at FROM generation_tasks WHERE id=$3 FOR UPDATE
)
UPDATE generation_tasks task SET
  status='running', progress=GREATEST(progress, 8), upstream_task_id=$1, upstream_status=$2,
  provider_request_id=$1, started_at=COALESCE(started_at, NOW()),
  claimed_at=NULL, claim_expires_at=NULL,
  next_poll_at=NOW()+INTERVAL '8 seconds', poll_attempts=0,
  submit_started=TRUE, submit_uncertain=FALSE,
  execution_deadline=COALESCE(execution_deadline, NOW()+INTERVAL '30 minutes'), updated_at=NOW()
FROM claimed
WHERE task.id=claimed.id AND task.status='running' AND task.execution_mode='provider'
  AND task.upstream_task_id='' AND task.upstream_status='SUBMITTING'
  AND ($4='' OR (task.submit_started=TRUE AND task.claim_token=$4))
RETURNING task.execution_type, COALESCE(EXTRACT(EPOCH FROM (NOW()-claimed.claimed_at))*1000,0)::bigint`, upstreamTaskID, upstreamStatus, taskID, claimToken).Scan(&executionType, &elapsedMs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := s.Pool.Exec(ctx, `INSERT INTO provider_task_events (task_id,event_type,execution_type,provider_request_id,upstream_status,detail,elapsed_ms)
VALUES ($1,'submitted',$2,$3,$4,'{}'::jsonb,$5)`, taskID, executionType, upstreamTaskID, upstreamStatus, elapsedMs); err != nil {
		// Best-effort telemetry: the upstream task is already recorded, so
		// log the miss instead of failing a submission that succeeded.
		log.Printf("provider event submitted task=%d: %v", taskID, err)
	}
	return nil
}

// RecordProviderEvent appends internal execution telemetry. It is deliberately
// separate from client task responses so provider details remain admin-only.
func (s *Store) RecordProviderEvent(ctx context.Context, taskID int64, eventType, executionType, providerRequestID, upstreamStatus string, detail map[string]any, elapsedMs int64) error {
	if detail == nil {
		detail = map[string]any{}
	}
	raw, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
INSERT INTO provider_task_events (task_id,event_type,execution_type,provider_request_id,upstream_status,detail,elapsed_ms)
VALUES ($1,$2,
  COALESCE(NULLIF($3,''),(SELECT execution_type FROM generation_tasks WHERE id=$1)),
  COALESCE(NULLIF($4,''),(SELECT provider_request_id FROM generation_tasks WHERE id=$1)),
  $5,$6::jsonb,$7)`, taskID, eventType, executionType, providerRequestID, upstreamStatus, string(raw), elapsedMs)
	return err
}

// RecordProviderPollFailure releases the lease and schedules a bounded retry.
// The task remains running until its execution deadline, allowing transient
// supplier/network failures to recover without a second charge.
func (s *Store) RecordProviderPollFailure(ctx context.Context, taskID int64, reason string, retryAfter time.Duration) error {
	return s.RecordProviderPollFailureWithToken(ctx, taskID, reason, retryAfter, "")
}

func (s *Store) RecordProviderPollFailureWithToken(ctx context.Context, taskID int64, reason string, retryAfter time.Duration, claimToken string) error {
	if retryAfter < 2*time.Second {
		retryAfter = 2 * time.Second
	}
	if retryAfter > 2*time.Minute {
		retryAfter = 2 * time.Minute
	}
	tag, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  claimed_at=NULL, claim_expires_at=NULL, poll_attempts=poll_attempts+1,
  next_poll_at=NOW()+make_interval(secs => $1::double precision), fail_reason=$2, updated_at=NOW()
WHERE id=$3 AND status='running' AND execution_mode='provider' AND upstream_task_id<>''
  AND ($4='' OR claim_token=$4)`, retryAfter.Seconds(), reason, taskID, claimToken)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return s.RecordProviderEvent(ctx, taskID, "poll_error", "", "", "", map[string]any{"reason": reason}, retryAfter.Milliseconds())
}

// MarkTaskWaitingForConfiguration is intentionally non-terminal: once an
// administrator configures the locked supplier or object storage, the worker
// can submit the same immutable task snapshot on its next poll. Submitted
// work stays running so it is polled rather than submitted a second time.
func (s *Store) MarkTaskWaitingForConfiguration(ctx context.Context, taskID int64, reason string) error {
	return s.MarkTaskWaitingForConfigurationWithToken(ctx, taskID, reason, "")
}

func (s *Store) MarkTaskWaitingForConfigurationWithToken(ctx context.Context, taskID int64, reason, claimToken string) error {
	tag, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  status=CASE WHEN upstream_task_id='' THEN 'queued' ELSE status END,
  upstream_status='WAITING_CONFIGURATION', fail_reason=$1,
  claimed_at=NULL, claim_expires_at=NULL, next_poll_at=NOW()+INTERVAL '30 seconds', updated_at=NOW()
WHERE id=$2 AND status IN ('queued','running') AND execution_mode='provider'
  AND NOT submit_uncertain AND (upstream_task_id<>'' OR NOT submit_started)
  AND ($3='' OR claim_token=$3)`, reason, taskID, claimToken)
	if err == nil {
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		_ = s.RecordProviderEvent(ctx, taskID, "configuration_wait", "", "", "WAITING_CONFIGURATION", map[string]any{"reason": reason}, 0)
	}
	return err
}

// MarkTaskDeliveryPending keeps a provider task executable after its upstream
// work has completed but the durable object-storage handoff is unavailable.
// The worker retries from this immutable task snapshot; no provider fallback
// or simulator completion is allowed.
func (s *Store) MarkTaskDeliveryPending(ctx context.Context, taskID int64, resultURL, reason string) error {
	return s.MarkTaskDeliveryPendingWithToken(ctx, taskID, resultURL, reason, "")
}

func (s *Store) MarkTaskDeliveryPendingWithToken(ctx context.Context, taskID int64, resultURL, reason, claimToken string) error {
	tag, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  status='running', progress=GREATEST(progress, 95), upstream_status='WAITING_STORAGE',
  result_url=$1, fail_reason=$2, claimed_at=NULL, claim_expires_at=NULL,
  next_poll_at=NOW()+INTERVAL '30 seconds', updated_at=NOW()
WHERE id=$3 AND status IN ('queued','running')
  AND ($4='' OR claim_token=$4)`, resultURL, reason, taskID, claimToken)
	if err == nil {
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		_ = s.RecordProviderEvent(ctx, taskID, "delivery_wait", "", "", "WAITING_STORAGE", map[string]any{"reason": reason}, 0)
	}
	return err
}

func (s *Store) UpdateTaskUpstream(ctx context.Context, taskID int64, upstreamStatus string, progress int) error {
	return s.UpdateTaskUpstreamWithRetry(ctx, taskID, upstreamStatus, progress, 12*time.Second)
}

func (s *Store) UpdateTaskUpstreamWithRetry(ctx context.Context, taskID int64, upstreamStatus string, progress int, retryAfter time.Duration) error {
	return s.UpdateTaskUpstreamWithRetryAndToken(ctx, taskID, upstreamStatus, progress, retryAfter, "")
}

func (s *Store) UpdateTaskUpstreamWithRetryAndToken(ctx context.Context, taskID int64, upstreamStatus string, progress int, retryAfter time.Duration, claimToken string) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 99 {
		progress = 99
	}
	if retryAfter < 2*time.Second {
		retryAfter = 2 * time.Second
	}
	tag, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET status='running', upstream_status=$1, progress=$2,
  claimed_at=NULL, claim_expires_at=NULL, poll_attempts=poll_attempts+1,
  next_poll_at=NOW()+make_interval(secs => $3::double precision), updated_at=NOW()
WHERE id=$4 AND status IN ('queued','running')
  AND ($5='' OR claim_token=$5)`, upstreamStatus, progress, retryAfter.Seconds(), taskID, claimToken)
	if err == nil {
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		// Keep this best-effort: a telemetry insert must not strand a task.
		_ = s.RecordProviderEvent(ctx, taskID, "poll", "", "", upstreamStatus, map[string]any{"progress": progress}, retryAfter.Milliseconds())
	}
	return err
}

func (s *Store) CompleteTaskUpstream(ctx context.Context, taskID int64, resultURL, ossURL string, upstreamDuration int) error {
	return s.CompleteTaskUpstreamWithToken(ctx, taskID, resultURL, ossURL, upstreamDuration, "")
}

func (s *Store) CompleteTaskUpstreamWithToken(ctx context.Context, taskID int64, resultURL, ossURL string, upstreamDuration int, claimToken string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var offeringID, executionType, providerRequestID string
	var elapsedMs int64
	err = tx.QueryRow(ctx, `
UPDATE generation_tasks SET
  status='completed', progress=100, upstream_status='SUCCESS',
  result_url=$1, result_oss_url=$2, upstream_duration_sec=$3,
  thumb_url=CASE WHEN $2 <> '' THEN $2 ELSE thumb_url END,
  completed_at=NOW(), claimed_at=NULL, claim_expires_at=NULL, next_poll_at=NULL,
  updated_at=NOW()
WHERE id=$4 AND status IN ('queued','running') AND execution_mode='provider'
  AND ($5='' OR claim_token=$5)
RETURNING supplier_route_id, execution_type, provider_request_id,
  COALESCE(EXTRACT(EPOCH FROM (NOW()-COALESCE(started_at, created_at)))*1000,0)::bigint`, resultURL, ossURL, upstreamDuration, taskID, claimToken).Scan(&offeringID, &executionType, &providerRequestID, &elapsedMs)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if offeringID != "" {
		if _, err = tx.Exec(ctx, `
UPDATE supplier_offerings SET failure_count=0, last_success_at=NOW(),
  health_status=CASE WHEN health_status='unavailable' THEN health_status ELSE 'healthy' END,
  updated_at=NOW()
WHERE public_id=$1`, offeringID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE supplier_executions execution SET failure_count=0, last_success_at=NOW(), health_status=CASE WHEN health_status='unavailable' THEN health_status ELSE 'healthy' END, updated_at=NOW()
WHERE execution.public_id=$1 OR execution.legacy_offering_id=(SELECT offering.id FROM supplier_offerings offering WHERE offering.public_id=$1)`, offeringID); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO provider_task_events (task_id,event_type,execution_type,provider_request_id,upstream_status,detail,elapsed_ms)
VALUES ($1,'completed',$2,$3,'SUCCESS',$4::jsonb,$5)`, taskID, executionType, providerRequestID, `{"resultUrl":true}`, elapsedMs); err != nil {
		// The completion event is part of the audit trail; if it cannot be
		// written, the completion must not commit silently.
		return err
	}
	return tx.Commit(ctx)
}

func providerFailureCountsHealth(reason string) bool {
	for _, marker := range []string{"超时", "用户终止", "取消", "对象存储", "存储", "不支持", "适配器", "未验证"} {
		if strings.Contains(reason, marker) {
			return false
		}
	}
	return true
}

func (s *Store) FailTaskUpstream(ctx context.Context, taskID int64, reason string) error {
	return s.FailTaskUpstreamWithToken(ctx, taskID, reason, "")
}

func (s *Store) FailTaskUpstreamWithToken(ctx context.Context, taskID int64, reason, claimToken string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := failTaskUpstreamTx(ctx, tx, taskID, reason, claimToken); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// failTaskUpstreamTx is shared by the worker and the audited reconciliation
// transaction so refunds and the administrator's evidence commit together.
func failTaskUpstreamTx(ctx context.Context, tx pgx.Tx, taskID int64, reason, claimToken string) error {
	var userID int64
	var publicID, status, title, offeringID, executionType, providerRequestID, currentClaimToken string
	var cost float64
	var elapsedMs int64
	err := tx.QueryRow(ctx, `SELECT user_id, public_id, status, title, cost::float8, supplier_route_id, execution_type, provider_request_id, claim_token,
COALESCE(EXTRACT(EPOCH FROM (NOW()-COALESCE(started_at, created_at)))*1000,0)::bigint
FROM generation_tasks WHERE id=$1 FOR UPDATE`, taskID).Scan(&userID, &publicID, &status, &title, &cost, &offeringID, &executionType, &providerRequestID, &currentClaimToken, &elapsedMs)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if claimToken != "" && currentClaimToken != claimToken {
		return ErrNotFound
	}
	if status != "queued" && status != "running" {
		return nil
	}
	if _, err = tx.Exec(ctx, `UPDATE generation_tasks SET status='failed', progress=0, upstream_status='FAILED', fail_reason=$1, claimed_at=NULL, claim_expires_at=NULL, next_poll_at=NULL, updated_at=NOW() WHERE id=$2`, reason, taskID); err != nil {
		return err
	}
	var alreadyRefunded bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM wallet_ledger WHERE ref_type='task' AND ref_id=$1 AND type='task_refund')`, publicID).Scan(&alreadyRefunded); err != nil {
		return err
	}
	if !alreadyRefunded {
		if _, err = tx.Exec(ctx, `UPDATE wallets SET balance=balance+$1, version=version+1, updated_at=NOW() WHERE user_id=$2`, cost, userID); err != nil {
			return err
		}
		var balance float64
		if err = tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id=$1`, userID).Scan(&balance); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO wallet_ledger (public_id, user_id, type, amount, balance_after, ref_type, ref_id, description) VALUES ($1,$2,'task_refund',$3,$4,'task',$5,$6)`, NewLedgerID(), userID, cost, balance, publicID, publicID+" 上游失败退款"); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE consumption_orders SET status='refunded', refunded_at=NOW() WHERE task_id=$1 AND status='paid'`, taskID); err != nil {
			return err
		}
	}
	if offeringID != "" && providerFailureCountsHealth(reason) {
		if _, err = tx.Exec(ctx, `
UPDATE supplier_offerings SET
		failure_count=failure_count+1,
  last_failure_at=NOW(),
  health_status=CASE WHEN failure_count+1 >= 3 THEN 'unavailable' ELSE 'degraded' END,
  enabled=CASE WHEN failure_count+1 >= 3 THEN FALSE ELSE enabled END,
  updated_at=NOW()
		WHERE public_id=$1`, offeringID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE supplier_executions execution SET failure_count=failure_count+1, last_failure_at=NOW(), health_status=CASE WHEN failure_count+1 >= 3 THEN 'unavailable' ELSE 'degraded' END, enabled=CASE WHEN failure_count+1 >= 3 THEN FALSE ELSE enabled END, updated_at=NOW()
WHERE execution.public_id=$1 OR execution.legacy_offering_id=(SELECT offering.id FROM supplier_offerings offering WHERE offering.public_id=$1)`, offeringID); err != nil {
			return err
		}
	}
	failureDetail, _ := json.Marshal(map[string]any{"reason": reason})
	if _, err = tx.Exec(ctx, `INSERT INTO provider_task_events (task_id,event_type,execution_type,provider_request_id,upstream_status,detail,elapsed_ms)
VALUES ($1,'failed',$2,$3,'FAILED',$4::jsonb,$5)`, taskID, executionType, providerRequestID, string(failureDetail), elapsedMs); err != nil {
		return err
	}
	return nil
}

func (s *Store) MarkTaskSimulated(ctx context.Context, taskID int64) error {
	_, err := s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  execution_mode='simulated_legacy', supplier_code='simulator', upstream_status='SIMULATED', updated_at=NOW()
WHERE id=$1 AND supplier_code='' AND execution_mode='simulated_legacy'`, taskID)
	return err
}

func (s *Store) AdvanceSimulatedTasks(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `
SELECT id, status, progress FROM generation_tasks
WHERE status IN ('queued', 'running')
  AND execution_mode = 'simulated_legacy'
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
