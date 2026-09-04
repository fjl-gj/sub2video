package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/db"
	"github.com/sbu2video/backend/internal/mail"
	"github.com/sbu2video/backend/internal/redisx"
	"github.com/sbu2video/backend/internal/server"
	"github.com/sbu2video/backend/internal/store"
	"github.com/sbu2video/backend/internal/worker"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := store.EnsureUploadDir(cfg.UploadDir); err != nil {
		log.Fatalf("upload dir: %v", err)
	}

	rdb, err := redisx.Connect(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	st := store.New(pool)
	if err := st.SeedBasics(ctx); err != nil {
		log.Fatalf("seed: %v", err)
	}
	if err := st.SeedSupplyChain(ctx, cfg.SecretKey); err != nil {
		log.Fatalf("seed supply: %v", err)
	}
	if err := st.SeedPlatformCommerce(ctx); err != nil {
		log.Fatalf("seed commerce: %v", err)
	}
	if err := st.SeedCreationCatalog(ctx); err != nil {
		log.Fatalf("seed creation catalog: %v", err)
	}
	if err := ensureDemoUser(ctx, st, cfg); err != nil {
		log.Fatalf("seed user: %v", err)
	}
	if cfg.PromoteAdminEmail != "" {
		if err := st.PromoteUserRole(ctx, cfg.PromoteAdminEmail, "admin"); err != nil {
			log.Printf("promote admin %s: %v", cfg.PromoteAdminEmail, err)
		}
	}

	mailSvc := &mail.Service{Store: st, Redis: rdb}
	authSvc := auth.NewService(st, cfg)
	authSvc.Mail = mailSvc

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	worker.Start(workerCtx, st, cfg, 4*time.Second)

	r := server.NewRouter(cfg, st, authSvc, mailSvc)

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("sub2video api listening on %s", cfg.Addr)
		if cfg.AutoDLToken == "" {
			log.Printf("autodl token empty: text tasks fall back to simulator until admin/env configures token")
		}
		obj := st.LoadObjectStore(ctx, cfg)
		if !obj.Configured() {
			log.Printf("object storage not configured: image workflows stay on simulator until minio/cos credentials arrive")
		} else if err := obj.EnsureBucket(ctx); err != nil {
			log.Printf("object storage ensure bucket: %v", err)
		} else {
			log.Printf("object storage ready: provider=%s", obj.Provider())
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func ensureDemoUser(ctx context.Context, st *store.Store, cfg config.Config) error {
	n, err := st.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	authSvc := auth.NewService(st, cfg)
	_, user, err := authSvc.Register(ctx, cfg.SeedEmail, cfg.SeedPassword, "林默", "")
	if err != nil {
		return err
	}
	if _, err = st.Pool.Exec(ctx, `UPDATE workspaces SET name = '李维工作室' WHERE user_id = $1`, user.ID); err != nil {
		return err
	}
	initBal := 256.0
	if v, _ := st.GetSetting(ctx, "initial_balance"); v != "" {
		if f, err2 := strconv.ParseFloat(v, 64); err2 == nil && f >= 0 {
			initBal = f
		}
	}
	if _, err = st.Pool.Exec(ctx, `UPDATE wallets SET balance = $1 WHERE user_id = $2`, initBal, user.ID); err != nil {
		return err
	}
	log.Printf("seeded demo user %s / %s (balance %.2f)", cfg.SeedEmail, cfg.SeedPassword, initBal)
	return nil
}
