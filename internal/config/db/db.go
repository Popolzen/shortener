package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/Popolzen/shortener/internal/config"
	migration "github.com/Popolzen/shortener/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DBConfig содержит конфигурацию для подключения к БД
type DBConfig struct {
	DBurl string
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

// NewDataBase создает абстракцию БД
func NewDataBase(c config.Config) (*DataBase, error) {
	cfg := NewDBConfig(c)
	db, err := sql.Open("pgx", cfg.DBurl)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть подключение: %w", err)
	}
	return &DataBase{
		DB:     db,
		config: &cfg,
	}, nil
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

func (d *DataBase) Migrate() error {
	return migration.MigrateUp(d.DB)
}
