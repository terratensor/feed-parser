package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env             string         `yaml:"env" env-default:"development"`
	Workers         int            `yaml:"workers" env-default:"5"`
	Delay           *time.Duration `yaml:"delay" env-default:"60s"`
	RandomDelay     *time.Duration `yaml:"random_delay"  env-default:"150s"`
	ManticoreIndex  string         `yaml:"manticore_index"`
	EntryChanBuffer int            `yaml:"entry_chan_buffer" env-default:"20"`
	URLS            []Link         `yaml:"parsers"`
}

type Link struct {
	Url        string `yaml:"url"`
	Lang       string `yaml:"lang"`
	ResourceID int    `yaml:"resource_id"`
	UserAgent  string `yaml:"user_agent"`
}

func MustLoad() *Config {
	// Получаем путь до конфиг-файла из env-переменной CONFIG_PATH
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH environment variable is not set")
	}

	// Проверяем существование конфиг-файла
	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("error opening config file: %s", err)
	}

	var cfg Config

	// Читаем конфиг-файл и заполняем нашу структуру
	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("error reading config file: %s", err)
	}

	return &cfg
}
