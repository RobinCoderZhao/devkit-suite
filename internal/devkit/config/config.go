// Package config provides DevKit CLI configuration management.
package config

import (
	"os"
	"path/filepath"

	appconfig "github.com/RobinCoderZhao/API-Change-Sentinel/pkg/config"
	"github.com/RobinCoderZhao/API-Change-Sentinel/pkg/llm"
)

// DevKitConfig is the main configuration for DevKit CLI.
type DevKitConfig struct {
	LLM     llm.Config `yaml:"llm"`
	License struct {
		Key string `yaml:"key" env:"DEVKIT_LICENSE_KEY"`
	} `yaml:"license"`
	Commit CommitConfig `yaml:"commit"`
	Review ReviewConfig `yaml:"review"`
}

// CommitConfig holds settings for the commit command.
type CommitConfig struct {
	Language  string   `yaml:"language"`   // "en" or "zh"
	MaxLength int      `yaml:"max_length"` // Max commit message length
	Types     []string `yaml:"types"`      // Allowed commit types
	AutoStage bool     `yaml:"auto_stage"` // Auto-stage all changes
}

// ReviewConfig holds settings for the review command.
type ReviewConfig struct {
	OutputFormat string `yaml:"output_format"` // "text", "json"
}

// DefaultConfig returns a DevKitConfig with sensible defaults.
func DefaultConfig() DevKitConfig {
	return DevKitConfig{
		LLM: llm.Config{
			Provider:    llm.OpenAI,
			Model:       "gpt-4o-mini",
			MaxRetries:  2,
			Temperature: 0.3,
		},
		Commit: CommitConfig{
			Language:  "en",
			MaxLength: 72,
			Types:     []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore"},
			AutoStage: false,
		},
		Review: ReviewConfig{
			OutputFormat: "text",
		},
	}
}

// Load loads DevKit configuration from the standard config file location.
func Load() (DevKitConfig, error) {
	cfg := DefaultConfig()

	// Check project-level config first
	if _, err := os.Stat(".devkit.yaml"); err == nil {
		if err := appconfig.Load(".devkit.yaml", &cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}

	// Then check home directory
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".devkit.yaml")
		if err := appconfig.LoadOrDefault(globalPath, &cfg); err != nil {
			return cfg, err
		}
	}

	// Environment variable overrides
	if key := os.Getenv("LLM_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" && cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = key
	}

	return cfg, nil
}
