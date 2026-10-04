package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sbu2video/backend/internal/storage/blob"
	"github.com/sbu2video/backend/internal/store"
)

type deliveryObjects struct {
	blob.Store
	refresh  func(context.Context, string, time.Duration) (string, error)
	download func(context.Context, string, string, time.Duration) (string, error)
}

func (s deliveryObjects) RefreshGetURL(ctx context.Context, raw string, ttl time.Duration) (string, error) {
	return s.refresh(ctx, raw, ttl)
}

func (s deliveryObjects) RefreshDownloadURL(ctx context.Context, raw, filename string, ttl time.Duration) (string, error) {
	return s.download(ctx, raw, filename, ttl)
}

func TestTaskDownloadRequiresAttachmentSignerAndPreservesPreview(t *testing.T) {
	for _, fails := range []bool{false, true} {
		task := &store.ClientTaskDetail{
			Task:           store.Task{ID: "tsk_download", Title: "海边镜头", ThumbURL: "preview"},
			DeliveryStatus: "ready", DownloadURL: "preview",
		}
		objects := deliveryObjects{download: func(_ context.Context, raw, filename string, ttl time.Duration) (string, error) {
			if raw != "preview" || filename != "海边镜头.mp4" || ttl != deliveryURLTTL {
				t.Fatal("download signer lost the authorized object, name or bounded lifetime")
			}
			if fails {
				return "", errors.New("signing unavailable")
			}
			return "attachment", nil
		}}
		err := refreshTaskDownload(context.Background(), objects, task)
		if fails && (err == nil || task.DownloadURL != "") {
			t.Fatal("failed attachment signing must not fall back to inline preview")
		}
		if !fails && (err != nil || task.DownloadURL != "attachment") {
			t.Fatal("successful download must use the attachment URL")
		}
		if task.ThumbURL != "preview" {
			t.Fatal("download signing must not turn the preview into an attachment")
		}
	}
	if err := refreshTaskDownload(context.Background(), deliveryObjects{}, &store.ClientTaskDetail{DeliveryStatus: "not_ready", DownloadURL: "unavailable"}); err != nil {
		t.Fatal("unfinished task should have no download without consulting signer")
	}
}

func TestTaskDeliveryReplacesExpiredURLWithoutChangingTask(t *testing.T) {
	const oldURL = "https://storage.test/bucket/result.mp4?expired=1"
	const newURL = "https://storage.test/bucket/result.mp4?fresh=1"
	task := &store.ClientTaskDetail{
		Task:           store.Task{ID: "tsk_owned", Status: "已完成", Cost: 2, ThumbURL: oldURL},
		DeliveryStatus: "ready", DownloadURL: oldURL,
	}
	calls := 0
	objects := deliveryObjects{refresh: func(ctx context.Context, raw string, ttl time.Duration) (string, error) {
		calls++
		if raw != oldURL || ttl != time.Hour || ctx == nil {
			t.Fatal("unexpected signing request")
		}
		return newURL, nil
	}}
	if err := refreshTaskDelivery(context.Background(), objects, task); err != nil {
		t.Fatal(err)
	}
	if task.DownloadURL != newURL || task.ThumbURL != newURL || task.DeliveryStatus != "ready" || calls != 1 {
		t.Fatalf("delivery was not refreshed consistently: calls=%d", calls)
	}
	if task.ID != "tsk_owned" || task.Status != "已完成" || task.Cost != 2 {
		t.Fatal("refresh must not change task or billing state")
	}
}

func TestTaskDeliveryFailureNeverReturnsStoredSignature(t *testing.T) {
	for _, failure := range []string{"signing failed", "empty result"} {
		t.Run(failure, func(t *testing.T) {
			task := &store.ClientTaskDetail{
				Task:           store.Task{Status: "已完成", ThumbURL: "expired-private-url"},
				DeliveryStatus: "ready", DownloadURL: "expired-private-url",
			}
			objects := deliveryObjects{refresh: func(context.Context, string, time.Duration) (string, error) {
				if failure == "empty result" {
					return "", nil
				}
				return "", errors.New("signer secret must stay internal")
			}}
			if err := refreshTaskDelivery(context.Background(), objects, task); err == nil {
				t.Fatal("failed signing must be reported")
			}
			if task.DownloadURL != "" || task.ThumbURL != "" || task.DeliveryStatus == "ready" {
				t.Fatal("failed signing retained a delivery URL or ready state")
			}
		})
	}
}

func TestPendingTaskDoesNotSignOrExposeResult(t *testing.T) {
	task := &store.ClientTaskDetail{
		Task:           store.Task{ThumbURL: "stored-url"},
		DeliveryStatus: "pending_storage", DownloadURL: "stored-url",
	}
	objects := deliveryObjects{refresh: func(context.Context, string, time.Duration) (string, error) {
		t.Fatal("pending delivery must not sign")
		return "", nil
	}}
	if err := refreshTaskDelivery(context.Background(), objects, task); err != nil || task.DownloadURL != "" || task.ThumbURL != "" || task.DeliveryStatus != "pending_storage" {
		t.Fatal("pending delivery changed or exposed an address")
	}
}

func TestThumbnailFailureDoesNotBreakTaskListOrLeakURL(t *testing.T) {
	tasks := []store.Task{{ID: "bad", ThumbURL: "expired"}, {ID: "good", ThumbURL: "owned"}, {ID: "empty"}}
	objects := deliveryObjects{refresh: func(_ context.Context, raw string, _ time.Duration) (string, error) {
		if raw == "owned" {
			return "fresh", nil
		}
		return "", errors.New("unavailable")
	}}
	refreshTaskThumbnails(context.Background(), objects, tasks)
	if len(tasks) != 3 || tasks[0].ThumbURL != "" || tasks[1].ThumbURL != "fresh" || tasks[2].ThumbURL != "" {
		t.Fatal("list must retain tasks while omitting unavailable thumbnails")
	}
}

func TestAssetDTOKeepsDownloadBehindOwnershipCheck(t *testing.T) {
	raw, err := json.Marshal(fileDTO(store.FsItem{ID: "file_owned", RemoteURL: "https://storage.test/private?old-signature=secret"}, "prj_owned"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "storage.test") || strings.Contains(string(raw), "secret") {
		t.Fatal("asset DTO exposed a persisted URL")
	}
	if !strings.Contains(string(raw), `"remoteUrl":"/api/v1/client/projects/prj_owned/files/file_owned"`) {
		t.Fatal("asset remote URL must resolve through protected download")
	}
}
