package config

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const configPathEnv = "SSO_CONFIG_PATH"

type Config struct {
	Env       string          `yaml:"env" env:"SSO_ENV" env-default:"local"`
	DbConfig  DbConfig        `yaml:"db" env-required:"true"`
	GRPC      GRPCConfig      `yaml:"grpc"`
	TokenTTL  time.Duration   `yaml:"token_ttl" env-required:"true"`
	Issuer    string          `yaml:"issuer" env-default:"go_sso_service"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type DbConfig struct {
	Driver          string        `yaml:"driver" env-default:"postgres"`
	Host            string        `yaml:"host" env:"SSO_DB_HOST"`
	Port            int           `yaml:"port" env:"SSO_DB_PORT"`
	User            string        `yaml:"user" env:"SSO_DB_USER"`
	Password        string        `yaml:"password" env:"SSO_DB_PASSWORD"`
	Dbname          string        `yaml:"dbname" env:"SSO_DB_NAME"`
	SSLMode         string        `yaml:"sslmode" env:"SSO_DB_SSLMODE" env-default:"require"`
	MaxOpenConns    int           `yaml:"max_open_conns" env-default:"25"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env-default:"25"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env-default:"30m"`
}

// LogValue keeps secrets out of logs: slog does not resolve LogValuer on nested fields.
func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("env", c.Env),
		slog.Any("db", c.DbConfig.LogValue()),
		slog.Int("grpc_port", c.GRPC.Port),
		slog.Duration("grpc_timeout", c.GRPC.Timeout),
		slog.Bool("tls", c.GRPC.TLSCertFile != ""),
		slog.Duration("token_ttl", c.TokenTTL),
		slog.String("issuer", c.Issuer),
		slog.Int("rate_limit_requests", c.RateLimit.Requests),
		slog.Duration("rate_limit_window", c.RateLimit.Window),
	)
}

// LogValue keeps the password out of logs.
func (c DbConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("driver", c.Driver),
		slog.String("host", c.Host),
		slog.Int("port", c.Port),
		slog.String("user", c.User),
		slog.String("dbname", c.Dbname),
		slog.String("sslmode", c.SSLMode),
	)
}

type GRPCConfig struct {
	Port int `yaml:"port" env:"SSO_GRPC_PORT" env-default:"50051"`
	// Timeout bounds the handling time of a single request.
	Timeout     time.Duration `yaml:"timeout" env-default:"10s"`
	TLSCertFile string        `yaml:"tls_cert_file" env:"SSO_TLS_CERT_FILE"`
	TLSKeyFile  string        `yaml:"tls_key_file" env:"SSO_TLS_KEY_FILE"`
}

// RateLimitConfig limits Login and Register attempts per client IP.
type RateLimitConfig struct {
	Requests int           `yaml:"requests" env-default:"10"`
	Window   time.Duration `yaml:"window" env-default:"1m"`
}

// MustLoad loads the config from the path given by the -config flag or the SSO_CONFIG_PATH env variable.
func MustLoad() *Config {
	var path string
	flag.StringVar(&path, "config", "", "path to config file")
	flag.Parse()

	return MustLoadPath(path)
}

// MustLoadPath loads the config from path, falling back to the SSO_CONFIG_PATH env variable when path is empty.
func MustLoadPath(path string) *Config {
	if path == "" {
		path = os.Getenv(configPathEnv)
	}
	if path == "" {
		log.Fatal("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("config file does not exist at path: %s", path)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	if cfg.TokenTTL <= 0 {
		log.Fatal("token_ttl must be positive")
	}
	if (cfg.GRPC.TLSCertFile == "") != (cfg.GRPC.TLSKeyFile == "") {
		log.Fatal("grpc.tls_cert_file and grpc.tls_key_file must be set together")
	}

	return &cfg
}
