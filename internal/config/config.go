package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env"
)

type Config struct {
	ServerAddr string `env:"SERVER_ADDRESS"`
	BaseURL    string `env:"BASE_URL"`
}

func (c *Config) getArgsFromCli() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "server host")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "base url for short links")
	flag.Parse()
}
func (c *Config) getArgsFromEnv() {
	err := env.Parse(c)
	if err != nil {
		log.Fatal(err)
	}
}

func NewConfig() *Config {
	c := &Config{
		ServerAddr: ":8080", // значения по умолчанию
		BaseURL:    "http://localhost:8080",
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
