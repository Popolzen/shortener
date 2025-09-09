package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/Popolzen/shortener/internal/config"
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
func NewDBConfig(c config.Config) DBConfig {
	fmt.Print("Строка коннекта", c.DBurl)
	return DBConfig{
		DBurl: c.DBurl,
	}
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
