// Package config provides YAML configuration loading with environment variable override.
package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads a YAML configuration file into the given struct.
// It also applies environment variable overrides using struct tags.
func Load(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}

	// Expand environment variables in the YAML
	expanded := os.ExpandEnv(string(data))

	if err := yaml.Unmarshal([]byte(expanded), out); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(out)

	return nil
}

// LoadOrDefault tries to load config from path, falls back to defaults if file doesn't exist.
func LoadOrDefault(path string, out any) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Use zero-value defaults
	}
	return Load(path, out)
}

// applyEnvOverrides sets struct fields from environment variables.
// It uses the `env` struct tag to determine the env var name.
func applyEnvOverrides(v any) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := val.Field(i)

		// Recurse into struct fields
		if fieldVal.Kind() == reflect.Struct {
			if fieldVal.CanAddr() {
				applyEnvOverrides(fieldVal.Addr().Interface())
			}
			continue
		}

		envTag := field.Tag.Get("env")
		if envTag == "" {
			continue
		}

		envVal, ok := os.LookupEnv(envTag)
		if !ok {
			continue
		}

		if !fieldVal.CanSet() {
			continue
		}

		switch fieldVal.Kind() {
		case reflect.String:
			fieldVal.SetString(envVal)
		case reflect.Int, reflect.Int64:
			var n int64
			if _, err := fmt.Sscanf(envVal, "%d", &n); err == nil {
				fieldVal.SetInt(n)
			}
		case reflect.Float64:
			var f float64
			if _, err := fmt.Sscanf(envVal, "%f", &f); err == nil {
				fieldVal.SetFloat(f)
			}
		case reflect.Bool:
			fieldVal.SetBool(strings.EqualFold(envVal, "true") || envVal == "1")
		}
	}
}
