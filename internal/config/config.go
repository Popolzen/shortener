package config

import "flag"

type Config struct {
	ServerAddr string
	BaseURL    string
}

func (c *Config) getArgsFromCli() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "server host")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "base url for short links")
	flag.Parse()
}

func NewConfig() *Config {
	c := &Config{
		ServerAddr: ":8080", // значения по умолчанию
		BaseURL:    "http://localhost:8080",
	}
	c.getArgsFromCli()
	return c
}

// Возвращает полнгый адрес сервера
func (c Config) Address() string {
	return c.ServerAddr
}

func (c Config) GetBaseURL() string {
	return c.BaseURL
}
