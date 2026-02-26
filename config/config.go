package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config represents application configuration
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Logger  LoggerConfig  `mapstructure:"logger"`
	Tracing TracingConfig `mapstructure:"tracing"`
	I18n    I18nConfig    `mapstructure:"i18n"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json or console
	OutputPath string `mapstructure:"output_path"`
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	Endpoint    string  `mapstructure:"endpoint"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// I18nConfig holds i18n configuration
type I18nConfig struct {
	DefaultLanguage string   `mapstructure:"default_language"`
	SupportedLangs  []string `mapstructure:"supported_languages"`
	Path            string   `mapstructure:"path"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)

	// Logger
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.output_path", "stdout")

	// Tracing
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.service_name", "microservice")
	v.SetDefault("tracing.endpoint", "http://localhost:14268/api/traces")
	v.SetDefault("tracing.sample_rate", 1.0)

	// I18n
	v.SetDefault("i18n.default_language", "en")
	v.SetDefault("i18n.supported_languages", []string{"en", "ru"})
	v.SetDefault("i18n.path", "./locales")
}
