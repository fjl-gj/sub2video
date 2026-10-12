package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"

	devJWTSecret      = "dev-change-me-sub2video-jwt"
	devSecretKey      = "dev-change-me-sub2video-secret"
	demoSeedEmail     = "demo@sub2video.local"
	demoSeedPassword  = "Demo1234!"
	minProdSecretSize = 32
	minProdSeedSize   = 12
)

// placeholderSecrets are values that appear in repository examples or dev
// defaults. They must never be accepted as production secrets.
var placeholderSecrets = map[string]bool{
	devJWTSecret:                     true,
	devSecretKey:                     true,
	"change-me-in-production":        true,
	"change-me-in-production-secret": true,
}

type Config struct {
	// Env is "production" unless SUB2VIDEO_ENV=development is set explicitly.
	// Production fails closed: unsafe defaults stop the process at startup.
	Env             string
	Addr            string
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	SecretKey       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	// CORSOrigin is "*" only in development. In production it is a
	// comma-separated allow-list of exact origins.
	CORSOrigin string
	// SeedDemo controls whether the demo account is created on an empty
	// database. It defaults to true only in development.
	SeedDemo     bool
	SeedEmail    string
	SeedPassword string
	UploadDir    string
	// SkipBootstrap is for an already initialized database during releases
	// without schema changes. Migrations/seeds must then run separately.
	SkipBootstrap     bool
	AutoDLToken       string
	AutoDLBaseURL     string
	COSSecretID       string
	COSSecretKey      string
	COSBucket         string
	COSRegion         string
	PromoteAdminEmail string
}

func (c Config) IsProduction() bool {
	return c.Env == EnvProduction
}

// CORSOrigins returns the exact origins allowed to call the API.
func (c Config) CORSOrigins() []string {
	var out []string
	for _, item := range strings.Split(c.CORSOrigin, ",") {
		if v := strings.TrimSpace(item); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func Load() Config {
	envName := strings.ToLower(strings.TrimSpace(env("SUB2VIDEO_ENV", EnvProduction)))
	dev := envName == EnvDevelopment

	defaultJWT, defaultSecret := "", ""
	defaultCORS := "" // empty forces an explicit allow-list in production
	defaultSeedDemo := "false"
	defaultPromote := ""
	if dev {
		defaultJWT, defaultSecret = devJWTSecret, devSecretKey
		defaultCORS = "*"
		defaultSeedDemo = "true"
		defaultPromote = demoSeedEmail
	}

	return Config{
		Env:               envName,
		Addr:              env("SUB2VIDEO_ADDR", "127.0.0.1:12080"),
		DatabaseURL:       env("SUB2VIDEO_DATABASE_URL", "postgres://sub2video:sub2video@127.0.0.1:15432/sub2video?sslmode=disable"),
		RedisURL:          env("SUB2VIDEO_REDIS_URL", "redis://127.0.0.1:16379/0"),
		JWTSecret:         env("SUB2VIDEO_JWT_SECRET", defaultJWT),
		SecretKey:         env("SUB2VIDEO_SECRET_KEY", defaultSecret),
		AccessTokenTTL:    envDuration("SUB2VIDEO_ACCESS_TTL", 30*time.Minute),
		RefreshTokenTTL:   envDuration("SUB2VIDEO_REFRESH_TTL", 30*24*time.Hour),
		CORSOrigin:        env("SUB2VIDEO_CORS_ORIGIN", defaultCORS),
		SeedDemo:          env("SUB2VIDEO_SEED_DEMO", defaultSeedDemo) == "true",
		SeedEmail:         env("SUB2VIDEO_SEED_EMAIL", demoSeedEmail),
		SeedPassword:      env("SUB2VIDEO_SEED_PASSWORD", demoSeedPassword),
		UploadDir:         env("SUB2VIDEO_UPLOAD_DIR", "./data/uploads"),
		SkipBootstrap:     env("SUB2VIDEO_SKIP_BOOTSTRAP", "false") == "true",
		AutoDLToken:       env("SUB2VIDEO_AUTODL_TOKEN", ""),
		AutoDLBaseURL:     env("SUB2VIDEO_AUTODL_BASE_URL", "https://autodl.art"),
		COSSecretID:       env("SUB2VIDEO_COS_SECRET_ID", ""),
		COSSecretKey:      env("SUB2VIDEO_COS_SECRET_KEY", ""),
		COSBucket:         env("SUB2VIDEO_COS_BUCKET", ""),
		COSRegion:         env("SUB2VIDEO_COS_REGION", ""),
		PromoteAdminEmail: env("SUB2VIDEO_PROMOTE_ADMIN_EMAIL", defaultPromote),
	}
}

// Validate rejects configurations that are unsafe to run in production.
// Development mode accepts the documented local defaults.
func Validate(c Config) error {
	switch c.Env {
	case EnvDevelopment:
		return nil
	case EnvProduction:
	default:
		return fmt.Errorf("SUB2VIDEO_ENV must be %q or %q, got %q", EnvDevelopment, EnvProduction, c.Env)
	}

	var problems []string
	checkSecret := func(name, value string) {
		switch {
		case value == "":
			problems = append(problems, name+" is required in production")
		case placeholderSecrets[value]:
			problems = append(problems, name+" uses a placeholder or development value")
		case len(value) < minProdSecretSize:
			problems = append(problems, fmt.Sprintf("%s must be at least %d characters", name, minProdSecretSize))
		}
	}
	checkSecret("SUB2VIDEO_JWT_SECRET", c.JWTSecret)
	checkSecret("SUB2VIDEO_SECRET_KEY", c.SecretKey)

	origins := c.CORSOrigins()
	if len(origins) == 0 {
		problems = append(problems, "SUB2VIDEO_CORS_ORIGIN must list allowed origins in production")
	}
	for _, o := range origins {
		if o == "*" {
			problems = append(problems, "SUB2VIDEO_CORS_ORIGIN must not be \"*\" in production")
			break
		}
	}

	if c.SeedDemo {
		if c.SeedPassword == demoSeedPassword || len(c.SeedPassword) < minProdSeedSize {
			problems = append(problems, fmt.Sprintf("SUB2VIDEO_SEED_DEMO requires a non-default SUB2VIDEO_SEED_PASSWORD of at least %d characters", minProdSeedSize))
		}
	}
	if c.PromoteAdminEmail == demoSeedEmail {
		problems = append(problems, "SUB2VIDEO_PROMOTE_ADMIN_EMAIL must not be the demo account in production")
	}

	if len(problems) > 0 {
		return errors.New("unsafe production configuration: " + strings.Join(problems, "; "))
	}
	return nil
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
