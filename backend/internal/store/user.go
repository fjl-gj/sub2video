package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}

type User struct {
	ID              int64
	PublicID        string
	Email           string
	PasswordHash    string
	DisplayName     string
	Role            string
	Status          string
	TokenVersion    int
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (u *User) IsActive() bool { return u.Status == "active" }

type Workspace struct {
	ID     int64
	UserID int64
	Name   string
}

func (s *Store) CreateUserWithWallet(ctx context.Context, email, passwordHash, displayName, workspaceName string, balance float64) (*User, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	publicID := "u_" + shortID()
	var user User
	err = tx.QueryRow(ctx, `
INSERT INTO users (public_id, email, password_hash, display_name, role, status)
VALUES ($1, $2, $3, $4, 'user', 'active')
RETURNING id, public_id, email, password_hash, display_name, role, status, token_version, email_verified_at, created_at, updated_at
`, publicID, email, passwordHash, displayName).Scan(
		&user.ID, &user.PublicID, &user.Email, &user.PasswordHash, &user.DisplayName,
		&user.Role, &user.Status, &user.TokenVersion, &user.EmailVerifiedAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if _, err = tx.Exec(ctx, `INSERT INTO workspaces (user_id, name) VALUES ($1, $2)`, user.ID, workspaceName); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO wallets (user_id, balance) VALUES ($1, $2)`, user.ID, balance); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO user_drafts (user_id, draft_prompt) VALUES ($1, '')`, user.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
SELECT id, public_id, email, password_hash, display_name, role, status, token_version, email_verified_at, created_at, updated_at
FROM users WHERE email = $1 AND deleted_at IS NULL`, email)
	return scanUser(row)
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row := s.Pool.QueryRow(ctx, `
SELECT id, public_id, email, password_hash, display_name, role, status, token_version, email_verified_at, created_at, updated_at
FROM users WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanUser(row)
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.PublicID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Status, &u.TokenVersion, &u.EmailVerifiedAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) EmailExists(ctx context.Context, email string) (bool, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE email = $1 AND deleted_at IS NULL`, email).Scan(&n)
	return n > 0, err
}

func (s *Store) GetWorkspaceName(ctx context.Context, userID int64) (string, error) {
	var name string
	err := s.Pool.QueryRow(ctx, `SELECT name FROM workspaces WHERE user_id = $1`, userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return name, err
}

func (s *Store) GetBalance(ctx context.Context, userID int64) (float64, error) {
	var bal float64
	err := s.Pool.QueryRow(ctx, `SELECT balance::float8 FROM wallets WHERE user_id = $1`, userID).Scan(&bal)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return bal, err
}

func (s *Store) SaveRefreshToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := s.Pool.Exec(ctx, `
INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt)
	return err
}

func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.Pool.Exec(ctx, `
UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	return err
}

func (s *Store) FindValidRefreshToken(ctx context.Context, tokenHash string) (userID int64, err error) {
	err = s.Pool.QueryRow(ctx, `
SELECT user_id FROM refresh_tokens
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return userID, err
}

func (s *Store) BumpTokenVersion(ctx context.Context, userID int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET token_version = token_version + 1, updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}

func (s *Store) SeedBasics(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `
INSERT INTO recharge_products (public_id, amount, bonus, price, sort_order) VALUES
 ('p-1', 50, 0, 50, 1),
 ('p-2', 100, 5, 100, 2),
 ('p-3', 300, 24, 300, 3),
 ('p-4', 500, 50, 500, 4)
ON CONFLICT (public_id) DO NOTHING`)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
INSERT INTO redeem_codes (code, amount, label) VALUES ('S2V-WELCOME-20', 20, '新用户体验金')
ON CONFLICT (code) DO NOTHING`)
	return err
}
