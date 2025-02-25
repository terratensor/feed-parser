package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env             string         `yaml:"env" env-default:"development"`
	Workers         int            `yaml:"workers" env-default:"1"`
	Delay           *time.Duration `yaml:"delay" env-default:"60s"`
	RandomDelay     *time.Duration `yaml:"random_delay" env-default:"150s"`
	UserAgent       string         `yaml:"user_agent" env-default:"Concepts/1.0"`
	IndexNow        bool           `yaml:"index_now" env-default:"false"`
	ManticoreIndex  string         `yaml:"manticore_index"`
	EntryChanBuffer int            `yaml:"entry_chan_buffer" env-default:"20"`
	Splitter        Splitter       `yaml:"splitter"`
	Parsers         []Parser       `yaml:"parsers"`
}

type Splitter struct {
	OptChunkSize int `yaml:"opt_chunk_size" env-default:"1800"`
	MaxChunkSize int `yaml:"max_chunk_size" env-default:"3600"`
}

type Parser struct {
	Url         string         `yaml:"url"`
	Lang        string         `yaml:"lang"`
	ResourceID  int            `yaml:"resource_id"`
	UserAgent   string         `yaml:"user_agent,omitempty"`
	Delay       *time.Duration `yaml:"delay,omitempty"`
	RandomDelay *time.Duration `yaml:"random_delay,omitempty"`
	Crawler     Crawler        `yaml:"crawler"` // Конфигурация для краулера
}

type Crawler struct {
	RandomDelayMin int           `yaml:"random_delay_min" env-default:"10"`                                                                                                        // Минимальная задержка в секундах
	RandomDelayMax int           `yaml:"random_delay_max" env-default:"30"`                                                                                                        // Максимальная задержка в секундах
	SleepMin       int           `yaml:"sleep_min" env-default:"10"`                                                                                                               // Минимальное время сна в секундах
	SleepMax       int           `yaml:"sleep_max" env-default:"40"`                                                                                                               // Максимальное время сна в секундах
	UserAgent      string        `yaml:"user_agent" env-default:"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"` // User-Agent для запросов
	MaxRetries     int           `yaml:"max_retries" env-default:"5"`                                                                                                              // Максимальное количество попыток ревизита
	RetryDelay     time.Duration `yaml:"retry_delay" env-default:"2s"`                                                                                                             // Задержка между повторными попытками
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
