package store

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestImageSlotsRejectAudioUploadsBeforeChargeIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	_, keyframeMode, multiMode, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 100)
	audio, err := env.store.CreateUpload(env.ctx, user.ID, "", "voice.mp3", "audio/mpeg", "fixture-audio-not-read", 10)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := env.store.CreateUpload(env.ctx, user.ID, "", "reference.png", "image/png", "fixture-image-not-read", 10)
	if err != nil {
		t.Fatal(err)
	}
	audioRef := SourceRefItem{Type: "upload", ID: audio.ID}
	pictureRef := SourceRefItem{Type: "upload", ID: picture.ID}
	// These are the user's own uploads. The rejection must be by media type,
	// before task/order/ledger creation or a wallet change, not by ownership.
	for _, tc := range []struct {
		name, mode string
		meta       SourceMeta
	}{
		{"first_frame", keyframeMode, SourceMeta{First: &audioRef, Last: &pictureRef}},
		{"last_frame", keyframeMode, SourceMeta{First: &pictureRef, Last: &audioRef}},
		{"multi_reference", multiMode, SourceMeta{Refs: []SourceRefItem{pictureRef, audioRef}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := seededDirectRequest(tc.mode, modelID)
			request.ImageCount = tc.meta.ImageCount()
			if _, err := env.store.ResolveDirectRoute(env.ctx, request); err != nil {
				t.Fatalf("fixture must otherwise have an available route: %v", err)
			}
			_, err := env.store.CreateTaskDirect(env.ctx, user.ID, DirectTaskInput{
				DirectRouteRequest: request, ClientRequestID: "audio-in-image-" + tc.name,
				Prompt: "图片素材类型检查", SourceMeta: tc.meta,
			})
			if err == nil || !strings.Contains(err.Error(), "请选择图片素材") {
				t.Fatalf("own audio upload must be rejected as an image before charging: %v", err)
			}
			var balance float64
			var tasks, orders, entries int
			if err := env.pool.QueryRow(env.ctx, `SELECT
(SELECT balance::float8 FROM wallets WHERE user_id=$1),
(SELECT COUNT(*) FROM generation_tasks WHERE user_id=$1),
(SELECT COUNT(*) FROM consumption_orders WHERE user_id=$1),
(SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1)`, user.ID).Scan(&balance, &tasks, &orders, &entries); err != nil {
				t.Fatal(err)
			}
			if balance != 100 || tasks != 0 || orders != 0 || entries != 0 {
				t.Fatalf("invalid image input changed financial state: balance=%v tasks=%d orders=%d ledger=%d", balance, tasks, orders, entries)
			}
		})
	}
	if err := env.store.validateReferenceAudioSource(env.ctx, user.ID, &audioRef); err != nil {
		t.Fatalf("same upload remains valid in the dedicated reference-audio slot: %v", err)
	}
	if err := env.store.validateSourceRefItem(env.ctx, user.ID, &pictureRef); err != nil {
		t.Fatalf("valid image upload must remain accepted: %v", err)
	}
}

func TestClientTaskActionsMatchCancellationIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 100)
	for _, tc := range []struct {
		name, status, upstreamStatus, upstreamID string
		started, uncertain, canCancel, review    bool
	}{
		{name: "queued", status: "queued", canCancel: true},
		{name: "waiting_for_configuration", status: "running", upstreamStatus: "WAITING_CONFIGURATION", canCancel: true},
		{name: "submitting", status: "running", started: true},
		{name: "accepted", status: "running", upstreamID: "upstream-private"},
		{name: "unknown", status: "running", upstreamStatus: "SUBMIT_UNKNOWN", uncertain: true, review: true},
		{name: "uncertain_flag", status: "running", uncertain: true, review: true},
		{name: "completed", status: "completed"},
		{name: "failed", status: "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			created, err := env.store.CreateTaskDirect(env.ctx, user.ID, DirectTaskInput{
				DirectRouteRequest: seededDirectRequest(textMode, modelID),
				ClientRequestID:    "task-actions-" + tc.name, Prompt: "动作边界测试",
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := env.pool.Exec(env.ctx, `UPDATE generation_tasks
SET execution_mode='provider',status=$2,upstream_status=$3,upstream_task_id=$4,submit_started=$5,submit_uncertain=$6
WHERE public_id=$1`, created.Task.ID, tc.status, tc.upstreamStatus, tc.upstreamID, tc.started, tc.uncertain); err != nil {
				t.Fatal(err)
			}
			detail, err := env.store.GetClientTaskDetail(env.ctx, user.ID, created.Task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if detail.CanCancel != tc.canCancel || detail.ReviewRequired != tc.review {
				t.Fatalf("detail actions got cancel=%v review=%v, want cancel=%v review=%v", detail.CanCancel, detail.ReviewRequired, tc.canCancel, tc.review)
			}
			items, _, _, err := env.store.ListTasks(env.ctx, user.ID, "全部", 1, 100)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, item := range items {
				if item.ID == created.Task.ID {
					found = true
					if item.CanCancel != detail.CanCancel || item.ReviewRequired != detail.ReviewRequired {
						t.Fatalf("list/detail actions disagree: list=%+v detail=%+v", item, detail.Task)
					}
				}
			}
			if !found {
				t.Fatal("created task not found in list")
			}
			_, _, _, err = env.store.CancelTask(env.ctx, user.ID, created.Task.ID)
			if (err == nil) != tc.canCancel {
				t.Fatalf("displayed cancellation eligibility disagrees with transaction: allowed=%v err=%v", tc.canCancel, err)
			}
		})
	}
}

