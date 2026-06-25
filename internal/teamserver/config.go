package teamserver

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the teamserver.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	JWT       JWTConfig       `yaml:"jwt"`
	MasterKey string          `yaml:"master_key"`
	LogLevel  string          `yaml:"log_level"`
	TLS       TLSConfig       `yaml:"tls"`
}

// ServerConfig holds the HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig holds the SQLite database settings.
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// JWTConfig holds JWT token settings.
type JWTConfig struct {
	Secret        string        `yaml:"secret"`
	AccessExpiry  time.Duration `yaml:"access_expiry"`
	RefreshExpiry time.Duration `yaml:"refresh_expiry"`
}

// TLSConfig holds TLS certificate settings.
type TLSConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8443,
		},
		Database: DatabaseConfig{
			Path: "./data/c2.db",
		},
		JWT: JWTConfig{
			Secret:        "",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
		MasterKey: "",
		LogLevel:  "info",
		TLS: TLSConfig{
			CertFile: "",
			KeyFile:  "",
		},
	}
}

// Load reads a YAML config file and applies defaults for missing values.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if config file doesn't exist
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the configuration to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Addr returns the listen address string for the HTTP server.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
