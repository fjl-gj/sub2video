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

// normalizeAdminPagination keeps the management list contracts predictable.
// The legacy list methods still return the complete catalog; the paged
// variants below are used only when an admin endpoint explicitly receives
// page/pageSize query parameters.
func normalizeAdminPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func (s *Store) ListAdminSuppliersPage(ctx context.Context, page, pageSize int, keyword string) ([]Supplier, int, error) {
	page, pageSize = normalizeAdminPagination(page, pageSize)
	var total int
	const filter = `WHERE $1 = '' OR LOWER(public_id) LIKE '%' || LOWER($1) || '%' OR LOWER(code) LIKE '%' || LOWER($1) || '%' OR LOWER(name) LIKE '%' || LOWER($1) || '%' OR LOWER(COALESCE(base_url,'')) LIKE '%' || LOWER($1) || '%'`
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM suppliers `+filter, keyword).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.Pool.Query(ctx, `
SELECT public_id, code, name, base_url, auth_header, token_encrypted, enabled, status, updated_at
FROM suppliers `+filter+` ORDER BY code LIMIT $2 OFFSET $3`, keyword, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Supplier, 0, pageSize)
	for rows.Next() {
		var item Supplier
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.BaseURL, &item.AuthHeader, &item.TokenEncrypted, &item.Enabled, &item.Status, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.TokenConfigured = item.TokenEncrypted != ""
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Store) ListAdminCarriersWithModelsPage(ctx context.Context, page, pageSize int, keyword string) ([]Carrier, []Model, int, error) {
	page, pageSize = normalizeAdminPagination(page, pageSize)
	const carrierFilter = `
WHERE ($1 = '' OR LOWER(c.public_id) LIKE '%' || LOWER($1) || '%' OR LOWER(c.code) LIKE '%' || LOWER($1) || '%' OR LOWER(c.name) LIKE '%' || LOWER($1) || '%' OR LOWER(COALESCE(c.description,'')) LIKE '%' || LOWER($1) || '%'
  OR EXISTS (
  SELECT 1 FROM models m
  WHERE m.carrier_id = c.id AND COALESCE(m.client_workflow_id, '') = ''
    AND ($1 = '' OR LOWER(m.public_id) LIKE '%' || LOWER($1) || '%' OR LOWER(m.code) LIKE '%' || LOWER($1) || '%' OR LOWER(m.name) LIKE '%' || LOWER($1) || '%' OR LOWER(COALESCE(m.description,'')) LIKE '%' || LOWER($1) || '%')
  ))
  AND (EXISTS (
    SELECT 1 FROM models m WHERE m.carrier_id = c.id AND COALESCE(m.client_workflow_id, '') = ''
  ) OR NOT EXISTS (
  SELECT 1 FROM models m WHERE m.carrier_id = c.id
  ))`
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM carriers c `+carrierFilter, keyword).Scan(&total); err != nil {
		return nil, nil, 0, err
	}
	carriersRows, err := s.Pool.Query(ctx, `
SELECT c.public_id, c.code, c.name, c.description, c.enabled, c.sort_order
FROM carriers c `+carrierFilter+`
ORDER BY c.sort_order LIMIT $2 OFFSET $3`, keyword, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, nil, 0, err
	}
	defer carriersRows.Close()
	carriers := make([]Carrier, 0, pageSize)
	carrierIDs := make([]string, 0, pageSize)
	for carriersRows.Next() {
		var item Carrier
		if err := carriersRows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Enabled, &item.SortOrder); err != nil {
			return nil, nil, 0, err
		}
		carriers = append(carriers, item)
		carrierIDs = append(carrierIDs, item.ID)
	}
	if err := carriersRows.Err(); err != nil {
		return nil, nil, 0, err
	}
	if len(carrierIDs) == 0 {
		return carriers, []Model{}, total, nil
	}
	modelRows, err := s.Pool.Query(ctx, `
SELECT m.public_id, c.public_id, c.code, m.code, m.name, m.description, m.client_workflow_id, m.enabled, m.sort_order
FROM models m
JOIN carriers c ON c.id = m.carrier_id
WHERE COALESCE(m.client_workflow_id, '') = '' AND c.public_id = ANY($1::text[])
ORDER BY c.sort_order, m.sort_order`, carrierIDs)
	if err != nil {
		return nil, nil, 0, err
	}
	defer modelRows.Close()
	models := make([]Model, 0)
	for modelRows.Next() {
		var item Model
		if err := modelRows.Scan(&item.ID, &item.CarrierID, &item.CarrierCode, &item.Code, &item.Name, &item.Description, &item.ClientWorkflowID, &item.Enabled, &item.SortOrder); err != nil {
			return nil, nil, 0, err
		}
		models = append(models, item)
	}
	return carriers, models, total, modelRows.Err()
}

