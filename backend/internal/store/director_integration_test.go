package store

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func directorTaskFixture(t *testing.T, env *integrationStore, userID int64, id, status, result string) {
	t.Helper()
	_, err := env.pool.Exec(env.ctx, `INSERT INTO generation_tasks
(public_id,user_id,title,workflow,status,duration_sec,resolution,cost,result_oss_url)
VALUES ($1,$2,'镜头候选','文生视频',$3,5,'768P',0,$4)`, id, userID, status, result)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDirectorBoardPersistenceAndOwnershipIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	user, other := newIntegrationUser(t, env, 10), newIntegrationUser(t, env, 10)
	project, err := env.store.CreateProject(env.ctx, user.ID, "导演测试", "", false)
	if err != nil {
		t.Fatal(err)
	}
	board, err := env.store.GetDirectorBoard(env.ctx, user.ID, project.ID)
	if err != nil || board.Revision != 0 || len(board.Shots) != 0 {
		t.Fatalf("initial board: %+v %v", board, err)
	}
	if _, err := env.store.GetDirectorBoard(env.ctx, other.ID, project.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user read: %v", err)
	}
	ownedUpload, err := env.store.CreateUpload(env.ctx, user.ID, "", "reference.png", "image/png", "not-read", 10)
	if err != nil {
		t.Fatal(err)
	}
	_, ownedAsset, err := env.store.AddProjectFile(env.ctx, user.ID, project.ID, []string{"资产", "素材"}, FsItem{ID: NewAssetID(), Name: "参考图", Type: "image"})
	if err != nil {
		t.Fatal(err)
	}
	shots := []DirectorShot{{ID: "shot_a", Title: "第一镜头", SourceMeta: SourceMeta{Refs: []SourceRefItem{{Type: "upload", ID: ownedUpload.ID}, {Type: "asset", ID: ownedAsset.ID}}}}, {ID: "shot_b", Title: "第二镜头"}}
	board, err = env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 0, shots)
	if err != nil || board.Revision != 1 {
		t.Fatalf("save: %+v %v", board, err)
	}
	if _, err := env.store.SaveDirectorBoard(env.ctx, other.ID, project.ID, 1, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user write: %v", err)
	}
	// Removed source material must not make the existing board unreadable.
	// Saving that stale reference must fail until the user removes/replaces it.
	if _, err := env.pool.Exec(env.ctx, `DELETE FROM uploads WHERE public_id=$1 AND user_id=$2`, ownedUpload.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	loaded, err := env.store.GetDirectorBoard(env.ctx, user.ID, project.ID)
	if err != nil || len(loaded.Shots[0].SourceMeta.Refs) != 2 {
		t.Fatalf("read with removed upload: %+v %v", loaded, err)
	}
	if _, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 1, loaded.Shots); !errors.Is(err, ErrDirectorReference) {
		t.Fatalf("saving removed upload: %v", err)
	}
	if _, err := env.store.DeleteProjectFile(env.ctx, user.ID, project.ID, ownedAsset.ID); err != nil {
		t.Fatal(err)
	}
	loaded, err = env.store.GetDirectorBoard(env.ctx, user.ID, project.ID)
	if err != nil || len(loaded.Shots[0].SourceMeta.Refs) != 2 {
		t.Fatalf("read with removed asset: %+v %v", loaded, err)
	}
	loaded.Shots[0].SourceMeta.Refs = []SourceRefItem{{Type: "asset", ID: ownedAsset.ID}}
	if _, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 1, loaded.Shots); !errors.Is(err, ErrDirectorReference) {
		t.Fatalf("saving removed asset: %v", err)
	}
	// Concurrent edits must not silently overwrite each other.
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 1, []DirectorShot{{ID: "shot_b"}, {ID: "shot_a"}})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	passed, conflicted := 0, 0
	for err := range errs {
		if err == nil {
			passed++
		} else if errors.Is(err, ErrDirectorConflict) {
			conflicted++
		} else {
			t.Fatal(err)
		}
	}
	if passed != 1 || conflicted != 1 {
		t.Fatalf("concurrent writes: passed=%d conflicted=%d", passed, conflicted)
	}
	board, err = env.store.GetDirectorBoard(env.ctx, user.ID, project.ID)
	if err != nil || board.Revision != 2 || board.Shots[0].ID != "shot_b" {
		t.Fatalf("reload ordered board: %+v %v", board, err)
	}

	foreign, err := env.store.CreateUpload(env.ctx, other.ID, "", "private.png", "image/png", "not-read", 10)
	if err != nil {
		t.Fatal(err)
	}
	board.Shots[0].SourceMeta.First = &SourceRefItem{Type: "upload", ID: foreign.ID}
	if _, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 2, board.Shots); !errors.Is(err, ErrDirectorReference) {
		t.Fatalf("foreign material: %v", err)
	}
	foreignProject, err := env.store.CreateProject(env.ctx, other.ID, "其他用户资产", "", false)
	if err != nil {
		t.Fatal(err)
	}
	_, foreignAsset, err := env.store.AddProjectFile(env.ctx, other.ID, foreignProject.ID, []string{"资产", "素材"}, FsItem{ID: NewAssetID(), Name: "私有素材", Type: "image"})
	if err != nil {
		t.Fatal(err)
	}
	board.Shots[0].SourceMeta.First = &SourceRefItem{Type: "asset", ID: foreignAsset.ID}
	if _, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 2, board.Shots); !errors.Is(err, ErrDirectorReference) {
		t.Fatalf("foreign project asset: %v", err)
	}
	board.Shots[0].SourceMeta = SourceMeta{}
	directorTaskFixture(t, env, other.ID, "tsk_foreign_director", "completed", "https://store.example/video.mp4")
	if _, err := env.store.AddDirectorCandidate(env.ctx, user.ID, project.ID, "shot_b", "tsk_foreign_director"); !errors.Is(err, ErrDirectorTask) {
		t.Fatalf("foreign candidate: %v", err)
	}
	directorTaskFixture(t, env, user.ID, "tsk_waiting_director", "running", "")
	board, err = env.store.AddDirectorCandidate(env.ctx, user.ID, project.ID, "shot_b", "tsk_waiting_director")
	if err != nil || board.Revision != 3 {
		t.Fatalf("candidate append: %+v %v", board, err)
	}
	replay, err := env.store.AddDirectorCandidate(env.ctx, user.ID, project.ID, "shot_b", "tsk_waiting_director")
	if err != nil || replay.Revision != 3 || len(replay.Shots[0].CandidateTaskIDs) != 1 {
		t.Fatalf("candidate replay: %+v %v", replay, err)
	}
	board.Shots[0].SelectedTaskID = "tsk_waiting_director"
	if _, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 3, board.Shots); !errors.Is(err, ErrDirectorSelection) {
		t.Fatalf("premature selection: %v", err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE generation_tasks SET status='completed',result_oss_url='' WHERE public_id='tsk_waiting_director'`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 3, board.Shots); !errors.Is(err, ErrDirectorSelection) {
		t.Fatalf("completed without stored result: %v", err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE generation_tasks SET status='completed',result_oss_url='https://store.example/video.mp4' WHERE public_id='tsk_waiting_director'`); err != nil {
		t.Fatal(err)
	}
	board, err = env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 3, board.Shots)
	if err != nil || board.Revision != 4 || board.Shots[0].SelectedTaskID != "tsk_waiting_director" {
		t.Fatalf("delivered selection: %+v %v", board, err)
	}
	// A candidate arriving during a whole-board edit must either survive the
	// edit or make that stale edit fail, never disappear after both return.
	directorTaskFixture(t, env, user.ID, "tsk_late_director", "queued", "")
	writeResult, appendResult := make(chan error, 1), make(chan error, 1)
	go func() {
		_, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 4, board.Shots)
		writeResult <- err
	}()
	go func() {
		_, err := env.store.AddDirectorCandidate(env.ctx, user.ID, project.ID, "shot_b", "tsk_late_director")
		appendResult <- err
	}()
	if err := <-writeResult; err != nil && !errors.Is(err, ErrDirectorConflict) {
		t.Fatal(err)
	}
	if err := <-appendResult; err != nil {
		t.Fatal(err)
	}
	board, err = env.store.GetDirectorBoard(env.ctx, user.ID, project.ID)
	if err != nil || len(board.Shots[0].CandidateTaskIDs) != 2 {
		t.Fatalf("concurrent edit lost candidate: %+v %v", board, err)
	}
	if _, err := env.store.AddDirectorCandidate(env.ctx, user.ID, project.ID, "deleted_shot", "tsk_waiting_director"); !errors.Is(err, ErrDirectorShotMissing) {
		t.Fatalf("deleted shot: %v", err)
	}
	if _, err := env.store.TrashProject(env.ctx, user.ID, project.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.GetDirectorBoard(env.ctx, user.ID, project.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("trashed read: %v", err)
	}
	if _, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, board.Revision, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("trashed write: %v", err)
	}
}

