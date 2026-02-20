package benchmarks

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the benchmark tracker configuration.
type Config struct {
	Models []ModelConfig `yaml:"models"`
}

// LoadConfig loads model configuration from a YAML file.
// Falls back to DefaultModels if the file doesn't exist.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Models: DefaultModels}, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if len(cfg.Models) == 0 {
		cfg.Models = DefaultModels
	}

	// Assign display order if not set
	for i := range cfg.Models {
		if cfg.Models[i].DisplayOrder == 0 {
			cfg.Models[i].DisplayOrder = i + 1
		}
	}

	return &cfg, nil
}

// SaveConfig writes the model configuration to a YAML file.
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	header := "# Benchmark Tracker 模型配置\n# 自定义需要对比的模型列表\n# 用法: watchbot benchmark --config=config/benchmark_models.yaml\n\n"
	return os.WriteFile(path, []byte(header+string(data)), 0644)
}

// ParseModelsCLI parses a comma-separated model list from CLI flag.
// Format: "ModelName" or "ModelName,Provider"
func ParseModelsCLI(modelsFlag string) []ModelConfig {
	if modelsFlag == "" {
		return nil
	}

	parts := strings.Split(modelsFlag, ",")
	var models []ModelConfig
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		models = append(models, ModelConfig{
			Name:         part,
			Provider:     guessProvider(part),
			DisplayOrder: i + 1,
		})
	}
	return models
}

// AddModel parses "--add-model=Name,Provider" and appends to models.
func AddModel(models []ModelConfig, addFlag string) []ModelConfig {
	parts := strings.SplitN(addFlag, ",", 2)
	name := strings.TrimSpace(parts[0])
	provider := ""
	if len(parts) > 1 {
		provider = strings.TrimSpace(parts[1])
	}
	if provider == "" {
		provider = guessProvider(name)
	}
	return append(models, ModelConfig{
		Name:         name,
		Provider:     strings.ToLower(provider),
		Gen:          "latest",
		DisplayOrder: len(models) + 1,
	})
}

// guessProvider tries to infer the provider from a model name.
func guessProvider(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "gemini"):
		return "google"
	case strings.Contains(lower, "gpt") || strings.Contains(lower, "o1") || strings.Contains(lower, "codex"):
		return "openai"
	case strings.Contains(lower, "claude") || strings.Contains(lower, "sonnet") || strings.Contains(lower, "opus") || strings.Contains(lower, "haiku"):
		return "anthropic"
	case strings.Contains(lower, "qwen"):
		return "alibaba"
	case strings.Contains(lower, "deepseek"):
		return "deepseek"
	case strings.Contains(lower, "minimax"):
		return "minimax"
	case strings.Contains(lower, "llama") || strings.Contains(lower, "maverick"):
		return "meta"
	case strings.Contains(lower, "mistral"):
		return "mistral"
	default:
		return "unknown"
	}
}
