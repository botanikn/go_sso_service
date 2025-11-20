package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string        `yaml:"env" env-default:"local"`
	DbConfig DbConfig      `yaml:"db" env-required:"true"`
	GRPC     GRPCConfig    `yaml:"grpc"`
	TokenTTL time.Duration `yaml:"token_ttl"`
}

// COMMENT структуру можно сделать приватной, особеность cleanenv, что поля нет, но при этом все равно стоит получать их через методы
type DbConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Dbname   string `yaml:"dbname"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

func MustLoad() *Config {
	path := fetchConfigPath()
	if path == "" {
		// COMMENT  не паникуй
		panic("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// COMMENT  не паникуй
		panic(fmt.Sprintf("config file does not exist at path: %s", path))
	}

	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		// COMMENT  не паникуй
		panic(fmt.Sprintf("failed to read config file: %v", err.Error()))
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string
	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("SSO_CONFIG_PATH")
	}

	return res
}
