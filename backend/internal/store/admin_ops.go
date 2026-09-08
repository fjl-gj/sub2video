package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Ticket struct {
	ID        string
	Title     string
	Category  string
	Status    string
	TaskRef   string
	Detail    string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserEmail string
	UserName  string
}

func ticketStatusCN(st string) string {
	switch st {
	case "open":
		return "待处理"
	case "processing":
		return "处理中"
	case "resolved":
		return "已解决"
	default:
		return st
	}
}

func ticketStatusDB(cn string) string {
	switch cn {
	case "待处理":
		return "open"
	case "处理中":
		return "processing"
	case "已解决":
		return "resolved"
	default:
		return cn
	}
}

func (s *Store) CreateTicket(ctx context.Context, userID int64, title, category, taskRef, detail string) (*Ticket, error) {
	if title == "" {
		return nil, errors.New("标题不能为空")
	}
	if category == "" {
		category = "售后支持"
	}
	if taskRef == "" {
		taskRef = "-"
	}
	id := DatedPublicID("TKT")
	_, err := s.Pool.Exec(ctx, `
INSERT INTO tickets (public_id, user_id, title, category, status, task_ref, detail)
VALUES ($1, $2, $3, $4, 'open', $5, $6)`, id, userID, title, category, taskRef, detail)
	if err != nil {
		return nil, err
	}
	return &Ticket{ID: id, Title: title, Category: category, Status: "待处理", TaskRef: taskRef, Detail: detail, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (s *Store) ListTickets(ctx context.Context, userID int64, page, pageSize int) ([]Ticket, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM tickets WHERE user_id=$1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.Pool.Query(ctx, `
SELECT public_id, title, category, status, task_ref, detail, created_at, updated_at
FROM tickets WHERE user_id=$1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []Ticket
	for rows.Next() {
		var t Ticket
		var st string
		if err := rows.Scan(&t.ID, &t.Title, &t.Category, &st, &t.TaskRef, &t.Detail, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		t.Status = ticketStatusCN(st)
		items = append(items, t)
	}
	return items, total, rows.Err()
}

func (s *Store) ListAdminTickets(ctx context.Context, page, pageSize int) ([]Ticket, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM tickets`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.Pool.Query(ctx, `
SELECT t.public_id, t.title, t.category, t.status, t.task_ref, t.detail, t.created_at, t.updated_at, u.email, u.display_name
FROM tickets t JOIN users u ON u.id = t.user_id
ORDER BY t.updated_at DESC LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []Ticket
	for rows.Next() {
		var t Ticket
		var st string
		if err := rows.Scan(&t.ID, &t.Title, &t.Category, &st, &t.TaskRef, &t.Detail, &t.CreatedAt, &t.UpdatedAt, &t.UserEmail, &t.UserName); err != nil {
			return nil, 0, err
		}
		t.Status = ticketStatusCN(st)
		items = append(items, t)
	}
	return items, total, rows.Err()
}

func (s *Store) UpdateTicketStatus(ctx context.Context, publicID, statusCN string) error {
	st := ticketStatusDB(statusCN)
	tag, err := s.Pool.Exec(ctx, `UPDATE tickets SET status=$1, updated_at=NOW() WHERE public_id=$2`, st, publicID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AdminCancelTask(ctx context.Context, taskPublicID string) (*Task, float64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)

	var id, userID int64
	var status, title string
	var cost float64
	err = tx.QueryRow(ctx, `
SELECT id, user_id, status, cost::float8, title FROM generation_tasks WHERE public_id=$1 FOR UPDATE`,
		taskPublicID).Scan(&id, &userID, &status, &cost, &title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, errors.New("任务不存在")
	}
	if err != nil {
		return nil, 0, err
	}
	if status != "queued" && status != "running" {
		return nil, 0, errors.New("该任务无法终止")
	}
	if _, err = tx.Exec(ctx, `
UPDATE generation_tasks SET status='failed', progress=0, upstream_status='CANCELLED_BY_ADMIN',
  fail_reason='管理员终止并退款', updated_at=NOW()
WHERE id=$1`, id); err != nil {
		return nil, 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance=balance+$1, version=version+1, updated_at=NOW() WHERE user_id=$2`, cost, userID); err != nil {
		return nil, 0, err
	}
	var balance float64
	if err = tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id=$1`, userID).Scan(&balance); err != nil {
		return nil, 0, err
	}
	ledgerID := NewLedgerID()
	if _, err = tx.Exec(ctx, `
INSERT INTO wallet_ledger (public_id, user_id, type, amount, balance_after, ref_type, ref_id, description)
VALUES ($1,$2,'task_refund',$3,$4,'task',$5,$6)`,
		ledgerID, userID, cost, balance, taskPublicID, taskPublicID+" 管理端终止退款"); err != nil {
		return nil, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return &Task{ID: taskPublicID, Title: title, Status: "失败", Cost: cost}, cost, nil
}

func (s *Store) AdminRetryTask(ctx context.Context, taskPublicID string) (*Task, error) {
	var status, quoteID string
	err := s.Pool.QueryRow(ctx, `SELECT status, quote_public_id FROM generation_tasks WHERE public_id=$1`, taskPublicID).Scan(&status, &quoteID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("任务不存在")
	}
	if err != nil {
		return nil, err
	}
	if status != "failed" {
		return nil, errors.New("仅失败任务可重试")
	}
	if quoteID != "" {
		return nil, errors.New("请为该任务创建新报价后再重试")
	}
	_, err = s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  status='queued', progress=0, fail_reason='',
  upstream_task_id='', upstream_status='', result_url='', result_oss_url='',
  supplier_code='', supplier_route_id='', upstream_workflow_id='',
  started_at=NULL, completed_at=NULL, updated_at=NOW()
WHERE public_id=$1`, taskPublicID)
	if err != nil {
		return nil, err
	}
	return &Task{ID: taskPublicID, Status: "排队中"}, nil
}

func (s *Store) GetAdminTaskDetail(ctx context.Context, taskPublicID string) (map[string]any, error) {
	var (
		title, workflow, modeID, modelID, quoteID, status, resolution, supplier, upWF, upTask, upStatus, fail string
		progress, duration                                                                                    int
		cost, pricePerSec, costPerSec                                                                         float64
		priceSnap, routeSnap, retailSnap, costSnap, offeringSnap, inputSnap                                   []byte
		generateAudio                                                                                         bool
		userEmail, userName                                                                                   string
		created, updated                                                                                      time.Time
		resultURL, ossURL                                                                                     string
	)
	err := s.Pool.QueryRow(ctx, `
SELECT t.title, t.workflow, t.status, t.progress, t.duration_sec, t.resolution,
       t.cost::float8, t.price_per_sec::float8, t.cost_per_sec::float8,
       t.supplier_code, t.upstream_workflow_id, t.upstream_task_id, t.upstream_status, t.fail_reason,
       t.price_snapshot, t.route_snapshot, t.mode_public_id, t.model_public_id, t.quote_public_id, t.generate_audio,
       t.retail_snapshot, t.cost_snapshot, t.offering_snapshot, t.input_snapshot, t.result_url, t.result_oss_url,
       u.email, u.display_name, t.created_at, t.updated_at
FROM generation_tasks t JOIN users u ON u.id=t.user_id
WHERE t.public_id=$1`, taskPublicID).Scan(
		&title, &workflow, &status, &progress, &duration, &resolution,
		&cost, &pricePerSec, &costPerSec,
		&supplier, &upWF, &upTask, &upStatus, &fail,
		&priceSnap, &routeSnap, &modeID, &modelID, &quoteID, &generateAudio,
		&retailSnap, &costSnap, &offeringSnap, &inputSnap, &resultURL, &ossURL,
		&userEmail, &userName, &created, &updated,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id": taskPublicID, "title": title, "mode": workflow, "modeId": modeID,
		"modelId": modelID, "quoteId": quoteID, "generateAudio": generateAudio,
		"status": statusCN(status), "progress": progress, "durationSec": duration, "resolution": resolution,
		"cost": cost, "pricePerSec": pricePerSec, "costPerSec": costPerSec,
		"supplierCode": supplier, "upstreamWorkflowId": upWF, "upstreamTaskId": upTask, "upstreamStatus": upStatus,
		"failReason": fail, "resultUrl": resultURL, "resultOssUrl": ossURL,
		"priceSnapshot": jsonRawOrObj(priceSnap), "routeSnapshot": jsonRawOrObj(routeSnap),
		"retailSnapshot": jsonRawOrObj(retailSnap), "costSnapshot": jsonRawOrObj(costSnap),
		"offeringSnapshot": jsonRawOrObj(offeringSnap), "inputSnapshot": jsonRawOrObj(inputSnap),
		"user": userName, "userEmail": userEmail,
		"createdAt": created, "updatedAt": updated,
	}, nil
}

func jsonRawOrObj(raw []byte) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	return json.RawMessage(raw)
}
