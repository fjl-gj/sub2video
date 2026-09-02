package simulator

import (
	"context"
	"log"
	"time"

	"github.com/sbu2video/backend/internal/store"
)

func Start(ctx context.Context, st *store.Store, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := st.AdvanceTasks(ctx); err != nil {
					log.Printf("task simulator: %v", err)
				}
			}
		}
	}()
}
