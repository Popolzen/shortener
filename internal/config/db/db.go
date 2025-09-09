package db

import (
	"database/sql"
	"flag"
	"log"

	"github.com/caarlos0/env"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DBConfig содержит конфигурацию для подключения к БД
type DBConfig struct {
	DBurl string `env:"DATABASE_DSN"`
}

// DataBase представляет подключение к базе данных
type DataBase struct {
	*sql.DB
	config *DBConfig
}

// NewDBConfig создает новую конфигурацию БД
func NewDBConfig() DBConfig {
	c := DBConfig{}

	c.getArgsFromCli()
	c.getArgsFromEnv()
	return c
}

// PingDB проверяет подключение к базе данных (без создания постоянного соединения)
func (d *DBConfig) PingDB() error {
	db, err := sql.Open("pgx", d.DBurl)
	if err != nil {
		log.Fatal("Ошибка при создании подключения:", err)
		return err
	}

	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatal("Ошибка при подключении к БД:", err)
		return err
	}

	return nil
}

func (c *DBConfig) getArgsFromCli() {
	flag.StringVar(&c.DBurl, "d", "", "db")

	flag.Parse()
}

func (c *DBConfig) getArgsFromEnv() {
	if err := env.Parse(c); err != nil {
		log.Fatal(err)
	}
}

func (c *DBConfig) IsEmpty() bool {
	return c.DBurl == ""
}
