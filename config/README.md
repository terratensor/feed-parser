# Конфигурация проекта

Этот документ описывает структуру конфигурационного файла проекта и его использование.

## Содержание
1. [Структура конфигурации](#структура-конфигурации)
2. [Пример конфигурации](#пример-конфигурации)
3. [Использование конфигурации](#использование-конфигурации)
4. [Ссылки](#ссылки)

---

## Структура конфигурации

Конфигурационный файл проекта написан в формате YAML. Он содержит следующие основные разделы:

### Основные параметры
- **`env`**: Окружение (`local`, `dev`, `prod`). По умолчанию: `development`.
- **`workers`**: Количество рабочих горутин. По умолчанию: `1`.
- **`delay`**: Задержка между запросами. По умолчанию: `60s`.
- **`random_delay`**: Случайная задержка между запросами. По умолчанию: `150s`.
- **`user_agent`**: User-Agent для HTTP-запросов. По умолчанию: `Concepts/1.0`.
- **`index_now`**: Флаг для включения/выключения индексации. По умолчанию: `false`.
- **`manticore_index`**: Название индекса Manticore.
- **`entry_chan_buffer`**: Размер буфера канала для записей. По умолчанию: `20`.

### Раздел `splitter`
- **`opt_chunk_size`**: Оптимальный размер фрагмента контента для поиска. По умолчанию: `1800`.
- **`max_chunk_size`**: Максимальный размер фрагмента контента для поиска. По умолчанию: `3600`.

### Раздел `parsers`
Список парсеров, каждый из которых содержит:
- **`url`**: URL источника данных.
- **`lang`**: Язык контента.
- **`resource_id`**: Уникальный идентификатор ресурса.
- **`user_agent`**: User-Agent для конкретного парсера (опционально).
- **`delay`**: Задержка для конкретного парсера (опционально).
- **`random_delay`**: Случайная задержка для конкретного парсера (опционально).
- **`crawler`**: Конфигурация краулера для парсера (опционально).

#### Конфигурация краулера (`crawler`)
- **`random_delay_min`**: Минимальная задержка в секундах. По умолчанию: `10`.
- **`random_delay_max`**: Максимальная задержка в секундах. По умолчанию: `30`.
- **`sleep_min`**: Минимальное время сна в секундах. По умолчанию: `10`.
- **`sleep_max`**: Максимальное время сна в секундах. По умолчанию: `40`.
- **`user_agent`**: User-Agent для запросов краулера. По умолчанию: `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36`.
- **`max_retries`**: Максимальное количество попыток повторного запроса. По умолчанию: `5`.
- **`retry_delay`**: Задержка между повторными попытками. По умолчанию: `2s`.

---

## Пример конфигурации

```yaml
env: "prod"
workers: 1
delay: 3600s
random_delay: 3600s
user_agent: "Concepts/1.0"
manticore_index: feed
entry_chan_buffer: 20

splitter:
opt_chunk_size: 1800
max_chunk_size: 3600

parsers:
- url: "http://kremlin.ru/events/all/feed/"
lang: "ru"
resource_id: 1

- url: "http://en.kremlin.ru/events/all/feed"
lang: "en"
resource_id: 1

- url: "https://mid.ru/ru/rss.php"
lang: "ru"
resource_id: 2

- url: "https://function.mil.ru/rss_feeds/reference_to_general.htm?contenttype=xml"
lang: "ru"
resource_id: 3
user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"
crawler:
random_delay_min: 10
random_delay_max: 30
sleep_min: 10
sleep_max: 40
user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36"
max_retries: 5
retry_delay: 2s
```

---

## Использование конфигурации

### Загрузка конфигурации
Для загрузки конфигурации используйте функцию `MustLoad` из пакета `config`:

```go
import "your-project/config"

func main() {
cfg := config.MustLoad()
fmt.Println("Environment:", cfg.Env)
}
```

### Получение конфигурации краулера
Для получения конфигурации краулера по `resource_id` и `lang` используйте функцию `GetCrawlerConfigByResourceID`:

```go
crawlerConfig, err := config.GetCrawlerConfigByResourceID(cfg, 3, "ru")
if err != nil {
log.Fatalf("Error getting crawler config: %v", err)
}

// Использование конфигурации
result, err := VisitMil(entry, *crawlerConfig)
if err != nil {
log.Fatalf("Error: %v", err)
}
```

---

## Ссылки
- [Общий README проекта](../README.md)
- [Документация по YAML](https://yaml.org/)

---
