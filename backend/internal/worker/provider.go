package worker

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/provider/autodl"
	"github.com/sbu2video/backend/internal/routing"
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
		// Still run simulator for tasks without token / no route match.
		if err := r.Store.AdvanceSimulatedTasks(ctx); err != nil {
			return err
		}
	} else {
		// Also advance any leftover simulator-only tasks.
		_ = r.Store.AdvanceSimulatedTasks(ctx)
	}
	return nil
}

func (r *Runner) dispatch(ctx context.Context, t *store.ProviderTask, routes []store.SupplierRoute) (bool, error) {
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
	// Image workflows need public URLs; without COS we cannot submit safely yet.
	if t.SourceType == "upload" || t.SourceType == "asset" || strings.Contains(selected.ModelCode, "image") {
		if t.ClientWorkflow != "wf-h3-text" {
			log.Printf("skip image workflow %s: COS/public URL not ready", t.PublicID)
			return false, nil
		}
	}

	_ = r.Store.AttachTaskRoute(ctx, t.ID, store.RouteAttach{
		CarrierCode:        "minimax-h3",
		ModelCode:          selected.ModelCode,
		SupplierCode:       selected.SupplierCode,
		SupplierRouteID:    selected.ID,
		UpstreamWorkflowID: selected.WorkflowID,
		Snapshot: map[string]any{
			"routeId": selected.ID, "workflowId": selected.WorkflowID, "costPerSec": cost,
			"name": selected.Name, "resolution": body["resolution"],
		},
	})

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

func (r *Runner) poll(ctx context.Context, t *store.ProviderTask) error {
	token, err := r.Store.ResolveSupplierToken(ctx, r.Cfg.SecretKey, t.SupplierCode, r.Cfg.AutoDLToken)
	if err != nil || token == "" {
		return nil
	}
	baseURL := r.Cfg.AutoDLBaseURL
	client := autodl.New(baseURL, token, "Authorization")
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
		// COS pull not implemented yet — keep ephemeral URL for now and log.
		log.Printf("task %s success; result url retained (COS pull pending)", t.PublicID)
		return r.Store.CompleteTaskUpstream(ctx, t.ID, url, "", res.Duration)
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