func TestClientTaskSavedDestinationSurvivesReloadIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	user := newIntegrationUser(t, env, 100)
	other := newIntegrationUser(t, env, 100)
	project, err := env.store.CreateProject(env.ctx, user.ID, "保存目标", "", false)
	if err != nil {
		t.Fatal(err)
	}
	const taskID = "tsk_saved_destination"
	if _, err := env.pool.Exec(env.ctx, `INSERT INTO generation_tasks
(public_id,user_id,title,workflow,status,progress,duration_sec,resolution,cost,result_oss_url)
VALUES ($1,$2,'目标镜头','文生视频','completed',100,5,'768P',1,'https://example.test/video.mp4')`, taskID, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := env.store.SaveTaskToProject(env.ctx, user.ID, taskID, project.ID); err != nil {
		t.Fatal(err)
	}
	want := &TaskSavedDestination{ProjectID: project.ID, FolderPath: []string{"视频", "正片镜头"}}
	for i := 0; i < 2; i++ {
		detail, err := env.store.GetClientTaskDetail(env.ctx, user.ID, taskID)
		if err != nil || !reflect.DeepEqual(detail.SavedDestination, want) {
			t.Fatalf("reload must recover actual saved destination: detail=%+v err=%v", detail, err)
		}
	}
	if _, err := env.store.GetClientTaskDetail(env.ctx, other.ID, taskID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other account must not read saved destination: %v", err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE projects SET trashed_at=NOW() WHERE public_id=$1`, project.ID); err != nil {
		t.Fatal(err)
	}
	detail, err := env.store.GetClientTaskDetail(env.ctx, user.ID, taskID)
	if err != nil || detail.SavedDestination != nil {
		t.Fatalf("trashed project must not be presented as an active destination: detail=%+v err=%v", detail, err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE projects SET trashed_at=NULL, folders_json='{}'::jsonb WHERE public_id=$1`, project.ID); err != nil {
		t.Fatal(err)
	}
	detail, err = env.store.GetClientTaskDetail(env.ctx, user.ID, taskID)
	if err != nil || detail.SavedDestination != nil {
		t.Fatalf("removed saved file must not retain a stale destination: detail=%+v err=%v", detail, err)
	}
}

func TestRetailPriceInsertAndUpdateRemainModeScopedIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	input := RetailPriceRule{ModeID: textMode, ModelID: modelID, Resolution: "2160P", Name: "新售价", PricePerSecond: 0.5}
	created, err := env.store.UpsertRetailPriceRule(env.ctx, input)
	if err != nil {
		t.Fatalf("insert fresh dimension with explicit NULL model id: %v", err)
	}
	if created.ModelID != "" || created.ModelCode != "" || created.CarrierName != "" {
		t.Fatalf("platform price must not acquire a model dimension: %+v", created)
	}
	input.PricePerSecond = 0.6
	updated, err := env.store.UpsertRetailPriceRule(env.ctx, input)
	if err != nil || updated.ID != created.ID || updated.PricePerSecond != 0.6 {
		t.Fatalf("same pricing dimensions must update original rule: updated=%+v err=%v", updated, err)
	}
	var count int
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM retail_price_rules WHERE public_id=$1 AND model_id IS NULL`, created.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("persisted model id must be SQL NULL: count=%d err=%v", count, err)
	}
}

func TestAdminTaskSearchPaginatesAcrossAllMatchingRowsIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	user := newIntegrationUser(t, env, 100)
	if _, err := env.pool.Exec(env.ctx, `INSERT INTO generation_tasks
(public_id,user_id,title,workflow,status,progress,duration_sec,resolution,cost)
SELECT 'tsk_search_'||n,$1,'可检索镜头 '||n,'文生视频','queued',0,5,'768P',1
FROM generate_series(1,51) AS n`, user.ID); err != nil {
		t.Fatal(err)
	}
	items, total, err := env.store.ListAdminTasks(env.ctx, "排队中", "可检索镜头", 2, 50)
	if err != nil || total != 51 || len(items) != 1 {
		t.Fatalf("second page must remain reachable: total=%d items=%d err=%v", total, len(items), err)
	}
	for _, keyword := range []string{"tsk_search_51", "可检索镜头 51", user.Email} {
		items, total, err := env.store.ListAdminTasks(env.ctx, "排队中", keyword, 1, 100)
		if err != nil || total < 1 || len(items) < 1 {
			t.Fatalf("server search must reach matching task for %q: total=%d err=%v", keyword, total, err)
		}
	}
	items, total, err = env.store.ListAdminTasks(env.ctx, "失败", "可检索镜头", 1, 100)
	if err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("search must retain status filter: total=%d err=%v", total, err)
	}
	items, total, err = env.store.ListAdminTasks(env.ctx, "全部", "%", 1, 100)
	if err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("search treats user wildcard characters as literals: total=%d err=%v", total, err)
	}
}
