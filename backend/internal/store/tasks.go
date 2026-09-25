package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Task struct {
	ID             string
	Title          string
	Workflow       string
	Status         string
	Progress       int
	DurationSec    int
	Resolution     string
	Cost           float64
	ThumbURL       string
	SavedToProject bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

// ClientTaskDetail is the creator-safe task view. It deliberately excludes
// supplier, procurement cost, route, upstream task, and raw provider result
// URL fields. The creator-facing precharged amount is retained on Task.
type ClientTaskDetail struct {
	Task
	AspectRatio      string
	GenerateAudio    bool
	FailureReason    string
	DeliveryStatus   string
	DownloadURL      string
	ExecutionWaiting bool
}
type CreateTaskInput struct {
	WorkflowID      string
	Title           string
	Prompt          string
	AspectRatio     string
	DurationSeconds int
	Resolution      string
	EstimatedCost   float64
	PricePerSecond  float64
	RuleID          string
	Version         string
	SourceType      string // text | upload | asset
	SourceRef       string
	SourceMeta      SourceMeta
}

// SourceRefItem points at an upload or project asset.
type SourceRefItem struct {
	Type string `json:"type"` // upload | asset
	ID   string `json:"id"`
}

// SourceMeta carries image inputs and an optional creator-provided reference
// audio upload. The audio source is retained in the task snapshot, never sent
// back through the client task API.
type SourceMeta struct {
	First  *SourceRefItem  `json:"first,omitempty"`
	Last   *SourceRefItem  `json:"last,omitempty"`
	Extras []SourceRefItem `json:"extras,omitempty"`
	Refs   []SourceRefItem `json:"refs,omitempty"`
	Audio  *SourceRefItem  `json:"audio,omitempty"`
}

func (m SourceMeta) Empty() bool {
	return m.VisualEmpty() && m.Audio == nil
}

func (m SourceMeta) VisualEmpty() bool {
	return m.First == nil && m.Last == nil && len(m.Extras) == 0 && len(m.Refs) == 0
}

func workflowLabel(id string) string {
	switch id {
	case "wf-h3-text":
		return "H3 文生视频"
	case "wf-h3-multi":
		return "H3 多图生视频"
	default:
		return "H3 图生视频"
	}
}

func workflowEnabled(id string) bool {
	return id == "wf-h3-image" || id == "wf-h3-text" || id == "wf-h3-multi"
}

func statusCN(status string) string {
	switch status {
	case "queued":
		return "排队中"
	case "running":
		return "生成中"
	case "completed":
		return "已完成"
	case "failed", "cancelled":
		return "失败"
	default:
		return status
	}
}

func statusDB(cn string) string {
	switch cn {
	case "排队中":
		return "queued"
	case "生成中":
		return "running"
	case "已完成":
		return "completed"
	case "失败":
		return "failed"
	default:
		return cn
	}
}

func (s *Store) CreateTask(ctx context.Context, userID int64, in CreateTaskInput) (*Task, string, float64, error) {
	wf, err := s.GetPlatformWorkflow(ctx, in.WorkflowID)
	if err != nil {
		return nil, "", 0, errors.New("工作流不存在")
	}
	if wf.Status != "published" {
		return nil, "", 0, errors.New("工作流未开放")
	}
	if in.Resolution == "" {
		in.Resolution = "768P"
		if wf.Params.DefaultResolution != "" {
			in.Resolution = wf.Params.DefaultResolution
		}
	}
	if in.DurationSeconds == 0 {
		in.DurationSeconds = 15
		if wf.Params.DefaultDuration > 0 {
			in.DurationSeconds = wf.Params.DefaultDuration
		}
	}
	if in.AspectRatio == "" {
		in.AspectRatio = "16:9"
	}

	rule, err := s.FindActivePrice(ctx, wf.Code, in.Resolution)
	if err != nil {
		return nil, "", 0, errors.New("暂无该分辨率定价")
	}
	cost := rule.PricePerSec * float64(in.DurationSeconds)
	pps := rule.PricePerSec

	if in.RuleID != "" && (in.RuleID != rule.ID || in.Version != rule.Version) {
		return nil, "", 0, errors.New("价格快照过期")
	}
	if in.EstimatedCost > 0 && abs(in.EstimatedCost-cost) > 0.01 {
		return nil, "", 0, errors.New("价格快照过期")
	}
	if in.PricePerSecond > 0 && abs(in.PricePerSecond-pps) > 0.001 {
		return nil, "", 0, errors.New("价格快照过期")
	}

	sourceType := in.SourceType
	sourceRef := strings.TrimSpace(in.SourceRef)
	switch wf.Code {
	case "wf-h3-text":
		if strings.TrimSpace(in.Prompt) == "" {
			return nil, "", 0, errors.New("请先填写画面描述")
		}
		if sourceType == "" {
			sourceType = "text"
		}
	case "wf-h3-image", "wf-h3-multi":
		if err := s.validateImageSources(ctx, userID, wf.Code, &sourceType, &sourceRef, &in.SourceMeta); err != nil {
			return nil, "", 0, err
		}
	default:
		return nil, "", 0, errors.New("工作流未开放")
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, "", 0, err
	}
	defer tx.Rollback(ctx)

	var balance float64
	if err := tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id = $1 FOR UPDATE`, userID).Scan(&balance); err != nil {
		return nil, "", 0, err
	}
	if balance < cost {
		return nil, "", 0, errors.New("insufficient balance")
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

	publicID := NewTaskID()
	thumb := "https://images.unsplash.com/photo-1519608487953-e999c86e7455?auto=format&fit=crop&w=480&q=80"
	priceSnap, _ := json.Marshal(map[string]any{
		"ruleId": rule.ID, "version": rule.Version, "pricePerSecond": pps,
		"estimatedCost": cost, "resolution": in.Resolution, "durationSeconds": in.DurationSeconds,
		"workflowId": wf.Code,
	})
	metaRaw, _ := json.Marshal(in.SourceMeta)
	var id int64
	err = tx.QueryRow(ctx, `
INSERT INTO generation_tasks (
  public_id, user_id, title, workflow, status, progress, duration_sec, resolution, cost, thumb_url,
  source_type, source_ref, source_meta, prompt, aspect_ratio,
  workflow_public_id, price_rule_id, price_version, price_per_sec, price_snapshot, model_code, carrier_code
) VALUES ($1, $2, $3, $4, 'queued', 0, $5, $6, $7, $8, $9, $10, $11::jsonb, $12, $13, $14, $15, $16, $17, $18::jsonb, $19, $20)
RETURNING id`,
		publicID, userID, title, wf.Name, in.DurationSeconds, in.Resolution, cost, thumb,
		sourceType, sourceRef, string(metaRaw), in.Prompt, in.AspectRatio,
		wf.Code, rule.ID, rule.Version, pps, string(priceSnap), wf.ModelCode, wf.CarrierCode,
	).Scan(&id)
	if err != nil {
		return nil, "", 0, err
	}

	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance = balance - $1, version = version + 1, updated_at = NOW() WHERE user_id = $2`, cost, userID); err != nil {
		return nil, "", 0, err
	}
	if err = tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id = $1`, userID).Scan(&balance); err != nil {
		return nil, "", 0, err
	}
	ledgerID := NewLedgerID()
	if _, err = tx.Exec(ctx, `
INSERT INTO wallet_ledger (public_id, user_id, type, amount, balance_after, ref_type, ref_id, description)
VALUES ($1, $2, 'task_charge', $3, $4, 'task', $5, $6)`,
		ledgerID, userID, -cost, balance, publicID, fmt.Sprintf("%s %s", publicID, title)); err != nil {
		return nil, "", 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE generation_tasks SET ledger_public_id=$1 WHERE id=$2`, ledgerID, id); err != nil {
		return nil, "", 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, "", 0, err
	}

	return &Task{
		ID: publicID, Title: title, Workflow: wf.Name,
		Status: "排队中", Progress: 0, DurationSec: in.DurationSeconds,
		Resolution: in.Resolution, Cost: cost, ThumbURL: thumb,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, ledgerID, balance, nil
}

