package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	Database   DatabaseConfig   `yaml:"database"`
	Log        LogConfig        `yaml:"log"`
	Auth       AuthConfig       `yaml:"auth"`
}

type ServerConfig struct {
	Mode            string        `yaml:"mode"`
	Listen          string        `yaml:"listen"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	CORS            CORSConfig    `yaml:"cors"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

type KubernetesConfig struct {
	Mode       string                `yaml:"mode"`
	Kubeconfig string                `yaml:"kubeconfig"`
	Namespace  string                `yaml:"namespace"`
	Cache      KubernetesCacheConfig `yaml:"cache"`
}

type KubernetesCacheConfig struct {
	Enabled      bool          `yaml:"enabled"`
	ResyncPeriod time.Duration `yaml:"resync_period"`
	SyncTimeout  time.Duration `yaml:"sync_timeout"`
}

type DatabaseConfig struct {
	Driver       string `yaml:"driver"`
	DSN          string `yaml:"dsn"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	AutoMigrate  bool   `yaml:"auto_migrate"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type AuthConfig struct {
	JWTIssuer       string              `yaml:"jwt_issuer"`
	JWTSecret       string              `yaml:"jwt_secret"`
	AccessTokenTTL  time.Duration       `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration       `yaml:"refresh_token_ttl"`
	MFAEnabled      bool                `yaml:"mfa_enabled"`
	MFAIssuer       string              `yaml:"mfa_issuer"`
	MFAChallengeTTL time.Duration       `yaml:"mfa_challenge_ttl"`
	MFASecretKey    string              `yaml:"mfa_secret_key"`
	RateLimit       AuthRateLimitConfig `yaml:"rate_limit"`
	LocalAdmin      LocalAdminConfig    `yaml:"local_admin"`
}

type AuthRateLimitConfig struct {
	Enabled            bool          `yaml:"enabled"`
	LoginMaxAttempts   int           `yaml:"login_max_attempts"`
	LoginWindow        time.Duration `yaml:"login_window"`
	LoginBlockDuration time.Duration `yaml:"login_block_duration"`
	MFAMaxAttempts     int           `yaml:"mfa_max_attempts"`
	MFAWindow          time.Duration `yaml:"mfa_window"`
	MFABlockDuration   time.Duration `yaml:"mfa_block_duration"`
}

type LocalAdminConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Mode:            "debug",
			Listen:          "0.0.0.0:8080",
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			CORS: CORSConfig{
				AllowedOrigins: []string{"http://localhost:5173"},
				AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"Authorization", "Content-Type", "X-Request-ID"},
			},
		},
		Kubernetes: KubernetesConfig{
			Mode:       "auto",
			Kubeconfig: "~/.kube/config",
			Namespace:  "ops-platform",
			Cache: KubernetesCacheConfig{
				Enabled:      true,
				ResyncPeriod: 10 * time.Minute,
				SyncTimeout:  15 * time.Second,
			},
		},
		Database: DatabaseConfig{
			Driver:       "sqlite",
			DSN:          "file:./data/ops-platform.db?_foreign_keys=1&_busy_timeout=5000",
			MaxOpenConns: 1,
			MaxIdleConns: 1,
			AutoMigrate:  true,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Auth: AuthConfig{
			JWTIssuer:       "ops-platform",
			JWTSecret:       "change-me-placeholder",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
			MFAEnabled:      false,
			MFAIssuer:       "ops-platform",
			MFAChallengeTTL: 5 * time.Minute,
			MFASecretKey:    "change-me-mfa-secret-key",
			RateLimit:       DefaultAuthRateLimitConfig(),
			LocalAdmin: LocalAdminConfig{
				Enabled:  true,
				Username: "admin",
				Email:    "admin@example.com",
				Password: "admin123",
			},
		},
	}
}

func DefaultAuthRateLimitConfig() AuthRateLimitConfig {
	return AuthRateLimitConfig{
		Enabled:            true,
		LoginMaxAttempts:   5,
		LoginWindow:        5 * time.Minute,
		LoginBlockDuration: 15 * time.Minute,
		MFAMaxAttempts:     5,
		MFAWindow:          5 * time.Minute,
		MFABlockDuration:   15 * time.Minute,
	}
}

func (c AuthRateLimitConfig) WithDefaults() AuthRateLimitConfig {
	defaults := DefaultAuthRateLimitConfig()
	if c.LoginMaxAttempts <= 0 {
		c.LoginMaxAttempts = defaults.LoginMaxAttempts
	}
	if c.LoginWindow <= 0 {
		c.LoginWindow = defaults.LoginWindow
	}
	if c.LoginBlockDuration <= 0 {
		c.LoginBlockDuration = defaults.LoginBlockDuration
	}
	if c.MFAMaxAttempts <= 0 {
		c.MFAMaxAttempts = defaults.MFAMaxAttempts
	}
	if c.MFAWindow <= 0 {
		c.MFAWindow = defaults.MFAWindow
	}
	if c.MFABlockDuration <= 0 {
		c.MFABlockDuration = defaults.MFABlockDuration
	}
	return c
}

func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config %q: %w", path, err)
		}
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("parse config %q: %w", path, err)
		}
	}
	applyEnvironmentOverrides(&cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyEnvironmentOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if value := os.Getenv("OPS_DATABASE_DRIVER"); value != "" {
		cfg.Database.Driver = value
	}
	if value := os.Getenv("OPS_DATABASE_DSN"); value != "" {
		cfg.Database.DSN = value
	}
	if value := os.Getenv("OPS_AUTH_JWT_SECRET"); value != "" {
		cfg.Auth.JWTSecret = value
	}
	if value := os.Getenv("OPS_AUTH_MFA_SECRET_KEY"); value != "" {
		cfg.Auth.MFASecretKey = value
	}
	if value := os.Getenv("OPS_AUTH_LOCAL_ADMIN_PASSWORD"); value != "" {
		cfg.Auth.LocalAdmin.Password = value
	}
}

func (c Config) Validate() error {
	if c.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}
	if c.Database.Driver == "" {
		return fmt.Errorf("database.driver is required")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if c.Kubernetes.Cache.Enabled {
		if c.Kubernetes.Cache.ResyncPeriod <= 0 {
			return fmt.Errorf("kubernetes.cache.resync_period must be positive")
		}
		if c.Kubernetes.Cache.SyncTimeout <= 0 {
			return fmt.Errorf("kubernetes.cache.sync_timeout must be positive")
		}
	}
	if c.Auth.JWTIssuer == "" {
		return fmt.Errorf("auth.jwt_issuer is required")
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret is required")
	}
	if c.Auth.AccessTokenTTL <= 0 {
		return fmt.Errorf("auth.access_token_ttl must be positive")
	}
	if c.Auth.RefreshTokenTTL <= 0 {
		return fmt.Errorf("auth.refresh_token_ttl must be positive")
	}
	if c.Auth.LocalAdmin.Enabled {
		if c.Auth.LocalAdmin.Email == "" {
			return fmt.Errorf("auth.local_admin.email is required when local admin is enabled")
		}
		if c.Auth.LocalAdmin.Password == "" {
			return fmt.Errorf("auth.local_admin.password is required when local admin is enabled")
		}
	}
	if c.Auth.MFAIssuer == "" {
		return fmt.Errorf("auth.mfa_issuer is required")
	}
	if c.Auth.MFAChallengeTTL <= 0 {
		return fmt.Errorf("auth.mfa_challenge_ttl must be positive")
	}
	if c.Auth.MFASecretKey == "" {
		return fmt.Errorf("auth.mfa_secret_key is required")
	}
	if c.Auth.RateLimit.Enabled {
		if c.Auth.RateLimit.LoginMaxAttempts <= 0 {
			return fmt.Errorf("auth.rate_limit.login_max_attempts must be positive")
		}
		if c.Auth.RateLimit.LoginWindow <= 0 {
			return fmt.Errorf("auth.rate_limit.login_window must be positive")
		}
		if c.Auth.RateLimit.LoginBlockDuration <= 0 {
			return fmt.Errorf("auth.rate_limit.login_block_duration must be positive")
		}
		if c.Auth.RateLimit.MFAMaxAttempts <= 0 {
			return fmt.Errorf("auth.rate_limit.mfa_max_attempts must be positive")
		}
		if c.Auth.RateLimit.MFAWindow <= 0 {
			return fmt.Errorf("auth.rate_limit.mfa_window must be positive")
		}
		if c.Auth.RateLimit.MFABlockDuration <= 0 {
			return fmt.Errorf("auth.rate_limit.mfa_block_duration must be positive")
		}
	}
	switch strings.ToLower(strings.TrimSpace(c.Database.Driver)) {
	case "sqlite", "sqlite3":
	case "postgres", "postgresql":
	default:
		return fmt.Errorf("unsupported database.driver %q", c.Database.Driver)
	}
	return nil
}
