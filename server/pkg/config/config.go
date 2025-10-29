package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP    HTTPConfig    `yaml:"http"`
	GRPC    GRPCConfig    `yaml:"grpc"`
	MySQL   MySQLConfig   `yaml:"mysql"`
	Redis   RedisConfig   `yaml:"redis"`
	Logging LoggingConfig `yaml:"logging"`
}

type HTTPConfig struct {
	Address string `yaml:"address"`
}

type GRPCConfig struct {
	Address string `yaml:"address"`
}

type MySQLConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	DBName    string `yaml:"dbname"`
	Charset   string `yaml:"charset"`
	ParseTime bool   `yaml:"parse_time"`
	Loc       string `yaml:"loc"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func LoadConfig() (*Config, error) {
	configPath := getConfigPath()

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

func getConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	return filepath.Join("server", "config", "config.yaml")
}

func getDefaultConfig() *Config {
	return &Config{
		HTTP: HTTPConfig{
			Address: ":8080",
		},
		GRPC: GRPCConfig{
			Address: ":50051",
		},
		MySQL: MySQLConfig{
			Host:      "127.0.0.1",
			Port:      3307,
			User:      "root",
			Password:  "123456",
			DBName:    "devops_manager",
			Charset:   "utf8mb4",
			ParseTime: true,
			Loc:       "Local",
		},
		Redis: RedisConfig{
			Host:     "127.0.0.1",
			Port:     6380,
			Password: "",
			DB:       0,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

func mergeDefaults(config *Config) {
	defaults := getDefaultConfig()

	if config.HTTP.Address == "" {
		config.HTTP.Address = defaults.HTTP.Address
	}
	if config.GRPC.Address == "" {
		config.GRPC.Address = defaults.GRPC.Address
	}
	if config.MySQL.Host == "" {
		config.MySQL = defaults.MySQL
	}
	if config.Redis.Host == "" {
		config.Redis = defaults.Redis
	}
	if config.Logging.Level == "" {
		config.Logging.Level = defaults.Logging.Level
	}
	if config.Logging.Format == "" {
		config.Logging.Format = defaults.Logging.Format
	}
}
