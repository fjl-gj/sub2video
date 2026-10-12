package config

import (
	"strings"
	"testing"
)

func strongProd() Config {
	return Config{
		Env:          EnvProduction,
		JWTSecret:    "prod-jwt-secret-0123456789abcdef-xyz",
		SecretKey:    "prod-secret-key-0123456789abcdef-xyz",
		CORSOrigin:   "https://app.example.com,https://admin.example.com",
		SeedDemo:     false,
		SeedPassword: "a-long-custom-seed-pass",
	}
}

func TestValidateAcceptsStrongProductionConfig(t *testing.T) {
	if err := Validate(strongProd()); err != nil {
		t.Fatalf("expected strong production config to pass, got %v", err)
	}
}

func TestValidateDevelopmentAllowsLocalDefaults(t *testing.T) {
	c := Config{Env: EnvDevelopment, JWTSecret: devJWTSecret, SecretKey: devSecretKey, CORSOrigin: "*", SeedDemo: true, SeedPassword: demoSeedPassword, PromoteAdminEmail: demoSeedEmail}
	if err := Validate(c); err != nil {
		t.Fatalf("development defaults must remain usable, got %v", err)
	}
}

func TestValidateRejectsUnsafeProductionSettings(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"empty jwt", func(c *Config) { c.JWTSecret = "" }, "SUB2VIDEO_JWT_SECRET is required"},
		{"dev jwt default", func(c *Config) { c.JWTSecret = devJWTSecret }, "placeholder or development value"},
		{"repo placeholder secret", func(c *Config) { c.SecretKey = "change-me-in-production-secret" }, "placeholder or development value"},
		{"short secret", func(c *Config) { c.SecretKey = "short" }, "at least 32 characters"},
		{"wildcard cors", func(c *Config) { c.CORSOrigin = "*" }, "must not be \"*\""},
		{"empty cors", func(c *Config) { c.CORSOrigin = " , " }, "must list allowed origins"},
		{"demo seed with default password", func(c *Config) { c.SeedDemo = true; c.SeedPassword = demoSeedPassword }, "SUB2VIDEO_SEED_DEMO requires"},
		{"promote demo admin", func(c *Config) { c.PromoteAdminEmail = demoSeedEmail }, "must not be the demo account"},
		{"unknown env", func(c *Config) { c.Env = "staging" }, "SUB2VIDEO_ENV must be"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := strongProd()
			tc.mutate(&c)
			err := Validate(c)
			if err == nil {
				t.Fatalf("expected error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestLoadDefaultsToProductionAndFailsClosed(t *testing.T) {
	t.Setenv("SUB2VIDEO_ENV", "")
	t.Setenv("SUB2VIDEO_JWT_SECRET", "")
	t.Setenv("SUB2VIDEO_SECRET_KEY", "")
	t.Setenv("SUB2VIDEO_CORS_ORIGIN", "")
	t.Setenv("SUB2VIDEO_PROMOTE_ADMIN_EMAIL", "")
	t.Setenv("SUB2VIDEO_SEED_DEMO", "")
	c := Load()
	if !c.IsProduction() {
		t.Fatalf("unset SUB2VIDEO_ENV must mean production, got %q", c.Env)
	}
	if err := Validate(c); err == nil {
		t.Fatal("unset environment with no secrets must not start")
	}
	if c.SeedDemo || c.PromoteAdminEmail != "" {
		t.Fatalf("production must not seed or promote the demo account by default")
	}
}

func TestLoadDevelopmentKeepsLocalDefaults(t *testing.T) {
	t.Setenv("SUB2VIDEO_ENV", "development")
	t.Setenv("SUB2VIDEO_JWT_SECRET", "")
	t.Setenv("SUB2VIDEO_SECRET_KEY", "")
	t.Setenv("SUB2VIDEO_CORS_ORIGIN", "")
	t.Setenv("SUB2VIDEO_SEED_DEMO", "")
	t.Setenv("SUB2VIDEO_PROMOTE_ADMIN_EMAIL", "")
	c := Load()
	if err := Validate(c); err != nil {
		t.Fatalf("development load must validate, got %v", err)
	}
	if !c.SeedDemo || c.CORSOrigin != "*" || c.PromoteAdminEmail != demoSeedEmail {
		t.Fatalf("development defaults changed: seed=%v cors=%q promote=%q", c.SeedDemo, c.CORSOrigin, c.PromoteAdminEmail)
	}
}

func TestCORSOriginsParsesAllowList(t *testing.T) {
	c := Config{CORSOrigin: " https://a.example.com , ,https://b.example.com "}
	got := c.CORSOrigins()
	if len(got) != 2 || got[0] != "https://a.example.com" || got[1] != "https://b.example.com" {
		t.Fatalf("unexpected origins: %#v", got)
	}
}
