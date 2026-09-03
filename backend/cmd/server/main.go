package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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
		if cfg.COSSecretID == "" || cfg.COSBucket == "" {
			log.Printf("cos not configured: image workflows stay on simulator until credentials arrive")
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
	if _, err = st.Pool.Exec(ctx, `UPDATE wallets SET balance = 256 WHERE user_id = $1`, user.ID); err != nil {
		return err
	}
	log.Printf("seeded demo user %s / %s (balance 256)", cfg.SeedEmail, cfg.SeedPassword)
	return nil
}