func (s *Store) ListAdminOfferingsPage(ctx context.Context, page, pageSize int, executionType, carrierID, modelID, supplierID string) ([]SupplierOffering, int, error) {
	page, pageSize = normalizeAdminPagination(page, pageSize)
	const from = `
FROM supplier_executions execution
JOIN supplier_model_bindings binding ON binding.id=execution.binding_id
JOIN suppliers supplier ON supplier.id=binding.supplier_id
JOIN models model ON model.id=binding.model_id
JOIN carriers carrier ON carrier.id=model.carrier_id
JOIN creation_modes mode ON mode.id=execution.mode_id
LEFT JOIN supplier_offerings o ON o.id=execution.legacy_offering_id`
	const filter = `WHERE ($1 = '' OR execution.execution_type=$1)
  AND ($2 = '' OR carrier.public_id=$2)
  AND ($3 = '' OR model.public_id=$3)
  AND ($4 = '' OR supplier.public_id=$4)`
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) `+from+` `+filter, executionType, carrierID, modelID, supplierID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.Pool.Query(ctx, `
SELECT execution.public_id, COALESCE(o.public_id,''), supplier.public_id, supplier.code, supplier.name, supplier.base_url, supplier.auth_header,
       model.public_id, model.code, model.name, carrier.name, mode.public_id, mode.code, mode.name,
       COALESCE(o.name, NULLIF(execution.workflow_code,''), NULLIF(execution.api_model_code,''), model.name),
       COALESCE(o.upstream_workflow_id, execution.workflow_code), execution.execution_type, execution.api_model_code, execution.api_version, execution.workflow_code, execution.adapter_code, execution.enabled, execution.health_status, execution.quota_status, execution.priority,
       execution.min_duration_sec, execution.max_duration_sec, execution.resolutions, execution.aspect_ratios, execution.min_images, execution.max_images,
       execution.supports_generated_audio `+from+` `+filter+`
ORDER BY supplier.code, model.code, mode.sort_order, execution.priority, execution.public_id LIMIT $5 OFFSET $6`, executionType, carrierID, modelID, supplierID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]SupplierOffering, 0, pageSize)
	for rows.Next() {
		var item SupplierOffering
		var resRaw, aspectRaw []byte
		if err := rows.Scan(&item.ID, &item.LegacyOfferingID, &item.SupplierID, &item.SupplierCode, &item.SupplierName, &item.SupplierBaseURL, &item.SupplierAuthHeader,
			&item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName, &item.ModeID, &item.ModeCode, &item.ModeName,
			&item.Name, &item.UpstreamWorkflowID, &item.ExecutionType, &item.APIModelCode, &item.APIVersion, &item.WorkflowCode, &item.AdapterCode, &item.Enabled, &item.HealthStatus, &item.QuotaStatus, &item.Priority,
			&item.MinDurationSec, &item.MaxDurationSec, &resRaw, &aspectRaw, &item.MinImages, &item.MaxImages, &item.SupportsGeneratedAudio); err != nil {
			return nil, 0, err
		}
		item.ExecutionID = item.ID
		_ = json.Unmarshal(resRaw, &item.Resolutions)
		_ = json.Unmarshal(aspectRaw, &item.AspectRatios)
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Store) ListAdminSupplierCostRulesPage(ctx context.Context, page, pageSize int, executionIDs []string) ([]SupplierCostRule, int, error) {
	page, pageSize = normalizeAdminPagination(page, pageSize)
	const from = `
