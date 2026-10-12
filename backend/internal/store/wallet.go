package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type LedgerEntry struct {
	ID     string
	Type   string
	Amount float64
	Desc   string
	Time   time.Time
}

func ledgerTypeCN(t string) string {
	switch t {
	case "task_charge":
		return "任务消费"
	case "task_refund":
		return "任务退款"
	case "redeem":
		return "兑换到账"
	case "recharge":
		return "充值到账"
	case "ticket_refund":
		return "工单退款"
	default:
		return t
	}
}

func (s *Store) ListLedger(ctx context.Context, userID int64, page, pageSize int) ([]LedgerEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.Pool.Query(ctx, `
SELECT public_id, type, amount::float8, description, created_at
FROM wallet_ledger WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		var rawType string
		if err := rows.Scan(&e.ID, &rawType, &e.Amount, &e.Desc, &e.Time); err != nil {
			return nil, 0, err
		}
		e.Type = ledgerTypeCN(rawType)
		items = append(items, e)
	}
	return items, total, rows.Err()
}

func (s *Store) SpentTotals(ctx context.Context, userID int64) (totalSpent, monthSpent float64, err error) {
	err = s.Pool.QueryRow(ctx, `
SELECT
  COALESCE(SUM(CASE WHEN type = 'task_charge' THEN -amount ELSE 0 END), 0)::float8,
  COALESCE(SUM(CASE WHEN type = 'task_charge' AND created_at >= date_trunc('month', NOW()) THEN -amount ELSE 0 END), 0)::float8
FROM wallet_ledger WHERE user_id = $1`, userID).Scan(&totalSpent, &monthSpent)
	return
}

type RechargeProduct struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Bonus  float64 `json:"bonus"`
	Price  float64 `json:"price"`
}

func (s *Store) ListRechargeProducts(ctx context.Context) ([]RechargeProduct, error) {
	rows, err := s.Pool.Query(ctx, `
SELECT public_id, amount::float8, bonus::float8, price::float8
FROM recharge_products WHERE enabled = TRUE ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RechargeProduct
	for rows.Next() {
		var p RechargeProduct
		if err := rows.Scan(&p.ID, &p.Amount, &p.Bonus, &p.Price); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (s *Store) Redeem(ctx context.Context, userID int64, code string) (amount, balance float64, err error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx)

	var codeID int64
	var label string
	var maxUses *int
	var usedCount int
	err = tx.QueryRow(ctx, `
SELECT id, amount::float8, label, max_uses, used_count FROM redeem_codes
WHERE code = $1 AND (expires_at IS NULL OR expires_at > NOW())`, normalized).Scan(&codeID, &amount, &label, &maxUses, &usedCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, errors.New("兑换码无效或已过期")
	}
	if err != nil {
		return 0, 0, err
	}
	if maxUses != nil && usedCount >= *maxUses {
		return 0, 0, errors.New("兑换码已达使用上限")
	}

	var used int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM redeem_code_uses WHERE code_id = $1 AND user_id = $2`, codeID, userID).Scan(&used); err != nil {
		// Fail closed: an unreadable usage record must never look like "unused".
		return 0, 0, err
	}
	if used > 0 {
		return 0, 0, errors.New("该兑换码已经使用过")
	}

	if _, err = tx.Exec(ctx, `INSERT INTO redeem_code_uses (code_id, user_id) VALUES ($1, $2)`, codeID, userID); err != nil {
		return 0, 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE redeem_codes SET used_count = used_count + 1 WHERE id = $1`, codeID); err != nil {
		return 0, 0, err
	}
	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance = balance + $1, version = version + 1, updated_at = NOW() WHERE user_id = $2`, amount, userID); err != nil {
		return 0, 0, err
	}
	if err = tx.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id = $1`, userID).Scan(&balance); err != nil {
		return 0, 0, err
	}
	ledgerID := NewLedgerID()
	if _, err = tx.Exec(ctx, `
INSERT INTO wallet_ledger (public_id, user_id, type, amount, balance_after, ref_type, ref_id, description)
VALUES ($1, $2, 'redeem', $3, $4, 'redeem_code', $5, $6)`,
		ledgerID, userID, amount, balance, normalized, fmt.Sprintf("%s · %s", normalized, label)); err != nil {
		return 0, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return amount, balance, nil
}
