package worker

import (
	"context"
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
	tasks, err := r.Store.ListPendingProviderTasks(ctx, 30)
	if err != nil {
		return err
	}
	routes, err := r.Store.ListRoutes(ctx)
	if err != nil {
		return err
	}

	needSimulate := false
	for i := range tasks {
		t := &tasks[i]
		if t.UpstreamTaskID != "" {
			if err := r.poll(ctx, t); err != nil {
				log.Printf("poll %s: %v", t.PublicID, err)
			}
			continue
		}
		handled, err := r.dispatch(ctx, t, routes)
		if err != nil {
			log.Printf("dispatch %s: %v", t.PublicID, err)
			_ = r.Store.FailTaskUpstream(ctx, t.ID, err.Error())
			continue
		}
		if !handled {
			needSimulate = true
		}
	}
	if needSimulate || len(tasks) == 0 {
		if err := r.Store.AdvanceSimulatedTasks(ctx); err != nil {
			return err
		}
	} else {
		_ = r.Store.AdvanceSimulatedTasks(ctx)
	}
	return nil
}

func (r *Runner) objectStore(ctx context.Context) blob.Store {
	return r.Store.LoadObjectStore(ctx, r.Cfg)
}

func (r *Runner) dispatch(ctx context.Context, t *store.ProviderTask, routes []store.SupplierRoute) (bool, error) {
	if t.QuotePublicID != "" {
		return r.dispatchLockedOffering(ctx, t)
	}
	selected, cost := routing.SelectLowestCost(routes, routing.Request{
		ClientWorkflowID: t.ClientWorkflow,
		SourceType:       t.SourceType,
		DurationSec:      t.DurationSec,
		Resolution:       t.Resolution,
		AspectRatio:      t.AspectRatio,
		Now:              time.Now(),
	})
	if selected == nil {
		return false, nil
	}

	token, err := r.Store.ResolveSupplierToken(ctx, r.Cfg.SecretKey, selected.SupplierCode, r.Cfg.AutoDLToken)
	if err != nil || token == "" {
		return false, nil
	}

	baseURL := selected.SupplierBaseURL
	if baseURL == "" {
		baseURL = r.Cfg.AutoDLBaseURL
	}
	client := autodl.New(baseURL, token, selected.SupplierAuthHeader)

	body := map[string]any{
		"prompt":     t.Prompt,
		"duration":   t.DurationSec,
		"resolution": routing.MapResolutionToUpstream(t.Resolution, t.AspectRatio),
	}

	if !isTextCreation(t.ClientWorkflow) {
		storeClient := r.objectStore(ctx)
		if !storeClient.Configured() {
			log.Printf("skip image workflow %s: object storage not configured", t.PublicID)
			return false, nil
		}
		_ = storeClient.EnsureBucket(ctx)
		if err := r.attachImageURLs(ctx, storeClient, t, body); err != nil {
			return true, err
		}
	}

	_ = r.Store.AttachTaskRoute(ctx, t.ID, store.RouteAttach{
		CarrierCode:        "minimax",
		ModelCode:          selected.ModelCode,
		SupplierCode:       selected.SupplierCode,
		SupplierRouteID:    selected.ID,
		UpstreamWorkflowID: selected.WorkflowID,
		Snapshot: map[string]any{
			"routeId": selected.ID, "workflowId": selected.WorkflowID, "costPerSec": cost,
			"name": selected.Name, "resolution": body["resolution"],
		},
	})
	_, _ = r.Store.Pool.Exec(ctx, `UPDATE generation_tasks SET cost_per_sec=$1 WHERE id=$2`, cost, t.ID)

	res, err := client.Submit(ctx, selected.WorkflowID, body)
	if err != nil {
		return true, err
	}
	if err := r.Store.MarkTaskSubmitted(ctx, t.ID, res.TaskID, res.Status); err != nil {
		return true, err
	}
	log.Printf("submitted %s -> autodl %s via %s", t.PublicID, res.TaskID, selected.WorkflowID)
	return true, nil
}