FROM supplier_cost_rules cost
LEFT JOIN supplier_offerings offering ON offering.id=cost.offering_id
	LEFT JOIN supplier_executions execution ON execution.id=cost.execution_id`
	const filter = `WHERE COALESCE(array_length($1::text[], 1), 0) = 0 OR execution.public_id = ANY($1::text[])`
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) `+from+` `+filter, executionIDs).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.Pool.Query(ctx, `
SELECT cost.public_id, COALESCE(offering.public_id,''), COALESCE(execution.public_id,''), cost.resolution, cost.generate_audio,
       cost.price_per_sec_peak::float8, cost.price_per_sec_offpeak::float8, cost.currency, cost.version, cost.status `+from+` `+filter+`
ORDER BY COALESCE(execution.public_id, offering.public_id), cost.resolution, cost.generate_audio LIMIT $2 OFFSET $3`, executionIDs, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]SupplierCostRule, 0, pageSize)
	for rows.Next() {
		var item SupplierCostRule
		if err := rows.Scan(&item.ID, &item.OfferingID, &item.ExecutionID, &item.Resolution, &item.GenerateAudio, &item.PeakPerSecond, &item.OffpeakPerSecond, &item.Currency, &item.Version, &item.Status); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Store) ListAdminRetailPriceRulesPage(ctx context.Context, page, pageSize int) ([]RetailPriceRule, int, error) {
	page, pageSize = normalizeAdminPagination(page, pageSize)
	const from = `
FROM retail_price_rules retail
JOIN creation_modes mode ON mode.id=retail.mode_id
LEFT JOIN models model ON model.id=retail.model_id
LEFT JOIN carriers carrier ON carrier.id=model.carrier_id`
	const filter = `WHERE retail.model_id IS NULL`
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) `+from+` `+filter).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.Pool.Query(ctx, `
SELECT retail.public_id, mode.public_id, mode.code, mode.name,
       COALESCE(model.public_id, ''), COALESCE(model.code, ''), COALESCE(model.name, ''), COALESCE(carrier.name, ''),
       retail.resolution, retail.generate_audio, retail.name, retail.price_per_sec::float8, retail.version, retail.status `+from+` `+filter+`
