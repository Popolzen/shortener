package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env"
)

const (
	DefaultServerAddr = ":8080"
	DefaultBaseURL    = "http://localhost:8080"
	DefaultFilePath   = "storage.json"
)

type Config struct {
	ServerAddr string `env:"SERVER_ADDRESS"`
	BaseURL    string `env:"BASE_URL"`
	FilePath   string `env:"FILE_STORAGE_PATH"`
	DBurl      string `env:"DATABASE_DSN"`
	SecretKey  string `env:"KEY"`
}

func (c *Config) getArgsFromCli() {
	flag.StringVar(&c.ServerAddr, "a", DefaultServerAddr, "server host")
	flag.StringVar(&c.BaseURL, "b", DefaultBaseURL, "base url for short links")
	flag.StringVar(&c.FilePath, "f", DefaultFilePath, "file storage path")
	flag.StringVar(&c.DBurl, "d", "", "DBurl")
	flag.StringVar(&c.SecretKey, "k", "", "secret key")
	flag.Parse()
}

func (c *Config) getArgsFromEnv() {
	if err := env.Parse(c); err != nil {
		log.Fatal(err)
	}
}

func NewConfig() *Config {
	c := &Config{
		ServerAddr: DefaultServerAddr,
		BaseURL:    DefaultBaseURL,
		FilePath:   DefaultFilePath,
	}
	c.getArgsFromCli()
	c.getArgsFromEnv()
	return c
}

// Возвращает полнгый адрес сервера
func (c Config) GetAddress() string {
	return c.ServerAddr
}

func (c Config) GetBaseURL() string {
	return c.BaseURL
}

func (c Config) GetFilePath() string {
	return c.FilePath
}