func directorFinancialSnapshot(t *testing.T, env *integrationStore, userID int64) string {
	t.Helper()
	var balance float64
	var tasks, orders, ledger int
	err := env.pool.QueryRow(env.ctx, `SELECT (SELECT balance::float8 FROM wallets WHERE user_id=$1),
(SELECT count(*) FROM generation_tasks WHERE user_id=$1),(SELECT count(*) FROM consumption_orders WHERE user_id=$1),(SELECT count(*) FROM wallet_ledger WHERE user_id=$1)`, userID).Scan(&balance, &tasks, &orders, &ledger)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%.2f/%d/%d/%d", balance, tasks, orders, ledger)
}

func assertDirectorCharges(t *testing.T, env *integrationStore, userID int64, expectedCount int, expectedAmount float64) {
	t.Helper()
	var balanceCents, orderCents, ledgerCents, taskCents int64
	var tasks, orders, charges, mismatches int
	err := env.pool.QueryRow(env.ctx, `SELECT
  (SELECT (balance*100)::bigint FROM wallets WHERE user_id=$1),
  (SELECT COUNT(*) FROM generation_tasks WHERE user_id=$1),
  (SELECT (COALESCE(SUM(cost),0)*100)::bigint FROM generation_tasks WHERE user_id=$1),
  (SELECT COUNT(*) FROM consumption_orders WHERE user_id=$1),
  (SELECT (COALESCE(SUM(amount),0)*100)::bigint FROM consumption_orders WHERE user_id=$1),
  (SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1 AND type='task_charge'),
  (SELECT (-COALESCE(SUM(amount),0)*100)::bigint FROM wallet_ledger WHERE user_id=$1 AND type='task_charge'),
  (SELECT COUNT(*) FROM consumption_orders o LEFT JOIN generation_tasks t ON t.id=o.task_id
   WHERE o.user_id=$1 AND (t.id IS NULL OR t.user_id<>o.user_id OR t.cost<>o.amount OR o.status<>'paid'
     OR (SELECT COUNT(*) FROM wallet_ledger l WHERE l.user_id=o.user_id AND l.type='task_charge' AND l.ref_type='order' AND l.ref_id=o.public_id AND l.amount=-o.amount)<>1))`, userID).
		Scan(&balanceCents, &tasks, &taskCents, &orders, &orderCents, &charges, &ledgerCents, &mismatches)
	if err != nil {
		t.Fatal(err)
	}
	wantCents := int64(math.Round(expectedAmount * 100))
	if tasks != expectedCount || orders != expectedCount || charges != expectedCount || mismatches != 0 || orderCents != wantCents || ledgerCents != wantCents || taskCents != wantCents || balanceCents != 1000-wantCents {
		t.Fatalf("director financial mismatch: tasks=%d orders=%d charges=%d brokenOrders=%d balanceCents=%d taskCents=%d orderCents=%d ledgerCents=%d wantCount=%d wantChargeCents=%d", tasks, orders, charges, mismatches, balanceCents, taskCents, orderCents, ledgerCents, expectedCount, wantCents)
	}
}

