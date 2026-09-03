package store

import (
	"context"

	"github.com/sbu2video/backend/internal/crypto"
	"github.com/sbu2video/backend/internal/storage/cos"
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
