package store

import (
	"context"
	"strings"

	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/crypto"
	"github.com/sbu2video/backend/internal/storage/blob"
	"github.com/sbu2video/backend/internal/storage/cos"
	miniostorage "github.com/sbu2video/backend/internal/storage/minio"
)

func (s *Store) LoadCOSConfig(ctx context.Context, secretKey string) cos.Config {
	m, _ := s.GetSettings(ctx, []string{
		"cos_secret_id", "cos_secret_key", "cos_bucket", "cos_region", "cos_use_intranet",
	})
	key := m["cos_secret_key"]
	if key != "" {
		if plain, err := crypto.DecryptAESGCM(secretKey, key); err == nil && plain != "" {
			key = plain
		}
	}
	return cos.Config{
		SecretID:    m["cos_secret_id"],
		SecretKey:   key,
		Bucket:      m["cos_bucket"],
		Region:      m["cos_region"],
		UseIntranet: m["cos_use_intranet"] == "true",
	}
}

func (s *Store) SaveCOSSecretKey(ctx context.Context, secretKey, plain string) error {
	if plain == "" {
		return nil
	}
	enc, err := crypto.EncryptAESGCM(secretKey, plain)
	if err != nil {
		return err
	}
	return s.SetSetting(ctx, "cos_secret_key", enc)
}

func (s *Store) LoadMinIOConfig(ctx context.Context, secretKey string) miniostorage.Config {
	m, _ := s.GetSettings(ctx, []string{
		"minio_endpoint", "minio_access_key", "minio_secret_key", "minio_bucket",
		"minio_use_ssl", "minio_public_base",
	})
	sec := m["minio_secret_key"]
	if sec != "" {
		if plain, err := crypto.DecryptAESGCM(secretKey, sec); err == nil && plain != "" {
			sec = plain
		}
	}
	return miniostorage.Config{
		Endpoint:   m["minio_endpoint"],
		AccessKey:  m["minio_access_key"],
		SecretKey:  sec,
		Bucket:     m["minio_bucket"],
		UseSSL:     m["minio_use_ssl"] == "true",
		PublicBase: m["minio_public_base"],
	}
}

func (s *Store) SaveMinIOSecretKey(ctx context.Context, secretKey, plain string) error {
	if plain == "" {
		return nil
	}
	enc, err := crypto.EncryptAESGCM(secretKey, plain)
	if err != nil {
		return err
	}
	return s.SetSetting(ctx, "minio_secret_key", enc)
}

func (s *Store) StorageProvider(ctx context.Context) string {
	v, _ := s.GetSetting(ctx, "storage_provider")
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return "minio"
	}
	return v
}

// LoadObjectStore returns the active object storage backend (minio or cos).
func (s *Store) LoadObjectStore(ctx context.Context, cfg config.Config) blob.Store {
	provider := s.StorageProvider(ctx)
	switch provider {
	case "cos":
		c := s.LoadCOSConfig(ctx, cfg.SecretKey)
		if c.SecretID == "" && cfg.COSSecretID != "" {
			c = cos.Config{
				SecretID: cfg.COSSecretID, SecretKey: cfg.COSSecretKey,
				Bucket: cfg.COSBucket, Region: cfg.COSRegion,
			}
		}
		return cos.New(c)
	default:
		m := s.LoadMinIOConfig(ctx, cfg.SecretKey)
		return miniostorage.New(m)
	}
}
