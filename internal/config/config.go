package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env            string         `yaml:"env" env-default:"development"`
	TimeDelay      *time.Duration `yaml:"time_delay" env-default:"1m"`
	ManticoreIndex string         `yaml:"manticore_index"`
	SaveToFile     bool           `yaml:"save_to_file"`
	OutputPath     string         `yaml:"output_path"`
	Parsers        struct {
		Kremlin Kremlin `yaml:"kremlin"`
		//Mid     Mid     `yaml:"mid"`
		//Mil     Mil     `yaml:"mil"`
	} `yaml:"parsers"`
}

type StartURL struct {
	Url  string `yaml:"url"`
	Lang string `yaml:"lang"`
}

type Kremlin Parser
type Mid Parser
type Mil Parser

type Parser struct {
	ResourceID int            `yaml:"resource_id" env-default:"1"`
	ParseDelay *time.Duration `yaml:"parse_delay" env-default:"2s"`
	StartURLs  []StartURL     `yaml:"start_urls"`
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
