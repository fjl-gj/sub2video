package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/provider/autodl"
	"github.com/sbu2video/backend/internal/routing"
	"github.com/sbu2video/backend/internal/storage/blob"
	"github.com/sbu2video/backend/internal/store"
)

type Runner struct {
	Store  *store.Store
	Cfg    config.Config
	Client *autodl.Client
}

// submitUncertainError means the request may have crossed the provider
// boundary. It must never enter the normal refund/retry path automatically.
type submitUncertainError struct {
	err               error
	providerRequestID string
}

func (e submitUncertainError) Error() string { return e.err.Error() }
func (e submitUncertainError) Unwrap() error { return e.err }

func Start(ctx context.Context, st *store.Store, cfg config.Config, interval time.Duration) {
	r := &Runner{Store: st, Cfg: cfg}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.Tick(ctx); err != nil {
					log.Printf("provider worker: %v", err)
				}
			}
		}
	}()
}

func (r *Runner) Tick(ctx context.Context) error {
	if err := r.Store.RecoverExpiredProviderSubmissions(ctx); err != nil {
		return err
	}
	tasks, err := r.Store.ListPendingProviderTasks(ctx, 30)
	if err != nil {
		return err
	}
	for i := range tasks {
		t := &tasks[i]
		if t.UpstreamTaskID != "" {
			claimed, claimToken, claimErr := r.Store.ClaimTaskForPollWithToken(ctx, t.ID)
			if claimErr != nil {
				log.Printf("claim poll %s: %v", t.PublicID, claimErr)
				continue
			}
			if !claimed {
				continue
			}
			t.ClaimToken = claimToken
			if t.ExecutionDeadline != nil && time.Now().After(*t.ExecutionDeadline) {
				reason := "供应商执行超时"
				if t.UpstreamStatus == "WAITING_STORAGE" {
					reason = "对象存储交付等待超时"
				}
				if err := r.Store.FailTaskUpstreamWithToken(ctx, t.ID, reason, claimToken); err != nil {
					log.Printf("timeout %s: %v", t.PublicID, err)
				}
				continue
			}
			if err := r.poll(ctx, t); err != nil {
				log.Printf("poll %s: %v", t.PublicID, err)
				// Transient poll failures release the lease and back off. The
				// deadline check above turns a long-running task into a refund.
				backoff := time.Duration(1<<minInt(t.PollAttempts, 5)) * time.Second
				if retryErr := r.Store.RecordProviderPollFailureWithToken(ctx, t.ID, err.Error(), backoff, claimToken); retryErr != nil {
					log.Printf("record poll failure %s: %v", t.PublicID, retryErr)
				}
			}
			continue
		}
		claimed, claimToken, err := r.Store.ClaimTaskForDispatchWithToken(ctx, t.ID)
		if err != nil {
			log.Printf("claim dispatch %s: %v", t.PublicID, err)
			continue
		}
		if !claimed {
			continue
		}
		t.ClaimToken = claimToken
		if t.ExecutionDeadline != nil && time.Now().After(*t.ExecutionDeadline) {
			if timeoutErr := r.Store.FailTaskUpstreamWithToken(ctx, t.ID, "任务执行超时", claimToken); timeoutErr != nil {
				log.Printf("timeout %s: %v", t.PublicID, timeoutErr)
			}
			continue
		}
		if err := r.dispatchLockedOffering(ctx, t, claimToken); err != nil {
			log.Printf("dispatch %s: %v", t.PublicID, err)
			var uncertain submitUncertainError
			if errors.As(err, &uncertain) {
				// Persist the uncertainty even when shutdown cancelled the parent
				// request, with a short bound. A database outage is recovered by
				// the expired-submit sweep on the next healthy worker tick.
				finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				unknownErr := r.Store.MarkTaskSubmissionUnknownWithRequestID(finalizeCtx, t.ID, err.Error(), claimToken, uncertain.providerRequestID)
				cancel()
				if unknownErr != nil {
					log.Printf("mark submit unknown %s: %v", t.PublicID, unknownErr)
				}
			} else {
				_ = r.Store.FailTaskUpstreamWithToken(ctx, t.ID, err.Error(), claimToken)
			}
		}
	}
	if err := r.Store.AdvanceSimulatedTasks(ctx); err != nil {
		return err
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *Runner) objectStore(ctx context.Context) blob.Store {
	return r.Store.LoadObjectStore(ctx, r.Cfg)
}

func (r *Runner) dispatchLockedOffering(ctx context.Context, t *store.ProviderTask, claimToken string) error {
	if t.SupplierRouteID == "" || (t.ExecutionType == "workflow" && t.UpstreamWorkflowID == "") || (t.ExecutionType == "api" && t.APIModelCode == "") {
		return fmt.Errorf("任务缺少锁定供给")
	}
	if t.AdapterCode != "autodl_h3_v1" {
		return fmt.Errorf("不支持的锁定供给适配器: %s", t.AdapterCode)
	}
	if t.ExecutionType == "api" {
		return fmt.Errorf("API 执行适配器未验证，当前不可执行")
	}
	token, err := r.Store.ResolveSupplierToken(ctx, r.Cfg.SecretKey, t.SupplierCode, r.Cfg.AutoDLToken)
	if err != nil || token == "" {
		return r.Store.MarkTaskWaitingForConfigurationWithToken(ctx, t.ID, "锁定供应商凭据尚未配置", claimToken)
	}
	baseURL := r.Cfg.AutoDLBaseURL
	authHeader := "Authorization"
	if supplier, supplierErr := r.Store.GetSupplierByCode(ctx, t.SupplierCode); supplierErr == nil {
		if supplier.BaseURL != "" {
			baseURL = supplier.BaseURL
		}
		if supplier.AuthHeader != "" {
			authHeader = supplier.AuthHeader
		}
	}
	if baseURL == "" {
		baseURL = r.Cfg.AutoDLBaseURL
	}
	client := autodl.New(baseURL, token, authHeader)
	body := buildProviderRequestBody(t)
	if !isTextCreation(t.ClientWorkflow) {
		storeClient := r.objectStore(ctx)
		if !storeClient.Configured() {
			return r.Store.MarkTaskWaitingForConfigurationWithToken(ctx, t.ID, "对象存储尚未配置，无法提交图片素材", claimToken)
		}
		if err := storeClient.EnsureBucket(ctx); err != nil {
			return r.Store.MarkTaskWaitingForConfigurationWithToken(ctx, t.ID, fmt.Sprintf("对象存储暂不可用: %v", err), claimToken)
		}
		if err := r.attachImageURLs(ctx, storeClient, t, body); err != nil {
			if !strings.Contains(err.Error(), "素材") && !strings.Contains(err.Error(), "首尾帧") && !strings.Contains(err.Error(), "多图") {
				return r.Store.MarkTaskWaitingForConfigurationWithToken(ctx, t.ID, fmt.Sprintf("对象存储暂不可用: %v", err), claimToken)
			}
			return err
		}
	}
	if err := r.attachReferenceAudio(ctx, t, body); err != nil {
		return r.Store.MarkTaskWaitingForConfigurationWithToken(ctx, t.ID, fmt.Sprintf("对象存储暂不可用: %v", err), claimToken)
	}
	if err := r.Store.MarkTaskSubmitStarted(ctx, t.ID, claimToken); err != nil {
		return err
	}
	var res *autodl.SubmitResult
	if t.ExecutionType == "api" {
		res, err = client.SubmitAPI(ctx, t.APIModelCode, t.APIVersion, body)
	} else {
		workflowCode := t.UpstreamWorkflowID
		if t.WorkflowCode != "" {
			workflowCode = t.WorkflowCode
		}
		res, err = client.Submit(ctx, workflowCode, body)
	}
	if err != nil {
		if autodl.IsSubmitUncertain(err) {
			var providerErr *autodl.SubmitError
			if errors.As(err, &providerErr) {
				return submitUncertainError{err: err, providerRequestID: providerErr.ProviderRequestID}
			}
			return submitUncertainError{err: err}
		}
		return err
	}
	if err := r.Store.MarkTaskSubmittedWithToken(ctx, t.ID, res.TaskID, res.Status, claimToken); err != nil {
		// The provider has accepted this request. A failed/ambiguous database
		// write must never fall into automatic refund or a fresh submission.
		log.Printf("accepted submission requires reconciliation task=%s provider_request_id=%s", t.PublicID, res.TaskID)
		return submitUncertainError{err: fmt.Errorf("persist accepted provider submission: %w", err), providerRequestID: res.TaskID}
	}
	log.Printf("submitted %s via locked offering %s", t.PublicID, t.SupplierRouteID)
	return nil
}

func buildProviderRequestBody(t *store.ProviderTask) map[string]any {
	body := map[string]any{
		"prompt":     t.Prompt,
		"duration":   t.DurationSec,
		"resolution": routing.MapResolutionToUpstream(t.Resolution, t.AspectRatio),
	}
	// Workflow contracts express audio through their own accepted fields. Only
	// API executions receive a generic generated-audio switch.
	if t.GenerateAudio && t.ExecutionType == "api" {
		body["generate_audio"] = true
	}
	if omitted, ok := t.ParameterMapping["omit"].([]any); ok {
		for _, value := range omitted {
			if key, ok := value.(string); ok {
				delete(body, key)
			}
		}
	}
	return body
}

func (r *Runner) attachReferenceAudio(ctx context.Context, t *store.ProviderTask, body map[string]any) error {
	if t.SourceMeta.Audio == nil {
		return nil
	}
	storeClient := r.objectStore(ctx)
	if !storeClient.Configured() {
		return fmt.Errorf("对象存储尚未配置，无法提交参考音频")
	}
	if err := storeClient.EnsureBucket(ctx); err != nil {
		return err
	}
	path, mime, remote, err := r.Store.ResolveLocalImage(ctx, t.UserID, *t.SourceMeta.Audio)
	if err != nil {
		return err
	}
	var url string
	if remote != "" && (path == "" || strings.HasPrefix(remote, "http")) {
		url = remote
	} else {
		if path == "" {
			return fmt.Errorf("参考音频文件不可用")
		}
		key := fmt.Sprintf("sources/%s/%s/reference_audio%s", time.Now().UTC().Format("20060102"), t.PublicID, extOf(path, mime))
		url, err = storeClient.PutFile(ctx, key, path, mime)
		if err != nil {
			return err
		}
	}
	field := "ref_audio_0"
	if configured, ok := t.ParameterMapping["audioField"].(string); ok && configured != "" {
		field = configured
	}
	body[field] = url
	if configured, ok := t.ParameterMapping["audioDurationField"].(string); ok && configured != "" {
		body[configured] = t.DurationSec
	}
	return nil
}

func isTextCreation(code string) bool {
	return code == "text_to_video" || code == "wf-h3-text"
}

func isMultiImageCreation(code string) bool {
	return code == "multi_image_to_video" || code == "wf-h3-multi"
}

func (r *Runner) attachImageURLs(ctx context.Context, storeClient blob.Store, t *store.ProviderTask, body map[string]any) error {
	meta := t.SourceMeta
	if meta.Empty() && t.SourceRef != "" {
		item := store.SourceRefItem{Type: t.SourceType, ID: t.SourceRef}
		if item.Type == "" {
			item.Type = "upload"
		}
		if isMultiImageCreation(t.ClientWorkflow) {
			meta.Refs = []store.SourceRefItem{item}
		} else {
			meta.First = &item
		}
	}

	uploadSlot := func(slot string, item store.SourceRefItem) (string, error) {
		path, mime, remote, err := r.Store.ResolveLocalImage(ctx, t.UserID, item)
		if err != nil {
			return "", err
		}
		if remote != "" && (path == "" || strings.HasPrefix(remote, "http")) {
			// Already a remote URL — optionally mirror to COS for durability.
			if strings.HasPrefix(remote, "http://") || strings.HasPrefix(remote, "https://") {
				key := fmt.Sprintf("sources/%s/%s/%s%s", time.Now().UTC().Format("20060102"), t.PublicID, slot, extOf(remote, mime))
				mirrored, err := storeClient.PullURL(ctx, remote, key, mime)
				if err == nil && mirrored != "" {
					return mirrored, nil
				}
				return remote, nil
			}
		}
		if path == "" {
			return "", fmt.Errorf("素材文件不可用: %s", slot)
		}
		key := fmt.Sprintf("sources/%s/%s/%s%s", time.Now().UTC().Format("20060102"), t.PublicID, slot, extOf(path, mime))
		return storeClient.PutFile(ctx, key, path, mime)
	}

	switch {
	case isMultiImageCreation(t.ClientWorkflow):
		refs := meta.Refs
		if len(refs) == 0 && meta.First != nil {
			refs = []store.SourceRefItem{*meta.First}
		}
		if len(refs) == 0 {
			return fmt.Errorf("多图任务缺少参考图")
		}
		for i, ref := range refs {
			if i > 8 {
				break
			}
			url, err := uploadSlot(fmt.Sprintf("ref_%d", i), ref)
			if err != nil {
				return err
			}
			body[fmt.Sprintf("ref_image_%d", i)] = url
		}
	default: // wf-h3-image keyframes
		if meta.First == nil {
			return fmt.Errorf("首尾帧任务缺少首帧")
		}
		firstURL, err := uploadSlot("first", *meta.First)
		if err != nil {
			return err
		}
		body["first_frame"] = firstURL
		lastItem := meta.Last
		if lastItem == nil {
			lastItem = meta.First
		}
		lastURL, err := uploadSlot("last", *lastItem)
		if err != nil {
			return err
		}
		body["last_frame"] = lastURL
	}
	return nil
}

func extOf(path, mime string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif" || ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".m4a" || ext == ".mp4" {
		return ext
	}
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/flac":
		return ".flac"
	case "audio/mp4", "video/mp4":
		return ".m4a"
	default:
		return ".jpg"
	}
}

