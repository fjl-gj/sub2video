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
	"github.com/sbu2video/backend/internal/server"
	"github.com/sbu2video/backend/internal/simulator"
	"github.com/sbu2video/backend/internal/store"
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

	st := store.New(pool)
	if err := st.SeedBasics(ctx); err != nil {
		log.Fatalf("seed: %v", err)
	}
	if err := ensureDemoUser(ctx, st, cfg); err != nil {
		log.Fatalf("seed user: %v", err)
	}

	authSvc := auth.NewService(st, cfg)
	simCtx, simCancel := context.WithCancel(context.Background())
	defer simCancel()
	simulator.Start(simCtx, st, 4*time.Second)

	r := server.NewRouter(cfg, st, authSvc)

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("sub2video api listening on %s", cfg.Addr)
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
	_, user, err := authSvc.Register(ctx, cfg.SeedEmail, cfg.SeedPassword, "林默")
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
