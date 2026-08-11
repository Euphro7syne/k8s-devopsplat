package config

import (
	"fmt"
	"os"
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
	Mode       string `yaml:"mode"`
	Kubeconfig string `yaml:"kubeconfig"`
	Namespace  string `yaml:"namespace"`
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
	JWTIssuer       string           `yaml:"jwt_issuer"`
	JWTSecret       string           `yaml:"jwt_secret"`
	AccessTokenTTL  time.Duration    `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration    `yaml:"refresh_token_ttl"`
	MFAEnabled      bool             `yaml:"mfa_enabled"`
	LocalAdmin      LocalAdminConfig `yaml:"local_admin"`
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
			LocalAdmin: LocalAdminConfig{
				Enabled:  true,
				Username: "admin",
				Email:    "admin@example.com",
				Password: "change-me-admin-password",
			},
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		return &cfg, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
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
	if c.Auth.LocalAdmin.Enabled {
		if c.Auth.LocalAdmin.Email == "" {
			return fmt.Errorf("auth.local_admin.email is required when local admin is enabled")
		}
		if c.Auth.LocalAdmin.Password == "" {
			return fmt.Errorf("auth.local_admin.password is required when local admin is enabled")
		}
	}
	switch c.Database.Driver {
	case "sqlite", "sqlite3":
	default:
		return fmt.Errorf("unsupported database.driver %q", c.Database.Driver)
	}
	return nil
}