func (r *Runner) dispatchLockedOffering(ctx context.Context, t *store.ProviderTask) (bool, error) {
	if t.SupplierRouteID == "" || t.UpstreamWorkflowID == "" {
		return true, fmt.Errorf("任务缺少锁定供给")
	}
	if t.AdapterCode != "autodl_h3_v1" {
		return true, fmt.Errorf("不支持的锁定供给适配器: %s", t.AdapterCode)
	}
	token, err := r.Store.ResolveSupplierToken(ctx, r.Cfg.SecretKey, t.SupplierCode, r.Cfg.AutoDLToken)
	if err != nil || token == "" {
		return false, nil
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
	body := map[string]any{
		"prompt":     t.Prompt,
		"duration":   t.DurationSec,
		"resolution": routing.MapResolutionToUpstream(t.Resolution, t.AspectRatio),
	}
	if t.GenerateAudio {
		body["generate_audio"] = true
	}
	if !isTextCreation(t.ClientWorkflow) {
		storeClient := r.objectStore(ctx)
		if !storeClient.Configured() {
			log.Printf("skip image workflow %s: object storage not configured", t.PublicID)
			return false, nil
		}
		_ = storeClient.EnsureBucket(ctx)
		if err := r.attachImageURLs(ctx, storeClient, t, body); err != nil {
			return true, err
		}
	}
	res, err := client.Submit(ctx, t.UpstreamWorkflowID, body)
	if err != nil {
		return true, err
	}
	if err := r.Store.MarkTaskSubmitted(ctx, t.ID, res.TaskID, res.Status); err != nil {
		return true, err
	}
	log.Printf("submitted %s via locked offering %s", t.PublicID, t.SupplierRouteID)
	return true, nil
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
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif" {
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
	default:
		return ".jpg"
	}
}

func (r *Runner) poll(ctx context.Context, t *store.ProviderTask) error {
	token, err := r.Store.ResolveSupplierToken(ctx, r.Cfg.SecretKey, t.SupplierCode, r.Cfg.AutoDLToken)
	if err != nil || token == "" {
		return nil
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
	res, err := client.Query(ctx, t.UpstreamTaskID)
	if err != nil {
		return err
	}
	status := strings.ToUpper(res.Status)
	switch status {
	case "SUCCESS", "COMPLETED", "DONE":
		url := ""
		for _, item := range res.Results {
			if item.URL != "" {
				url = item.URL
				break
			}
		}
		ossURL := ""
		storeClient := r.objectStore(ctx)
		if storeClient.Configured() && url != "" {
			_ = storeClient.EnsureBucket(ctx)
			key := fmt.Sprintf("results/%s/%s.mp4", time.Now().UTC().Format("20060102"), t.PublicID)
			pulled, err := storeClient.PullURL(ctx, url, key, "video/mp4")
			if err != nil {
				log.Printf("object store pull %s: %v (keep ephemeral url)", t.PublicID, err)
			} else {
				ossURL = pulled
				log.Printf("task %s pulled to %s %s", t.PublicID, storeClient.Provider(), ossURL)
			}
		} else {
			log.Printf("task %s success; object storage not configured, keep ephemeral url", t.PublicID)
		}
		return r.Store.CompleteTaskUpstream(ctx, t.ID, url, ossURL, res.Duration)
	case "FAILED", "ERROR", "CANCELLED", "CANCELED":
		return r.Store.FailTaskUpstream(ctx, t.ID, "upstream "+status)
	case "QUEUED", "PENDING":
		return r.Store.UpdateTaskUpstream(ctx, t.ID, res.Status, 12)
	case "RUNNING", "PROCESSING":
		prog := t.Progress
		if prog < 20 {
			prog = 20
		} else if prog < 90 {
			prog += 8
		}
		return r.Store.UpdateTaskUpstream(ctx, t.ID, res.Status, prog)
	default:
		return r.Store.UpdateTaskUpstream(ctx, t.ID, res.Status, t.Progress)
	}
}
