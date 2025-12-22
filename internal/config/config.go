package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/caarlos0/env"
)

const (
	DefaultServerAddr    = ":8080"
	DefaultBaseURL       = "http://localhost:8080"
	DefaultFilePath      = "storage.json"
	DefaultAuditFilePath = "audit_storage.json"
	DefaultPprofAddr     = "localhost:6060"
)

// Config содержит конфигурацию приложения
type Config struct {
	ServerAddr  string `json:"server_address" env:"SERVER_ADDRESS"`
	BaseURL     string `json:"base_url" env:"BASE_URL"`
	FilePath    string `json:"file_storage_path" env:"FILE_STORAGE_PATH"`
	DBurl       string `json:"database_dsn" env:"DATABASE_DSN"`
	SecretKey   string `env:"KEY"`
	AuditFile   string `env:"AUDIT_FILE"`
	AuditURL    string `env:"AUDIT_URL"`
	PprofAddr   string `env:"PPROF_ADDRESS"`
	EnableHTTPS bool   `json:"enable_https" env:"ENABLE_HTTPS"`
	CertFile    string `env:"CERT_FILE"`
	KeyFile     string `env:"KEY_FILE"`
}

func NewConfig() *Config {
	c := &Config{
		ServerAddr: DefaultServerAddr,
		BaseURL:    DefaultBaseURL,
		FilePath:   DefaultFilePath,
		PprofAddr:  DefaultPprofAddr,
		AuditFile:  DefaultAuditFilePath,
	}

	configFile := getConfigPath()
	c.loadFromFile(configFile)
	c.getArgsFromEnv()
	c.getArgsFromCli()

	return c
}

func getConfigPath() string {
	for i, arg := range os.Args {
		if (arg == "-c" || arg == "-config") && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return os.Getenv("CONFIG")
}

func (c *Config) loadFromFile(filename string) {
	if filename == "" {
		return
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	json.Unmarshal(data, c)
}

func (c *Config) getArgsFromCli() {
	flag.StringVar(&c.ServerAddr, "a", c.ServerAddr, "server host")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "base url for short links")
	flag.StringVar(&c.FilePath, "f", c.FilePath, "file storage path")
	flag.StringVar(&c.DBurl, "d", c.DBurl, "database DSN")
	flag.StringVar(&c.SecretKey, "k", c.SecretKey, "secret key")
	flag.StringVar(&c.AuditFile, "audit-file", c.AuditFile, "audit file path")
	flag.StringVar(&c.AuditURL, "audit-url", c.AuditURL, "audit server URL")
	flag.StringVar(&c.PprofAddr, "pprof", c.PprofAddr, "pprof server address")
	flag.BoolVar(&c.EnableHTTPS, "s", c.EnableHTTPS, "enable HTTPS")
	flag.String("c", "", "config file path")
	flag.String("config", "", "config file path")
	flag.Parse()
}

func (c *Config) getArgsFromEnv() {
	if err := env.Parse(c); err != nil {
		log.Fatal(err)
	}
}

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