func TestDirectorTaskBindingAtomicAndIdempotentIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	user := newIntegrationUser(t, env, 10)
	project, err := env.store.CreateProject(env.ctx, user.ID, "镜头绑定", "", false)
	if err != nil {
		t.Fatal(err)
	}
	board, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 0, []DirectorShot{{ID: "shot_one", Title: "首镜"}})
	if err != nil {
		t.Fatal(err)
	}
	textMode, _, _, modelID := testCatalogIDs(t, env)
	in := DirectTaskInput{DirectRouteRequest: seededDirectRequest(textMode, modelID), ClientRequestID: "director_atomic", Prompt: "山路缓缓延伸到海边", DirectorContext: &DirectorContext{ProjectID: project.ID, ShotID: "shot_one", ExpectedRevision: board.Revision}}
	before := directorFinancialSnapshot(t, env, user.ID)
	in.DirectorContext.ShotID = "missing_shot"
	if _, err := env.store.CreateTaskDirect(env.ctx, user.ID, in); !errors.Is(err, ErrDirectorShotMissing) {
		t.Fatalf("missing shot: %v", err)
	}
	if after := directorFinancialSnapshot(t, env, user.ID); after != before {
		t.Fatalf("missing shot changed financial state: %s != %s", after, before)
	}
	in.DirectorContext.ShotID = "shot_one"
	other := newIntegrationUser(t, env, 10)
	foreignProject, err := env.store.CreateProject(env.ctx, other.ID, "他人镜头", "", false)
	if err != nil {
		t.Fatal(err)
	}
	foreignBoard, err := env.store.SaveDirectorBoard(env.ctx, other.ID, foreignProject.ID, 0, []DirectorShot{{ID: "shot_one"}})
	if err != nil {
		t.Fatal(err)
	}
	trashedProject, err := env.store.CreateProject(env.ctx, user.ID, "回收镜头", "", false)
	if err != nil {
		t.Fatal(err)
	}
	trashedBoard, err := env.store.SaveDirectorBoard(env.ctx, user.ID, trashedProject.ID, 0, []DirectorShot{{ID: "shot_one"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.TrashProject(env.ctx, user.ID, trashedProject.ID); err != nil {
		t.Fatal(err)
	}
	otherBefore := directorFinancialSnapshot(t, env, other.ID)
	for _, inaccessible := range []*DirectorBoard{foreignBoard, trashedBoard} {
		request := in
		request.DirectorContext = &DirectorContext{ProjectID: inaccessible.ProjectID, ShotID: "shot_one", ExpectedRevision: inaccessible.Revision}
		if _, err := env.store.CreateTaskDirect(env.ctx, user.ID, request); !errors.Is(err, ErrNotFound) {
			t.Fatalf("inaccessible director context: %v", err)
		}
		if after := directorFinancialSnapshot(t, env, user.ID); after != before {
			t.Fatalf("inaccessible context charged caller: %s != %s", after, before)
		}
		if after := directorFinancialSnapshot(t, env, other.ID); after != otherBefore {
			t.Fatalf("inaccessible context changed other user: %s != %s", after, otherBefore)
		}
	}
	// Inject a board-write failure after task/order/charge statements have run.
	if _, err := env.pool.Exec(env.ctx, `CREATE FUNCTION reject_director_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture board write rejected'; END $$;
CREATE TRIGGER reject_director_write BEFORE UPDATE OF director_board ON projects FOR EACH ROW EXECUTE FUNCTION reject_director_write()`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.CreateTaskDirect(env.ctx, user.ID, in); err != nil {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "P0001" || pgErr.Message != "fixture board write rejected" {
			t.Fatalf("expected injected board-write failure, got: %v", err)
		}
	} else {
		t.Fatal("injected board failure must abort task creation")
	}
	if after := directorFinancialSnapshot(t, env, user.ID); after != before {
		t.Fatalf("partial financial write: %s != %s", after, before)
	}
	if _, err := env.pool.Exec(env.ctx, `DROP TRIGGER reject_director_write ON projects`); err != nil {
		t.Fatal(err)
	}
	type taskOutcome struct {
		result *DirectTaskResult
		err    error
	}
	outcomes := make(chan taskOutcome, 2)
	for i := 0; i < 2; i++ {
		go func() {
			result, err := env.store.CreateTaskDirect(env.ctx, user.ID, in)
			outcomes <- taskOutcome{result, err}
		}()
	}
	var created *DirectTaskResult
	replays := 0
	for i := 0; i < 2; i++ {
		outcome := <-outcomes
		if outcome.err != nil {
			t.Fatal(outcome.err)
		}
		if created != nil && (outcome.result.Task.ID != created.Task.ID || outcome.result.OrderID != created.OrderID) {
			t.Fatal("concurrent idempotency created two tasks")
		}
		created = outcome.result
		if created.AlreadyDone {
			replays++
		}
	}
	if replays != 1 {
		t.Fatalf("expected one original and one replay, got %d replays", replays)
	}
	assertDirectorCharges(t, env, user.ID, 1, created.Task.Cost)
	board, err = env.store.GetDirectorBoard(env.ctx, user.ID, project.ID)
	if err != nil || board.Revision != 2 || len(board.Shots[0].CandidateTaskIDs) != 1 || board.Shots[0].CandidateTaskIDs[0] != created.Task.ID || board.Shots[0].Prompt != in.Prompt || board.Shots[0].ModelID != modelID {
		t.Fatalf("atomic board binding: %+v %v", board, err)
	}
	// Different paid requests from the same revision must not both submit.
	for i := 0; i < 2; i++ {
		request := in
		request.ClientRequestID = fmt.Sprintf("director_concurrent_%d", i)
		request.DirectorContext = &DirectorContext{ProjectID: project.ID, ShotID: "shot_one", ExpectedRevision: 2}
		go func(input DirectTaskInput) {
			result, err := env.store.CreateTaskDirect(env.ctx, user.ID, input)
			outcomes <- taskOutcome{result, err}
		}(request)
	}
	succeeded, conflicted := 0, 0
	totalCharged := created.Task.Cost
	for i := 0; i < 2; i++ {
		outcome := <-outcomes
		if outcome.err == nil {
			succeeded++
			totalCharged += outcome.result.Task.Cost
		} else if errors.Is(outcome.err, ErrDirectorConflict) {
			conflicted++
		} else {
			t.Fatal(outcome.err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("same revision submissions: success=%d conflict=%d", succeeded, conflicted)
	}
	assertDirectorCharges(t, env, user.ID, 2, totalCharged)
	board, err = env.store.GetDirectorBoard(env.ctx, user.ID, project.ID)
	if err != nil || board.Revision != 3 || len(board.Shots[0].CandidateTaskIDs) != 2 {
		t.Fatalf("concurrent submit board: %+v %v", board, err)
	}
	charged := directorFinancialSnapshot(t, env, user.ID)
	if _, err := env.store.SaveDirectorBoard(env.ctx, user.ID, project.ID, 3, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE suppliers SET enabled=FALSE`); err != nil {
		t.Fatal(err)
	}
	replay, err := env.store.CreateTaskDirect(env.ctx, user.ID, in)
	if err != nil || !replay.AlreadyDone || replay.Task.ID != created.Task.ID || replay.OrderID != created.OrderID {
		t.Fatalf("replay after shot deletion and supply shutdown: %+v %v", replay, err)
	}
	if after := directorFinancialSnapshot(t, env, user.ID); after != charged {
		t.Fatalf("replay recharged: %s != %s", after, charged)
	}
	assertDirectorCharges(t, env, user.ID, 2, totalCharged)
}
