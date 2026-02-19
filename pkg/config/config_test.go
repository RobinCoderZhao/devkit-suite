package config

import (
	"os"
	"testing"
)

type testConfig struct {
	Name     string `yaml:"name" env:"APP_NAME"`
	Port     int    `yaml:"port" env:"APP_PORT"`
	Debug    bool   `yaml:"debug" env:"APP_DEBUG"`
	Database struct {
		DSN string `yaml:"dsn" env:"DATABASE_URL"`
	} `yaml:"database"`
}

func TestLoad(t *testing.T) {
	// Create a temporary config file
	content := `
name: test-app
port: 8080
debug: false
database:
  dsn: sqlite3://test.db
`
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	var cfg testConfig
	if err := Load(f.Name(), &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Name != "test-app" {
		t.Fatalf("expected 'test-app', got '%s'", cfg.Name)
	}
	if cfg.Port != 8080 {
		t.Fatalf("expected 8080, got %d", cfg.Port)
	}
	if cfg.Debug {
		t.Fatal("expected debug to be false")
	}
}

func TestEnvOverride(t *testing.T) {
	content := `
name: default
port: 3000
`
	f, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	// Set env vars
	t.Setenv("APP_NAME", "from-env")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_DEBUG", "true")

	var cfg testConfig
	if err := Load(f.Name(), &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Name != "from-env" {
		t.Fatalf("expected 'from-env', got '%s'", cfg.Name)
	}
	if cfg.Port != 9090 {
		t.Fatalf("expected 9090, got %d", cfg.Port)
	}
	if !cfg.Debug {
		t.Fatal("expected debug to be true from env")
	}
}

func TestLoadOrDefault_MissingFile(t *testing.T) {
	var cfg testConfig
	if err := LoadOrDefault("/nonexistent/config.yaml", &cfg); err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	// Should use zero values
	if cfg.Name != "" {
		t.Fatalf("expected empty name, got '%s'", cfg.Name)
	}
}
