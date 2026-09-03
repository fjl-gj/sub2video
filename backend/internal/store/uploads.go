package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
)

type Upload struct {
	ID          string
	UserID      int64
	FileName    string
	MimeType    string
	SizeBytes   int64
	StoragePath string
	CreatedAt   time.Time
}

func (s *Store) CreateUpload(ctx context.Context, userID int64, publicID, fileName, mimeType, storagePath string, sizeBytes int64) (*Upload, error) {
	if publicID == "" {
		publicID = "upl_" + shortID()
	}
	_, err := s.Pool.Exec(ctx, `
INSERT INTO uploads (public_id, user_id, file_name, mime_type, size_bytes, storage_path)
VALUES ($1, $2, $3, $4, $5, $6)`, publicID, userID, fileName, mimeType, sizeBytes, storagePath)
	if err != nil {
		return nil, err
	}
	return &Upload{
		ID: publicID, UserID: userID, FileName: fileName, MimeType: mimeType,
		SizeBytes: sizeBytes, StoragePath: storagePath, CreatedAt: time.Now(),
	}, nil
}

func (s *Store) GetUpload(ctx context.Context, userID int64, publicID string) (*Upload, error) {
	var u Upload
	var id int64
	err := s.Pool.QueryRow(ctx, `
SELECT id, public_id, user_id, file_name, mime_type, size_bytes, storage_path, created_at
FROM uploads WHERE public_id = $1 AND user_id = $2`, publicID, userID).Scan(
		&id, &u.ID, &u.UserID, &u.FileName, &u.MimeType, &u.SizeBytes, &u.StoragePath, &u.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func EnsureUploadDir(root string) error {
	return os.MkdirAll(root, 0o755)
}

func UploadPath(root, publicID, fileName string) string {
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".bin"
	}
	return filepath.Join(root, fmt.Sprintf("%s%s", publicID, ext))
}
