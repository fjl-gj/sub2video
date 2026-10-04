package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var (
	ErrReconciliationInput     = errors.New("请确认核对结果并填写 10–1000 字的核对依据")
	ErrReconciliationConflict  = errors.New("任务状态已变化，或核对结果与已保存的上游任务不一致，请刷新核对")
	ErrReconciliationForbidden = errors.New("仅管理员可核对任务")
)

type TaskReconciliationInput struct {
	Action         string `json:"action"`
	UpstreamTaskID string `json:"upstreamTaskId"`
	Evidence       string `json:"evidence"`
	Confirmed      bool   `json:"confirmed"`
}

func (in *TaskReconciliationInput) Validate() error {
	in.Action = strings.TrimSpace(in.Action)
	in.UpstreamTaskID = strings.TrimSpace(in.UpstreamTaskID)
	in.Evidence = strings.TrimSpace(in.Evidence)
	if !in.Confirmed || utf8.RuneCountInString(in.Evidence) < 10 || utf8.RuneCountInString(in.Evidence) > 1000 {
		return ErrReconciliationInput
	}
	if in.Action != "resume" && in.Action != "refund" {
		return ErrReconciliationInput
	}
	if in.Action == "refund" && in.UpstreamTaskID != "" {
		return ErrReconciliationInput
	}
	if in.Action == "resume" {
		if len(in.UpstreamTaskID) == 0 || len(in.UpstreamTaskID) > 256 {
			return ErrReconciliationInput
		}
		for _, r := range in.UpstreamTaskID {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
				return ErrReconciliationInput
			}
		}
	}
	return nil
}

type TaskReconciliationResult struct {
	Task            map[string]any `json:"task"`
	Refund          float64        `json:"refund"`
	AlreadyResolved bool           `json:"alreadyResolved"`
}

// ReconcileTaskSubmission requires operator evidence. The handler verifies a
// resume ID exists at the locked supplier before invoking this transaction.
// There is intentionally no "retry submission" action.
func (s *Store) ReconcileTaskSubmission(ctx context.Context, actorID int64, taskPublicID string, in TaskReconciliationInput) (*TaskReconciliationResult, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var admin bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='admin')`, actorID).Scan(&admin); err != nil {
		return nil, err
	}
	if !admin {
		return nil, ErrReconciliationForbidden
	}
	var taskID int64
	var status, upstreamStatus, supplierCode, requestID string
	var uncertain bool
	var cost float64
	err = tx.QueryRow(ctx, `SELECT id,status,upstream_status,submit_uncertain,supplier_code,provider_request_id,cost::float8
FROM generation_tasks WHERE public_id=$1 AND execution_mode='provider' FOR UPDATE`, taskPublicID).
		Scan(&taskID, &status, &upstreamStatus, &uncertain, &supplierCode, &requestID, &cost)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var previous []byte
	err = tx.QueryRow(ctx, `SELECT details FROM admin_audit_logs WHERE target_type='task' AND target_id=$1 AND action='task.reconcile'
ORDER BY id DESC LIMIT 1`, taskPublicID).Scan(&previous)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if len(previous) > 0 {
		var last TaskReconciliationInput
		if err := json.Unmarshal(previous, &last); err != nil {
			return nil, err
		}
		if last.Action != in.Action || last.UpstreamTaskID != in.UpstreamTaskID {
			return nil, ErrReconciliationConflict
		}
		refund := float64(0)
		if in.Action == "refund" {
			refund = cost
		}
		return &TaskReconciliationResult{Task: map[string]any{"id": taskPublicID, "status": statusCN(status), "upstreamStatus": upstreamStatus}, Refund: refund, AlreadyResolved: true}, tx.Commit(ctx)
	}
	if status != "running" || upstreamStatus != "SUBMIT_UNKNOWN" || !uncertain {
		return nil, ErrReconciliationConflict
	}
	refund := float64(0)
	if in.Action == "resume" {
		if requestID != "" && requestID != in.UpstreamTaskID {
			return nil, ErrReconciliationConflict
		}
		// Serialize assignment of an upstream task across operators/local tasks.
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, supplierCode+":"+in.UpstreamTaskID); err != nil {
			return nil, err
		}
		var assigned bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM generation_tasks WHERE id<>$1 AND supplier_code=$2 AND (upstream_task_id=$3 OR provider_request_id=$3))`, taskID, supplierCode, in.UpstreamTaskID).Scan(&assigned); err != nil {
			return nil, err
		}
		if assigned {
			return nil, ErrReconciliationConflict
		}
		_, err = tx.Exec(ctx, `UPDATE generation_tasks SET upstream_task_id=$2,provider_request_id=$2,upstream_status='RECONCILED',
 submit_uncertain=FALSE,submit_started=TRUE,claim_token='',claimed_at=NULL,claim_expires_at=NULL,
 next_poll_at=NOW(),execution_deadline=NOW()+INTERVAL '30 minutes',fail_reason='',updated_at=NOW() WHERE id=$1`, taskID, in.UpstreamTaskID)
		upstreamStatus = "RECONCILED"
	} else {
		// An accepted task ID is positive evidence of acceptance. It cannot be
		// discarded by the "not accepted" refund action.
		if requestID != "" {
			return nil, ErrReconciliationConflict
		}
		err = failTaskUpstreamTx(ctx, tx, taskID, "管理员核对：供应商明确未受理，终止并退款", "")
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE generation_tasks SET submit_uncertain=FALSE,claim_token='',updated_at=NOW() WHERE id=$1`, taskID)
		}
		status, upstreamStatus, refund = "failed", "FAILED", cost
	}
	if err != nil {
		return nil, err
	}
	details, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO admin_audit_logs (public_id,actor_user_id,action,target_type,target_id,details)
 VALUES ($1,$2,'task.reconcile','task',$3,$4::jsonb)`, NewAuditID(), actorID, taskPublicID, string(details)); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO provider_task_events (task_id,event_type,execution_type,provider_request_id,upstream_status,detail)
 SELECT id,'reconciled',execution_type,provider_request_id,upstream_status,$2::jsonb FROM generation_tasks WHERE id=$1`, taskID, string(details)); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &TaskReconciliationResult{Task: map[string]any{"id": taskPublicID, "status": statusCN(status), "upstreamStatus": upstreamStatus}, Refund: refund}, nil
}
