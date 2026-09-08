package store

import (
	"context"
	"encoding/json"
	"time"
)

// AdminAuditLog records an administrator's mutating action without storing
// credentials, access tokens, or other secret input values.
type AdminAuditLog struct {
	ID         string
	ActorName  string
	ActorEmail string
	Action     string
	TargetType string
	TargetID   string
	Details    map[string]any
	CreatedAt  time.Time
}

func (s *Store) AppendAdminAudit(ctx context.Context, actorUserID int64, action, targetType, targetID string, details map[string]any) error {
	raw, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
INSERT INTO admin_audit_logs (public_id, actor_user_id, action, target_type, target_id, details)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
		NewAuditID(), actorUserID, action, targetType, targetID, string(raw))
	return err
}

func (s *Store) ListAdminAuditLogs(ctx context.Context, limit int) ([]AdminAuditLog, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := s.Pool.Query(ctx, `
SELECT l.public_id, u.display_name, u.email, l.action, l.target_type, l.target_id, l.details, l.created_at
FROM admin_audit_logs l
JOIN users u ON u.id=l.actor_user_id
ORDER BY l.created_at DESC, l.id DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminAuditLog, 0)
	for rows.Next() {
		var item AdminAuditLog
		var raw []byte
		if err := rows.Scan(&item.ID, &item.ActorName, &item.ActorEmail, &item.Action, &item.TargetType, &item.TargetID, &raw, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &item.Details)
		items = append(items, item)
	}
	return items, rows.Err()
}