func (s *Store) ListTasks(ctx context.Context, userID int64, statusCNFilter string, page, pageSize int) ([]Task, int, map[string]int, error) {
	counts := map[string]int{"全部": 0, "排队中": 0, "生成中": 0, "已完成": 0, "失败": 0}
	rows, err := s.Pool.Query(ctx, `SELECT status, COUNT(*) FROM generation_tasks WHERE user_id = $1 GROUP BY status`, userID)
	if err != nil {
		return nil, 0, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var c int
		if err := rows.Scan(&st, &c); err != nil {
			return nil, 0, nil, err
		}
		cn := statusCN(st)
		counts[cn] += c
		counts["全部"] += c
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := `SELECT public_id, title, workflow, status, progress, duration_sec, resolution, cost::float8, thumb_url, saved_to_project, completed_at, created_at, updated_at
FROM generation_tasks WHERE user_id = $1`
	args := []any{userID}
	if statusCNFilter != "" && statusCNFilter != "全部" {
		query += ` AND status = $2`
		args = append(args, statusDB(statusCNFilter))
	}
	query += ` ORDER BY updated_at DESC`

	countQuery := `SELECT COUNT(*) FROM generation_tasks WHERE user_id = $1`
	countArgs := []any{userID}
	if statusCNFilter != "" && statusCNFilter != "全部" {
		countQuery += ` AND status = $2`
		countArgs = append(countArgs, statusDB(statusCNFilter))
	}
	var total int
	if err := s.Pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, nil, err
	}

	query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, (page-1)*pageSize)
	rows, err = s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, nil, err
	}
	defer rows.Close()

	var items []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, nil, err
		}
		items = append(items, *t)
	}
	return items, total, counts, nil
}

