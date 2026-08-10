package config

import (
	"log"
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
	Seed     SeedConfig     `yaml:"seed"`
	OAuth    OAuthConfig    `yaml:"oauth"`
	OCR      OCRConfig      `yaml:"ocr"`
}

// OAuthConfig 第三方登录（默认关闭；启用后按 provider 交换 code）。
type OAuthConfig struct {
	Enabled   bool              `yaml:"enabled"`
	Providers map[string]string `yaml:"providers"` // provider -> secret/appid placeholder
}

// OCRConfig 身份证 OCR（默认关闭）。
type OCRConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"`
}

type SeedConfig struct {
	// Demo controls showcase fake data for all menus. nil = auto (on for sqlite).
	Demo *bool `yaml:"demo"`
}

func (s SeedConfig) DemoEnabled() bool {
	if s.Demo != nil {
		return *s.Demo
	}
	return true
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
	Addr    string `yaml:"addr"`
	WebRoot string `yaml:"web_root"` // empty = embedded UI; set to web/dist for external
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
	c.WarnInsecureIfNeeded()
	return &c, nil
}

var weakJWTSecrets = map[string]bool{
	"": true, "dev-only-change-me": true, "CHANGE_ME_IN_PRODUCTION": true,
	"secret": true, "jwt-secret": true, "change-me": true,
	"REPLACE_WITH_ENV_ERP_JWT_SECRET_32CHARS": true,
}

// IsProductionLike 非 demo 且非默认弱密钥开发配置时视为生产倾向。
func (c *Config) IsProductionLike() bool {
	return !c.Seed.DemoEnabled()
}

// WarnInsecureIfNeeded 生产倾向配置下对弱 JWT / 开放 CORS 打告警；致命弱密钥直接 Fatal。
func (c *Config) WarnInsecureIfNeeded() {
	demo := c.Seed.DemoEnabled()
	weakJWT := weakJWTSecrets[c.JWT.Secret] || len(c.JWT.Secret) < 16
	openCORS := len(c.CORS.AllowOrigins) == 0 ||
		(len(c.CORS.AllowOrigins) == 1 && (c.CORS.AllowOrigins[0] == "*" || strings.TrimSpace(c.CORS.AllowOrigins[0]) == ""))

	if demo {
		if weakJWT {
			log.Printf("[config] WARN demo mode: weak JWT secret (ok for local)")
		}
		if openCORS {
			log.Printf("[config] WARN demo mode: CORS allow_origins is open")
		}
		return
	}
	if weakJWT {
		log.Fatalf("[config] FATAL: production (seed.demo=false) requires a strong jwt.secret (len>=16, not a known placeholder)")
	}
	if openCORS {
		log.Printf("[config] WARN production: CORS allow_origins is open (*); tighten to explicit origins")
	}
}

func applyEnv(c *Config) {
	if v := os.Getenv("ERP_SERVER_ADDR"); v != "" {
		c.Server.Addr = v
	}
	if v := os.Getenv("ERP_WEB_ROOT"); v != "" {
		c.Server.WebRoot = v
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
	if v := os.Getenv("ERP_OAUTH_ENABLED"); v != "" {
		c.OAuth.Enabled = v == "1" || strings.EqualFold(v, "true")
	}
	if v := os.Getenv("ERP_OCR_ENABLED"); v != "" {
		c.OCR.Enabled = v == "1" || strings.EqualFold(v, "true")
	}
	if v := os.Getenv("ERP_OCR_PROVIDER"); v != "" {
		c.OCR.Provider = v
	}
}
