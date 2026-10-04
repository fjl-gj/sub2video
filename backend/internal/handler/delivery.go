package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/sbu2video/backend/internal/storage/blob"
	"github.com/sbu2video/backend/internal/store"
)

// These bearer URLs are produced after ownership checks and never persisted
// back to the database. A later read always gets a new bounded lifetime.
const deliveryURLTTL = time.Hour

func refreshTaskThumbnails(ctx context.Context, objects blob.Store, tasks []store.Task) {
	for i := range tasks {
		if tasks[i].ThumbURL == "" {
			continue
		}
		fresh, err := objects.RefreshGetURL(ctx, tasks[i].ThumbURL, deliveryURLTTL)
		tasks[i].ThumbURL = ""
		if err == nil {
			tasks[i].ThumbURL = fresh
		}
	}
}

func refreshTaskDelivery(ctx context.Context, objects blob.Store, task *store.ClientTaskDetail) error {
	storedThumb := task.ThumbURL
	task.ThumbURL = ""
	if task.DeliveryStatus != "ready" {
		task.DownloadURL = ""
		return nil
	}
	storedResult := task.DownloadURL
	task.DownloadURL = ""
	task.DeliveryStatus = "not_ready"
	fresh, err := objects.RefreshGetURL(ctx, storedResult, deliveryURLTTL)
	if err != nil {
		return err
	}
	if fresh == "" {
		return errors.New("empty delivery URL")
	}
	task.DownloadURL = fresh
	task.DeliveryStatus = "ready"
	if storedThumb == storedResult {
		task.ThumbURL = fresh
	} else if storedThumb != "" {
		if thumbnail, err := objects.RefreshGetURL(ctx, storedThumb, deliveryURLTTL); err == nil {
			task.ThumbURL = thumbnail
		}
	}
	return nil
}

func refreshTaskDownload(ctx context.Context, objects blob.Store, task *store.ClientTaskDetail) error {
	if task.DeliveryStatus != "ready" {
		task.DownloadURL = ""
		return nil
	}
	filename := strings.TrimSpace(task.Title)
	if filename == "" {
		filename = task.ID
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".mp4") {
		filename += ".mp4"
	}
	storedResult := task.DownloadURL
	task.DownloadURL = ""
	fresh, err := objects.RefreshDownloadURL(ctx, storedResult, filename, deliveryURLTTL)
	if err != nil {
		return err
	}
	if fresh == "" {
		return errors.New("empty download URL")
	}
	task.DownloadURL = fresh
	return nil
}
