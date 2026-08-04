package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	CORS     CORSConfig     `yaml:"cors"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type DatabaseConfig struct {
	Driver     string `yaml:"driver"` // sqlite | mysql
	SQLitePath string `yaml:"sqlite_path"`
	MySQLDSN   string `yaml:"mysql_dsn"`
}

type JWTConfig struct {
	Secret         string `yaml:"secret"`
	AccessTTLMin   int    `yaml:"access_ttl_min"`
	RefreshTTLMin  int    `yaml:"refresh_ttl_min"`
}

type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	applyEnv(&c)
	if c.Server.Addr == "" {
		c.Server.Addr = ":18080"
	}
	if c.Database.Driver == "" {
		c.Database.Driver = "sqlite"
	}
	if c.Database.SQLitePath == "" {
		c.Database.SQLitePath = "data/erp_dev.db"
	}
	if c.JWT.AccessTTLMin <= 0 {
		c.JWT.AccessTTLMin = 120
	}
	if c.JWT.RefreshTTLMin <= 0 {
		c.JWT.RefreshTTLMin = 10080
	}
	if c.JWT.Secret == "" {
		c.JWT.Secret = "dev-only-change-me"
	}
	return &c, nil
}

func applyEnv(c *Config) {
	if v := os.Getenv("ERP_SERVER_ADDR"); v != "" {
		c.Server.Addr = v
	}
	if v := os.Getenv("ERP_DATABASE_DRIVER"); v != "" {
		c.Database.Driver = v
	}
	if v := os.Getenv("ERP_DATABASE_SQLITE_PATH"); v != "" {
		c.Database.SQLitePath = v
	}
	if v := os.Getenv("ERP_DATABASE_MYSQL_DSN"); v != "" {
		c.Database.MySQLDSN = v
	}
	if v := os.Getenv("ERP_JWT_SECRET"); v != "" {
		c.JWT.Secret = v
	}
	if v := os.Getenv("ERP_JWT_ACCESS_TTL_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.JWT.AccessTTLMin = n
		}
	}
	if v := os.Getenv("ERP_CORS_ALLOW_ORIGINS"); v != "" {
		c.CORS.AllowOrigins = strings.Split(v, ",")
	}
}
