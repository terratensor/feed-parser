# feed-parser

Сервис позволяет периодически в заданный промежуток времени, например, 1-2 минуты 
опрашивать список RSS лент сайтов. Сервис читает ленту, разбирает ее, создает из
элементов ленты объект единого вида Entry, который сохраняет в индексе
ManticoreSearch. Если в ленте нет контента, то сервис идет по ссылке записи, скачивает страницу, разбирает ее и создает объект стандартного вида Entry. 
На данный момент реализованы парсеры сайтов mid.ru, kremlin.ru, mil.ru 
и их языковые версии.

Существенное отличие от прошлых реализация парсеров — универсальность, 
можно подключить rss ленту любого сайта или агентства новостей, например tass.ru.
В инженерном плане применена концепция worker poll — пул воркеров, при старте службы, сервис читает ссылки лент и запускает для каждой ссылки парсер, парсеры работают в параллельном режиме. 

```go
urls := []Link{
    {Url: "http://kremlin.ru/events/all/feed/", Lang: "ru", ResourceID: 1},
    {Url: "http://en.kremlin.ru/events/all/feed", Lang: "en", ResourceID: 1},
    {Url: "https://mid.ru/ru/rss.php", Lang: "ru", ResourceID: 2},
    {Url: "https://mid.ru/en/rss.php", Lang: "en", ResourceID: 2},
    {Url: "https://mid.ru/de/rss.php", Lang: "de", ResourceID: 2},
    {Url: "https://mid.ru/fr/rss.php", Lang: "fr", ResourceID: 2},
    {Url: "https://mid.ru/es/rss.php", Lang: "es", ResourceID: 2},
    {Url: "https://mid.ru/pt/rss.php", Lang: "pt", ResourceID: 2},
    {Url: "https://function.mil.ru/rss_feeds/reference_to_general.htm?contenttype=xml", Lang: "ru", ResourceID: 3},
}
```

Парсеры с периодичностью опрашиват свои ссылки-ленты, читают их и создают из записей ленты задачи. Каждая запись-задача отправляется в очередь на обработку, которую осуществляют worker-ы обработчики, они так же работают параллельно. Количество воркеров по-умолчанию 5.

Каждый обработчик берет задачу из очереди, и обрабатывает её. Делает запрос в индекс ManticoreSearch, ищет эту запись, если не находит, то создает новую запись, и при необходимости запускает crawler, который идет за контентом (сайты mil.ru, mid.ru). Обычно в ленте содержится только заголовок и короткое описание, за статьей обработчик отправляется на сайт и парсит страницу. 

Если при разборе ленты и запросе в базу ManticoreSearch, запись уже существует, то осуществляется проверка, по полям updated, если поля в ленте и в базе различаются, то обработчик сохраняет обновленную версию события. По наблюдениям такое довольно часто происходит, так как статьи дополняются со временем, такое точно замечено на сайтах kremlin.ru и mid.ru

Когда все задачи обработаны, обработчики ожидают задачи в очереди, работают в фоне в ожидании записей из лент.

<details><summary>Диаграмма процессов</summary>
<p>

![diagram drawio](https://github.com/terratensor/kremlin-parser/assets/10896447/75e21cac-10a4-4820-a8c1-56acf0745b99)

</p>
</details> 

### Реализовано

- парсер rss ленты, для этого надо предать адрес ленты парсеру, поддерживаются языковые ленты kremlin.ru (ru,en), mid.ru (ru, en, de, fr, es, pt). mil.ru (ru)
- добавление новых записей из ленты событий в мантикору
- обновление существующих записей из ленты в мантикору по условию, обновления роля updated.


#### TODO
- бэкенд сервер для поиска.
- фронтенд для поиска.

### Используемые библиотеки

- [Manticorer Go client. Сlient for Manticore Search.](https://github.com/manticoresoftware/manticoresearch-go)
- [Gofeed: A Robust Feed Parser for Golang](https://github.com/mmcdole/gofeed)
- [Colly Lightning Fast and Elegant Scraping Framework for Gophers](https://github.com/gocolly/colly/tree/master)
- [goquery - a little like that j-thing, only in Go](https://github.com/PuerkitoBio/goquery)

Полный список зависимостей в файле [go.mod](https://github.com/terratensor/feed-parser/blob/a279808983af6ade816521b8d4c2751ac2de45d5/go.mod)
    