func (s *Store) GetClientTaskDetail(ctx context.Context, userID int64, taskID string) (*ClientTaskDetail, error) {
	var item ClientTaskDetail
	var status, upstreamStatus, resultOSS string
	var completed *time.Time
	err := s.Pool.QueryRow(ctx, `
SELECT public_id, title, workflow, status, progress, duration_sec, resolution, cost::float8,
       thumb_url, saved_to_project, completed_at, created_at, updated_at,
       COALESCE(aspect_ratio,''), generate_audio, COALESCE(fail_reason,''),
       COALESCE(upstream_status,''), COALESCE(result_oss_url,'')
FROM generation_tasks
WHERE public_id=$1 AND user_id=$2`, taskID, userID).Scan(
		&item.ID, &item.Title, &item.Workflow, &status, &item.Progress, &item.DurationSec, &item.Resolution,
		&item.Cost, &item.ThumbURL, &item.SavedToProject, &completed, &item.CreatedAt, &item.UpdatedAt,
		&item.AspectRatio, &item.GenerateAudio, &item.FailureReason, &upstreamStatus, &resultOSS,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.Status = statusCN(status)
	item.CompletedAt = completed
	item.ExecutionWaiting = upstreamStatus == "WAITING_CONFIGURATION"
	switch {
	case item.Status == "已完成" && resultOSS != "":
		item.DeliveryStatus = "ready"
		item.DownloadURL = resultOSS
	case item.Status == "已完成" || upstreamStatus == "WAITING_STORAGE":
		item.DeliveryStatus = "pending_storage"
	default:
		item.DeliveryStatus = "not_ready"
	}
	return &item, nil
}

func scanTask(scanner interface{ Scan(dest ...any) error }) (*Task, error) {
	var t Task
	var status string
	var saved bool
	var completed *time.Time
	if err := scanner.Scan(&t.ID, &t.Title, &t.Workflow, &status, &t.Progress, &t.DurationSec, &t.Resolution, &t.Cost, &t.ThumbURL, &saved, &completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	t.Status = statusCN(status)
	t.SavedToProject = saved
	t.CompletedAt = completed
	return &t, nil
}

func (s *Store) CancelTask(ctx context.Context, userID int64, taskPublicID string) (*Task, float64, float64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	defer tx.Rollback(ctx)

	var id int64
	var status string
	var cost float64
	var title string
	err = tx.QueryRow(ctx, `
SELECT id, status, cost::float8, title FROM generation_tasks WHERE public_id = $1 AND user_id = $2 FOR UPDATE`,
		taskPublicID, userID).Scan(&id, &status, &cost, &title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, 0, errors.New("任务不存在")
	}
	if err != nil {
		return nil, 0, 0, err
	}
	if status != "queued" && status != "running" {
		return nil, 0, 0, errors.New("该任务无法终止")
	}
	if _, err = tx.Exec(ctx, `
UPDATE generation_tasks SET status='failed', progress=0, upstream_status='CANCELLED_BY_USER',
  fail_reason='用户终止并退款', updated_at=NOW()
WHERE id=$1`, id); err != nil {
		return nil, 0, 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance = balance + $1, version = version + 1, updated_at = NOW() WHERE user_id = $2`, cost, userID); err != nil {
		return nil, 0, 0, err
	}
	var balance float64
	if err = tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id = $1`, userID).Scan(&balance); err != nil {
		return nil, 0, 0, err
	}
	ledgerID := NewLedgerID()
	if _, err = tx.Exec(ctx, `
INSERT INTO wallet_ledger (public_id, user_id, type, amount, balance_after, ref_type, ref_id, description)
VALUES ($1, $2, 'task_refund', $3, $4, 'task', $5, $6)`,
		ledgerID, userID, cost, balance, taskPublicID, taskPublicID+" 终止退款"); err != nil {
		return nil, 0, 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE consumption_orders SET status='refunded', refunded_at=NOW() WHERE task_id=$1 AND status='paid'`, id); err != nil {
		return nil, 0, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, 0, 0, err
	}
	return &Task{ID: taskPublicID, Title: title, Status: "失败", Cost: cost}, cost, balance, nil
}

func (s *Store) RetryTask(ctx context.Context, userID int64, taskPublicID string) (*Task, error) {
	var status, quoteID string
	err := s.Pool.QueryRow(ctx, `SELECT status, quote_public_id FROM generation_tasks WHERE public_id = $1 AND user_id = $2`, taskPublicID, userID).Scan(&status, &quoteID)
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
		return nil, errors.New("请重新配置并获取新报价后再提交")
	}
	_, err = s.Pool.Exec(ctx, `
UPDATE generation_tasks SET
  status = 'queued', progress = 0, fail_reason = '',
  upstream_task_id = '', upstream_status = '', result_url = '', result_oss_url = '',
  supplier_code = '', supplier_route_id = '', upstream_workflow_id = '',
  started_at = NULL, completed_at = NULL, updated_at = NOW()
WHERE public_id = $1`, taskPublicID)
	if err != nil {
		return nil, err
	}
	return &Task{ID: taskPublicID, Status: "排队中"}, nil
}

func (s *Store) AdvanceTasks(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT id, status, progress FROM generation_tasks WHERE status IN ('queued', 'running')`)
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
		switch it.status {
		case "queued":
			_, err = s.Pool.Exec(ctx, `UPDATE generation_tasks SET status = 'running', progress = 5, updated_at = NOW() WHERE id = $1`, it.id)
		case "running":
			next := it.progress + 12
			if next >= 100 {
				_, err = s.Pool.Exec(ctx, `UPDATE generation_tasks SET status = 'completed', progress = 100, completed_at = NOW(), updated_at = NOW() WHERE id = $1`, it.id)
			} else {
				_, err = s.Pool.Exec(ctx, `UPDATE generation_tasks SET progress = $1, updated_at = NOW() WHERE id = $2`, next, it.id)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func (s *Store) validateSourceRefItem(ctx context.Context, userID int64, item *SourceRefItem) error {
	if item == nil || strings.TrimSpace(item.ID) == "" {
		return errors.New("图生视频需要素材")
	}
	switch item.Type {
	case "upload":
		if _, err := s.GetUpload(ctx, userID, item.ID); err != nil {
			return errors.New("上传素材不存在")
		}
	case "asset":
		ast, _, err := s.FindUserAsset(ctx, userID, item.ID)
		if err != nil {
			return errors.New("资产素材不存在")
		}
		if ast.Type != "image" {
			return errors.New("请选择图片资产")
		}
	default:
		return errors.New("不支持的素材来源")
	}
	return nil
}

func isReferenceAudioMIME(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "audio/mpeg", "audio/wav", "audio/x-wav", "audio/flac", "audio/mp4", "video/mp4":
		return true
	default:
		return false
	}
}

func (s *Store) validateReferenceAudioSource(ctx context.Context, userID int64, item *SourceRefItem) error {
	if item == nil || strings.TrimSpace(item.ID) == "" {
		return errors.New("请选择参考音频")
	}
	if item.Type != "upload" {
		return errors.New("参考音频仅支持本地上传")
	}
	upload, err := s.GetUpload(ctx, userID, item.ID)
	if err != nil {
		return errors.New("上传参考音频不存在")
	}
	if !isReferenceAudioMIME(upload.MimeType) {
		return errors.New("参考音频格式不受支持")
	}
	return nil
}

func (s *Store) validateImageSources(ctx context.Context, userID int64, workflowCode string, sourceType, sourceRef *string, meta *SourceMeta) error {
	if meta == nil {
		meta = &SourceMeta{}
	}
	// Back-compat: single source → first / refs[0]
	if meta.VisualEmpty() && strings.TrimSpace(*sourceRef) != "" {
		item := SourceRefItem{Type: *sourceType, ID: strings.TrimSpace(*sourceRef)}
		if item.Type == "" {
			item.Type = "upload"
		}
		if workflowCode == "wf-h3-multi" {
			meta.Refs = []SourceRefItem{item}
		} else {
			meta.First = &item
		}
	}

	switch workflowCode {
	case "wf-h3-image":
		if meta.First == nil {
			return errors.New("图生视频需要首帧素材")
		}
		if err := s.validateSourceRefItem(ctx, userID, meta.First); err != nil {
			return err
		}
		if meta.Last != nil {
			if err := s.validateSourceRefItem(ctx, userID, meta.Last); err != nil {
				return err
			}
		}
		for i := range meta.Extras {
			if err := s.validateSourceRefItem(ctx, userID, &meta.Extras[i]); err != nil {
				return err
			}
		}
		*sourceType = meta.First.Type
		*sourceRef = meta.First.ID
	case "wf-h3-multi":
		if len(meta.Refs) == 0 && meta.First != nil {
			meta.Refs = append(meta.Refs, *meta.First)
		}
		if len(meta.Refs) == 0 {
			return errors.New("多图生视频至少需要一张参考图")
		}
		if len(meta.Refs) > 9 {
			return errors.New("参考图最多 9 张")
		}
		for i := range meta.Refs {
			if err := s.validateSourceRefItem(ctx, userID, &meta.Refs[i]); err != nil {
				return err
			}
		}
		*sourceType = meta.Refs[0].Type
		*sourceRef = meta.Refs[0].ID
	}
	return nil
}

// ResolveLocalImage returns local path + mime for an upload/asset ref.
func (s *Store) ResolveLocalImage(ctx context.Context, userID int64, item SourceRefItem) (path, mime string, remoteURL string, err error) {
	switch item.Type {
	case "upload":
		up, e := s.GetUpload(ctx, userID, item.ID)
		if e != nil {
			return "", "", "", errors.New("上传素材不存在")
		}
		return up.StoragePath, up.MimeType, "", nil
	case "asset":
		ast, _, e := s.FindUserAsset(ctx, userID, item.ID)
		if e != nil {
			return "", "", "", errors.New("资产素材不存在")
		}
		if ast.RemoteURL != "" && ast.StoragePath == "" {
			return "", ast.MimeType, ast.RemoteURL, nil
		}
		return ast.StoragePath, ast.MimeType, ast.RemoteURL, nil
	default:
		return "", "", "", errors.New("不支持的素材来源")
	}
}