func (r *Runner) poll(ctx context.Context, t *store.ProviderTask) error {
	if t.UpstreamStatus == "WAITING_STORAGE" && t.ResultURL != "" {
		return r.persistResult(ctx, t, t.ResultURL, 0)
	}
	token, err := r.Store.ResolveSupplierToken(ctx, r.Cfg.SecretKey, t.SupplierCode, r.Cfg.AutoDLToken)
	if err != nil || token == "" {
		return r.Store.MarkTaskWaitingForConfigurationWithToken(ctx, t.ID, "锁定供应商凭据尚未配置", t.ClaimToken)
	}
	baseURL := r.Cfg.AutoDLBaseURL
	authHeader := "Authorization"
	if supplier, err := r.Store.GetSupplierByCode(ctx, t.SupplierCode); err == nil {
		if supplier.BaseURL != "" {
			baseURL = supplier.BaseURL
		}
		if supplier.AuthHeader != "" {
			authHeader = supplier.AuthHeader
		}
	}
	client := autodl.New(baseURL, token, authHeader)
	started := time.Now()
	var res *autodl.QueryResult
	if t.ExecutionType == "api" {
		res, err = client.QueryAPI(ctx, t.UpstreamTaskID)
	} else {
		res, err = client.Query(ctx, t.UpstreamTaskID)
	}
	if err != nil {
		return err
	}
	_ = r.Store.RecordProviderEvent(ctx, t.ID, "polled", t.ExecutionType, t.UpstreamTaskID, res.Status, nil, time.Since(started).Milliseconds())
	status := strings.ToUpper(res.Status)
	switch status {
	case "SUCCESS", "COMPLETED", "DONE":
		url := t.ResultURL
		for _, item := range res.Results {
			if item.URL != "" {
				url = item.URL
				break
			}
		}
		return r.persistResult(ctx, t, url, res.Duration)
	case "FAILED", "ERROR", "CANCELLED", "CANCELED":
		return r.Store.FailTaskUpstreamWithToken(ctx, t.ID, "upstream "+status, t.ClaimToken)
	case "QUEUED", "PENDING":
		return r.Store.UpdateTaskUpstreamWithRetryAndToken(ctx, t.ID, res.Status, 12, 12*time.Second, t.ClaimToken)
	case "RUNNING", "PROCESSING":
		prog := t.Progress
		if prog < 20 {
			prog = 20
		} else if prog < 90 {
			prog += 8
		}
		return r.Store.UpdateTaskUpstreamWithRetryAndToken(ctx, t.ID, res.Status, prog, 12*time.Second, t.ClaimToken)
	default:
		return r.Store.UpdateTaskUpstreamWithRetryAndToken(ctx, t.ID, res.Status, t.Progress, 12*time.Second, t.ClaimToken)
	}
}

