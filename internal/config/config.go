package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env"
)

const (
	DefaultServerAddr    = ":8080"
	DefaultBaseURL       = "http://localhost:8080"
	DefaultFilePath      = "storage.json"
	DefaultAuditFilePath = "audit_storage.json"
	DefaultPprofAddr     = "localhost:6060"
)

type Config struct {
	ServerAddr  string `env:"SERVER_ADDRESS"`
	BaseURL     string `env:"BASE_URL"`
	FilePath    string `env:"FILE_STORAGE_PATH"`
	DBurl       string `env:"DATABASE_DSN"`
	SecretKey   string `env:"KEY"`
	AuditFile   string `env:"AUDIT_FILE"`
	AuditURL    string `env:"AUDIT_URL"`
	PprofAddr   string `env:"PPROF_ADDRESS"`
	EnableHTTPS bool   `env:"ENABLE_HTTPS"`
	CertFile    string `env:"CERT_FILE"`
	KeyFile     string `env:"KEY_FILE"`
}

func (c *Config) getArgsFromCli() {
	flag.StringVar(&c.ServerAddr, "a", DefaultServerAddr, "server host")
	flag.StringVar(&c.BaseURL, "b", DefaultBaseURL, "base url for short links")
	flag.StringVar(&c.FilePath, "f", DefaultFilePath, "file storage path")
	flag.StringVar(&c.DBurl, "d", "", "DBurl")
	flag.StringVar(&c.SecretKey, "k", "", "secret key")
	flag.StringVar(&c.AuditFile, "audit-file", DefaultAuditFilePath, "audit file path")
	flag.StringVar(&c.AuditURL, "audit-url", "", "audit server URL")
	flag.StringVar(&c.PprofAddr, "pprof", DefaultPprofAddr, "pprof server address")
	flag.BoolVar(&c.EnableHTTPS, "s", false, "enable HTTPS")
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
		PprofAddr:  DefaultPprofAddr,
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

func (c Config) GetAuditFile() string {
	return c.AuditFile
}

func (c Config) GetAuditURL() string {
	return c.AuditURL
}