ORDER BY mode.sort_order, COALESCE(carrier.name,''), COALESCE(model.name,''), retail.resolution, retail.generate_audio LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]RetailPriceRule, 0, pageSize)
	for rows.Next() {
		var item RetailPriceRule
		if err := rows.Scan(&item.ID, &item.ModeID, &item.ModeCode, &item.ModeName, &item.ModelID, &item.ModelCode, &item.ModelName, &item.CarrierName,
			&item.Resolution, &item.GenerateAudio, &item.Name, &item.PricePerSecond, &item.Version, &item.Status); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
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
	var providerInFlight bool
	err = tx.QueryRow(ctx, `
SELECT id, user_id, status, cost::float8, title, execution_mode='provider' AND (submit_started OR submit_uncertain OR upstream_task_id<>'') FROM generation_tasks WHERE public_id=$1 FOR UPDATE`,
		taskPublicID).Scan(&id, &userID, &status, &cost, &title, &providerInFlight)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, errors.New("任务不存在")
	}
	if err != nil {
		return nil, 0, err
	}
	if status != "queued" && status != "running" {
		// Admin cancellation is also retried by the UI after a lost response.
		// Return the existing refund rather than issuing a second ledger entry
		// or reporting a misleading state transition error.
		if status == "failed" {
			var refunded float64
			if err = tx.QueryRow(ctx, `
SELECT COALESCE(SUM(amount::float8), 0)
FROM wallet_ledger
WHERE user_id=$1 AND ref_type='task' AND ref_id=$2 AND type='task_refund'`, userID, taskPublicID).Scan(&refunded); err != nil {
				return nil, 0, err
			}
			if refunded > 0 {
				if err = tx.Commit(ctx); err != nil {
					return nil, 0, err
				}
				return &Task{ID: taskPublicID, Title: title, Status: "失败", Cost: cost}, refunded, nil
			}
		}
		return nil, 0, errors.New("该任务无法终止")
	}
	if providerInFlight {
		return nil, 0, errors.New("任务已提交或提交结果未知，请先核对上游状态")
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
	if _, err = tx.Exec(ctx, `UPDATE consumption_orders SET status='refunded', refunded_at=NOW() WHERE task_id=$1 AND status='paid'`, id); err != nil {
		return nil, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return &Task{ID: taskPublicID, Title: title, Status: "失败", Cost: cost}, cost, nil
}

func (s *Store) AdminRetryTask(ctx context.Context, taskPublicID string) (*Task, error) {
	var status, quoteID, modeID string
	err := s.Pool.QueryRow(ctx, `SELECT status, quote_public_id, mode_public_id FROM generation_tasks WHERE public_id=$1`, taskPublicID).Scan(&status, &quoteID, &modeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("任务不存在")
	}
	if err != nil {
		return nil, err
	}
	if status != "failed" {
		return nil, errors.New("仅失败任务可重试")
	}
	if quoteID != "" || modeID != "" {
		return nil, errors.New("请按原参数重新开始创作并提交新任务")
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
		priceSnap, routeSnap, retailSnap, costSnap, offeringSnap, inputSnap, executionSnap                    []byte
		generateAudio                                                                                         bool
		executionType                                                                                         string
		userEmail, userName                                                                                   string
		created, updated                                                                                      time.Time
		resultURL, ossURL                                                                                     string
		providerRequestID                                                                                     string
		submitUncertain                                                                                       bool
		claimedAt, claimExpiresAt, deadline, nextPollAt                                                       *time.Time
		pollAttempts                                                                                          int
	)
	err := s.Pool.QueryRow(ctx, `
SELECT t.title, t.workflow, t.status, t.progress, t.duration_sec, t.resolution,
       t.cost::float8, t.price_per_sec::float8, t.cost_per_sec::float8,
       t.supplier_code, t.upstream_workflow_id, t.upstream_task_id, t.upstream_status, t.fail_reason,
       t.price_snapshot, t.route_snapshot, t.mode_public_id, t.model_public_id, t.quote_public_id, t.generate_audio,
       t.retail_snapshot, t.cost_snapshot, t.offering_snapshot, t.input_snapshot, t.execution_type, t.execution_snapshot, t.result_url, t.result_oss_url,
       u.email, u.display_name, t.created_at, t.updated_at,
       t.provider_request_id, t.submit_uncertain, t.claimed_at, t.claim_expires_at, t.execution_deadline, t.poll_attempts, t.next_poll_at
FROM generation_tasks t JOIN users u ON u.id=t.user_id
WHERE t.public_id=$1`, taskPublicID).Scan(
		&title, &workflow, &status, &progress, &duration, &resolution,
		&cost, &pricePerSec, &costPerSec,
		&supplier, &upWF, &upTask, &upStatus, &fail,
		&priceSnap, &routeSnap, &modeID, &modelID, &quoteID, &generateAudio,
		&retailSnap, &costSnap, &offeringSnap, &inputSnap, &executionType, &executionSnap, &resultURL, &ossURL,
		&userEmail, &userName, &created, &updated,
		&providerRequestID, &submitUncertain, &claimedAt, &claimExpiresAt, &deadline, &pollAttempts, &nextPollAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	events := make([]map[string]any, 0)
	rows, err := s.Pool.Query(ctx, `SELECT event_type, provider_request_id, upstream_status, detail, elapsed_ms, created_at FROM provider_task_events WHERE task_id=(SELECT id FROM generation_tasks WHERE public_id=$1) ORDER BY id DESC LIMIT 100`, taskPublicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, request, state string
		var detail []byte
		var elapsed int64
		var created time.Time
		if err := rows.Scan(&kind, &request, &state, &detail, &elapsed, &created); err != nil {
			return nil, err
		}
		events = append(events, map[string]any{"eventType": kind, "providerRequestId": request, "upstreamStatus": state, "detail": jsonRawOrObj(detail), "elapsedMs": elapsed, "createdAt": created})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{
		"id": taskPublicID, "title": title, "mode": workflow, "modeId": modeID,
		"modelId": modelID, "quoteId": quoteID, "generateAudio": generateAudio,
		"status": statusCN(status), "progress": progress, "durationSec": duration, "resolution": resolution,
		"cost": cost, "pricePerSec": pricePerSec, "costPerSec": costPerSec,
		"supplierCode": supplier, "upstreamWorkflowId": upWF, "upstreamTaskId": upTask, "upstreamStatus": upStatus,
		"executionType": executionType, "executionSnapshot": jsonRawOrObj(executionSnap),
		"providerRequestId": providerRequestID, "submitUncertain": submitUncertain,
		"claimedAt": claimedAt, "claimExpiresAt": claimExpiresAt, "executionDeadline": deadline,
		"pollAttempts": pollAttempts, "nextPollAt": nextPollAt, "providerEvents": events,
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
