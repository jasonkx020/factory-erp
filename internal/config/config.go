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
	Trace    TraceConfig    `yaml:"trace"`
	Mqtt     MqttConfig     `yaml:"mqtt"`
}

type TraceConfig struct {
	HMACSecret string `yaml:"hmac_secret"`
}

type MqttConfig struct {
	Enabled          bool   `yaml:"enabled"`
	Broker           string `yaml:"broker"` // nanomq
	BrokerURL        string `yaml:"broker_url"`
	WSURL            string `yaml:"ws_url"`
	HTTPAuthEnabled  bool   `yaml:"http_auth_enabled"`
	ServerUsername   string `yaml:"server_username"`
	ServerPassword   string `yaml:"server_password"`
	ClientPrefix     string `yaml:"client_prefix"`
	Tenant           string `yaml:"tenant"`
	TokenTTLSeconds  int64  `yaml:"token_ttl_seconds"`
	KeepAliveSeconds int    `yaml:"keep_alive_seconds"`
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
	if c.Mqtt.Broker == "" {
		c.Mqtt.Broker = "nanomq"
	}
	if c.Mqtt.BrokerURL == "" {
		c.Mqtt.BrokerURL = "tcp://127.0.0.1:1883"
	}
	if c.Mqtt.WSURL == "" {
		c.Mqtt.WSURL = "ws://127.0.0.1:8083"
	}
	if c.Mqtt.ServerUsername == "" {
		c.Mqtt.ServerUsername = "erp-hub"
	}
	if c.Mqtt.ClientPrefix == "" {
		c.Mqtt.ClientPrefix = "erp"
	}
	if c.Mqtt.Tenant == "" {
		c.Mqtt.Tenant = "default"
	}
	if c.Mqtt.TokenTTLSeconds <= 0 {
		c.Mqtt.TokenTTLSeconds = 43200
	}
	if c.Mqtt.KeepAliveSeconds <= 0 {
		c.Mqtt.KeepAliveSeconds = 60
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
	if v := os.Getenv("ERP_TRACE_HMAC_SECRET"); v != "" {
		c.Trace.HMACSecret = v
	}
	if v := os.Getenv("ERP_MQTT_ENABLED"); v != "" {
		c.Mqtt.Enabled = v == "1" || strings.EqualFold(v, "true")
	}
	if v := os.Getenv("ERP_MQTT_BROKER_URL"); v != "" {
		c.Mqtt.BrokerURL = v
	}
	if v := os.Getenv("ERP_MQTT_WS_URL"); v != "" {
		c.Mqtt.WSURL = v
	}
	if v := os.Getenv("ERP_MQTT_HUB_USER"); v != "" {
		c.Mqtt.ServerUsername = v
	}
	if v := os.Getenv("ERP_MQTT_HUB_PASS"); v != "" {
		c.Mqtt.ServerPassword = v
	}
}
