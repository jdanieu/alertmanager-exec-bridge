package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Listen     string        `mapstructure:"listen"`
	Token      string        `mapstructure:"token"`
	Command    string        `mapstructure:"command"`
	Args       []string      `mapstructure:"args"`
	Timeout    time.Duration `mapstructure:"-"`       // se rellena a partir de TimeoutRaw
	TimeoutRaw string        `mapstructure:"timeout"` // string: "5s", "1m", etc.
	LogLevel   string        `mapstructure:"log_level"`
}

// Load loads configuration with the following precedence:
// 1. Defaults (hardcoded)
// 2. Config file (if provided)
// 3. Environment variables (ALERT_EXEC_*)
// Flags se aplican en main.go sobre el resultado.
func Load(configFile string) (Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("listen", ":9095")
	v.SetDefault("token", "")
	v.SetDefault("command", "/usr/local/bin/send-evolution")
	v.SetDefault("args", []string{})
	v.SetDefault("timeout", "5s")
	v.SetDefault("log_level", "info")

	// Config file (optional)
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables
	// Ej: ALERT_EXEC_LISTEN, ALERT_EXEC_COMMAND, ALERT_EXEC_TIMEOUT, etc.
	v.SetEnvPrefix("ALERT_EXEC")
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Parse timeout string a time.Duration
	if cfg.TimeoutRaw == "" {
		cfg.TimeoutRaw = "5s"
	}
	dur, err := time.ParseDuration(cfg.TimeoutRaw)
	if err != nil {
		return Config{}, fmt.Errorf("invalid timeout value %q: %w", cfg.TimeoutRaw, err)
	}
	cfg.Timeout = dur

	// Validaciones mínimas
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Validate(cfg Config) error {
	if cfg.Listen == "" {
		return fmt.Errorf("listen address cannot be empty")
	}
	if cfg.Command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}
	return nil
}
