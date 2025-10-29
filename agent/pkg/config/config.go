package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Agent  AgentConfig  `yaml:"agent"`
	Log    LogConfig    `yaml:"logging"`
}

type ServerConfig struct {
	Address       string        `yaml:"address"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryInterval time.Duration `yaml:"retry_interval"`
}

type AgentConfig struct {
	ReportInterval time.Duration     `yaml:"report_interval"`
	AgentID        string            `yaml:"agent_id"`
	Tags           map[string]string `yaml:"tags"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return getDefaultConfig(), nil // 如果配置文件不存在，使用默认配置
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// 合并默认配置
	mergeDefaults(&config)

	return &config, nil
}

func getDefaultConfigPath() string {
	return filepath.Join("agent", "config", "config.yaml")
}

func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:       "localhost:50051",
			Timeout:       10 * time.Second,
			RetryInterval: 5 * time.Second,
		},
		Agent: AgentConfig{
			ReportInterval: 30 * time.Second,
			AgentID:        "",
			Tags: map[string]string{
				"role":    "agent",
				"env":     "production",
				"version": "1.0.0",
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func mergeDefaults(config *Config) {
	defaults := getDefaultConfig()

	if config.Server.Address == "" {
		config.Server.Address = defaults.Server.Address
	}
	if config.Server.Timeout == 0 {
		config.Server.Timeout = defaults.Server.Timeout
	}
	if config.Server.RetryInterval == 0 {
		config.Server.RetryInterval = defaults.Server.RetryInterval
	}
	if config.Agent.ReportInterval == 0 {
		config.Agent.ReportInterval = defaults.Agent.ReportInterval
	}
	if config.Agent.Tags == nil {
		config.Agent.Tags = defaults.Agent.Tags
	}
	if config.Log.Level == "" {
		config.Log.Level = defaults.Log.Level
	}
	if config.Log.Format == "" {
		config.Log.Format = defaults.Log.Format
	}
}
