package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	SecretKey       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CORSOrigin      string
	SeedEmail       string
	SeedPassword    string
	UploadDir       string
	AutoDLToken     string
	AutoDLBaseURL   string
	COSSecretID     string
	COSSecretKey    string
	COSBucket       string
	COSRegion       string
	PromoteAdminEmail string
}

func Load() Config {
	return Config{
		Addr:              env("SUB2VIDEO_ADDR", "127.0.0.1:12080"),
		DatabaseURL:       env("SUB2VIDEO_DATABASE_URL", "postgres://sub2video:sub2video@127.0.0.1:15432/sub2video?sslmode=disable"),
		RedisURL:          env("SUB2VIDEO_REDIS_URL", "redis://127.0.0.1:16379/0"),
		JWTSecret:         env("SUB2VIDEO_JWT_SECRET", "dev-change-me-sub2video-jwt"),
		SecretKey:         env("SUB2VIDEO_SECRET_KEY", "dev-change-me-sub2video-secret"),
		AccessTokenTTL:    envDuration("SUB2VIDEO_ACCESS_TTL", 30*time.Minute),
		RefreshTokenTTL:   envDuration("SUB2VIDEO_REFRESH_TTL", 30*24*time.Hour),
		CORSOrigin:        env("SUB2VIDEO_CORS_ORIGIN", "*"),
		SeedEmail:         env("SUB2VIDEO_SEED_EMAIL", "demo@sub2video.local"),
		SeedPassword:      env("SUB2VIDEO_SEED_PASSWORD", "Demo1234!"),
		UploadDir:         env("SUB2VIDEO_UPLOAD_DIR", "./data/uploads"),
		AutoDLToken:       env("SUB2VIDEO_AUTODL_TOKEN", ""),
		AutoDLBaseURL:     env("SUB2VIDEO_AUTODL_BASE_URL", "https://autodl.art"),
		COSSecretID:       env("SUB2VIDEO_COS_SECRET_ID", ""),
		COSSecretKey:      env("SUB2VIDEO_COS_SECRET_KEY", ""),
		COSBucket:         env("SUB2VIDEO_COS_BUCKET", ""),
		COSRegion:         env("SUB2VIDEO_COS_REGION", ""),
		PromoteAdminEmail: env("SUB2VIDEO_PROMOTE_ADMIN_EMAIL", "demo@sub2video.local"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		return time.Duration(secs) * time.Second
	}
	return fallback
}