func (r *Runner) persistResult(ctx context.Context, t *store.ProviderTask, resultURL string, upstreamDuration int) error {
	if resultURL == "" {
		return fmt.Errorf("upstream success without result URL")
	}
	// PullURL can download a large video. Keep it below the two-minute poll
	// lease so a second worker can safely retry after a crash; completion is
	// fenced by the claim token below.
	deliveryCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	storeClient := r.objectStore(deliveryCtx)
	if !storeClient.Configured() {
		return r.Store.MarkTaskDeliveryPendingWithToken(ctx, t.ID, resultURL, "对象存储尚未配置，无法完成持久化交付", t.ClaimToken)
	}
	if err := storeClient.EnsureBucket(deliveryCtx); err != nil {
		return r.Store.MarkTaskDeliveryPendingWithToken(ctx, t.ID, resultURL, fmt.Sprintf("对象存储桶不可用: %v", err), t.ClaimToken)
	}
	key := fmt.Sprintf("results/%s/%s.mp4", time.Now().UTC().Format("20060102"), t.PublicID)
	ossURL, err := storeClient.PullURL(deliveryCtx, resultURL, key, "video/mp4")
	if err != nil {
		return r.Store.MarkTaskDeliveryPendingWithToken(ctx, t.ID, resultURL, fmt.Sprintf("对象存储交付失败: %v", err), t.ClaimToken)
	}
	log.Printf("task %s persisted to %s object %s", t.PublicID, storeClient.Provider(), key)
	return r.Store.CompleteTaskUpstreamWithToken(ctx, t.ID, resultURL, ossURL, upstreamDuration, t.ClaimToken)
}
