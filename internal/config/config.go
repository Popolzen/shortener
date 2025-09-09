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

type ServerConfig struct {
	ServerAddr string `env:"SERVER_ADDRESS"`
	BaseURL    string `env:"BASE_URL"`
	FilePath   string `env:"FILE_STORAGE_PATH"`
}

func (c *ServerConfig) getArgsFromCli() {
	flag.StringVar(&c.ServerAddr, "a", DefaultServerAddr, "server host")
	flag.StringVar(&c.BaseURL, "b", DefaultBaseURL, "base url for short links")
	flag.StringVar(&c.FilePath, "f", DefaultFilePath, "file storage path")
	flag.Parse()
}

func (c *ServerConfig) getArgsFromEnv() {
	if err := env.Parse(c); err != nil {
		log.Fatal(err)
	}
}

func NewConfig() *ServerConfig {
	c := &ServerConfig{
		ServerAddr: DefaultServerAddr,
		BaseURL:    DefaultBaseURL,
		FilePath:   DefaultFilePath,
	}
	c.getArgsFromCli()
	c.getArgsFromEnv()
	return c
}

// Возвращает полнгый адрес сервера
func (c ServerConfig) GetAddress() string {
	return c.ServerAddr
}

func (c ServerConfig) GetBaseURL() string {
	return c.BaseURL
}

func (c ServerConfig) GetFilePath() string {
	return c.FilePath
}
