package config

import "flag"

type Config struct {
	Host string
	Port string
}

func (c *Config) getArgsFromCli() {
	flag.StringVar(&c.Host, "a", "localhost", "server host")
	flag.StringVar(&c.Port, "b", "8080", "server Port")
	flag.Parse()
}

func NewConfig() *Config {
	c := &Config{
		Host: "localhost",
		Port: "8080",
	}
	c.getArgsFromCli()
	return c
}

// Возвращает полнгый адрес сервера
func (c Config) Address() string {
	return c.Host + ":" + c.Port
}
