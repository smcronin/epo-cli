package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigDirEnvVar  = "EPO_CLI_CONFIG_DIR"
	ConfigFileEnvVar = "EPO_CLI_CONFIG_FILE"
)

var clientIDEnvKeys = []string{
	"EPO_CLIENT_ID",
	"EPO_CONSUMER_KEY",
	"CONSUMER_KEY",
}

var clientSecretEnvKeys = []string{
	"EPO_CLIENT_SECRET",
	"EPO_CONSUMER_SECRET",
	"CONSUMER_SECRET_KEY",
}

type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func configDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(ConfigDirEnvVar)); dir != "" {
		return dir, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "epo-cli"), nil
}

func ConfigFilePath() (string, error) {
	if path := strings.TrimSpace(os.Getenv(ConfigFileEnvVar)); path != "" {
		return path, nil
	}
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	return cfg, nil
}

func Save(cfg Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

func ResolveCredentials(flagClientID, flagClientSecret string, cfg Config) (clientID, clientSecret, clientIDSource, clientSecretSource string) {
	if v := strings.TrimSpace(flagClientID); v != "" {
		clientID = v
		clientIDSource = "flag"
	} else if v, key := firstSetEnv(clientIDEnvKeys); v != "" {
		clientID = v
		clientIDSource = "env:" + key
	} else if v := strings.TrimSpace(cfg.ClientID); v != "" {
		clientID = v
		clientIDSource = "config"
	}

	if v := strings.TrimSpace(flagClientSecret); v != "" {
		clientSecret = v
		clientSecretSource = "flag"
	} else if v, key := firstSetEnv(clientSecretEnvKeys); v != "" {
		clientSecret = v
		clientSecretSource = "env:" + key
	} else if v := strings.TrimSpace(cfg.ClientSecret); v != "" {
		clientSecret = v
		clientSecretSource = "config"
	}

	return clientID, clientSecret, clientIDSource, clientSecretSource
}

func firstSetEnv(keys []string) (string, string) {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v, k
		}
	}
	return "", ""
}

func Mask(value string) string {
	value = strings.TrimSpace(value)
	n := len(value)
	if n == 0 {
		return ""
	}
	if n <= 6 {
		return strings.Repeat("*", n)
	}
	return value[:3] + strings.Repeat("*", n-6) + value[n-3:]
}

// LoadCredentialsFromDotEnv loads credentials from a dotenv-style file.
// It supports either "KEY=value" or "export KEY=value" lines.
func LoadCredentialsFromDotEnv(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read dotenv %q: %w", path, err)
	}

	var cfg Config
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}

		key := strings.TrimSpace(line[:eq])
		value := unquoteEnvValue(strings.TrimSpace(line[eq+1:]))
		if value == "" {
			continue
		}

		if cfg.ClientID == "" && containsKey(clientIDEnvKeys, key) {
			cfg.ClientID = value
		}
		if cfg.ClientSecret == "" && containsKey(clientSecretEnvKeys, key) {
			cfg.ClientSecret = value
		}
	}

	return cfg, nil
}

func containsKey(keys []string, key string) bool {
	for _, candidate := range keys {
		if candidate == key {
			return true
		}
	}
	return false
}

func unquoteEnvValue(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	value = strings.ReplaceAll(value, "\\\"", "\"")
	value = strings.ReplaceAll(value, "\\\\", "\\")
	return strings.TrimSpace(value)
}
