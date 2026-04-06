package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Config хранит настройки утилиты
type Config struct {
	MaxDepth    int
	Concurrency int
	OutputDir   string
	Timeout     time.Duration
	Domain      string // основной домен (нормализованный)
}

// Task описывает задачу на загрузку
type Task struct {
	URL    string
	IsPage bool
	Depth  int
}

// Crawler управляет процессом зеркалирования
type Crawler struct {
	config    *Config
	client    *http.Client
	visited   sync.Map // map[string]bool для всех URL (страницы и ресурсы)
	taskQueue chan Task
	wg        sync.WaitGroup
	host      string // хост основного домена (без схемы)
}

// NewCrawler создает новый экземпляр обходчика
func NewCrawler(cfg *Config) *Crawler {
	return &Crawler{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		taskQueue: make(chan Task, cfg.Concurrency*2),
		host:      "",
	}
}

// normalizeURL приводит URL к каноническому виду: убирает фрагмент, нормализует слеши, схему в нижний регистр
func normalizeURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	// Убираем фрагмент (якорь)
	u.Fragment = ""
	// Приводим схему к нижнему регистру
	u.Scheme = strings.ToLower(u.Scheme)
	// Убираем дублирующие слеши в пути
	u.Path = path.Clean(u.Path)
	if u.Path == "." || u.Path == "/" {
		u.Path = "/"
	}
	// Если путь пуст, ставим "/"
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), nil
}

// sameDomain проверяет, принадлежит ли URL тому же домену (с учётом www)
func sameDomain(baseURL, targetURL string) bool {
	base, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	target, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	// Нормализуем хосты: убираем префикс www.
	baseHost := strings.TrimPrefix(base.Hostname(), "www.")
	targetHost := strings.TrimPrefix(target.Hostname(), "www.")
	return baseHost == targetHost
}

// getLocalPath возвращает локальный путь для сохранения файла относительно outputDir
// isPage указывает, что это HTML-страница (тогда добавим .html при необходимости)
func getLocalPath(rawURL string, outputDir string, isPage bool) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	// Строим путь: домен + путь
	host := strings.TrimPrefix(u.Hostname(), "www.")
	// Убираем параметры запроса для пути (файлы с параметрами не сохраняем)
	cleanPath := u.Path
	if u.RawQuery != "" {
		cleanPath = cleanPath + "?" + u.RawQuery // но в имени файла нельзя ? поэтому лучше не включать
		// Заменим ? на _ для допустимости
		cleanPath = strings.ReplaceAll(cleanPath, "?", "_")
	}
	// Заменяем недопустимые символы для Windows/Linux
	cleanPath = strings.ReplaceAll(cleanPath, ":", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "*", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "|", "_")

	if isPage {
		// Для страниц: если путь заканчивается на "/", добавляем index.html
		if strings.HasSuffix(cleanPath, "/") {
			cleanPath = cleanPath + "index.html"
		} else if cleanPath == "" || cleanPath == "/" {
			cleanPath = "/index.html"
		} else {
			// Если нет расширения .html или .htm, добавляем .html
			ext := strings.ToLower(path.Ext(cleanPath))
			if ext != ".html" && ext != ".htm" {
				cleanPath = cleanPath + ".html"
			}
		}
	}
	// Убираем начальный слеш для объединения
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	localPath := filepath.Join(outputDir, host, cleanPath)
	return localPath, nil
}

// downloadFile загружает файл по URL и сохраняет его по локальному пути
func (c *Crawler) downloadFile(task Task) error {
	// Проверяем, не загружали ли уже
	if _, ok := c.visited.Load(task.URL); ok {
		return nil
	}
	c.visited.Store(task.URL, true)

	// Выполняем запрос
	resp, err := c.client.Get(task.URL)
	if err != nil {
		return fmt.Errorf("request error for %s: %w", task.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status %d for %s", resp.StatusCode, task.URL)
	}

	// Определяем, является ли контент HTML (по Content-Type)
	contentType := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html") || (task.IsPage && strings.HasSuffix(task.URL, ".html"))

	// Получаем локальный путь
	localPath, err := getLocalPath(task.URL, c.config.OutputDir, isHTML)
	if err != nil {
		return fmt.Errorf("cannot get local path for %s: %w", task.URL, err)
	}

	// Создаём директорию
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("cannot read body: %w", err)
	}

	// Если это HTML, обрабатываем ссылки и ресурсы
	if isHTML {
		// Изменяем ссылки на локальные
		modifiedHTML, links, resources, err := c.processHTML(body, task.URL)
		if err != nil {
			return fmt.Errorf("cannot process HTML %s: %w", task.URL, err)
		}
		// Сохраняем изменённый HTML
		if err := os.WriteFile(localPath, []byte(modifiedHTML), 0644); err != nil {
			return fmt.Errorf("cannot write HTML file: %w", err)
		}
		// Добавляем новые задачи для страниц (ссылки) и ресурсов
		c.enqueueLinks(links, task.Depth+1)
		c.enqueueResources(resources)
	} else {
		// Не HTML – сохраняем как есть
		if err := os.WriteFile(localPath, body, 0644); err != nil {
			return fmt.Errorf("cannot write file: %w", err)
		}
	}
	return nil
}

// processHTML разбирает HTML, заменяет ссылки на локальные и возвращает модифицированный HTML,
// а также списки ссылок на страницы и ресурсы для дальнейшей загрузки.
func (c *Crawler) processHTML(htmlData []byte, baseURL string) (string, []string, []string, error) {
	doc, err := html.Parse(bytes.NewReader(htmlData))
	if err != nil {
		return "", nil, nil, err
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", nil, nil, err
	}

	var links []string     // ссылки на другие страницы (того же домена)
	var resources []string // ссылки на ресурсы (img, script, link[stylesheet])

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
				for i, attr := range n.Attr {
					if attr.Key == "href" {
						href := attr.Val
						absURL := resolveURL(base, href)
						if absURL == "" {
							continue
						}
						// Проверяем, что ссылка ведёт на тот же домен и не является ресурсом (по расширению)
						if sameDomain(baseURL, absURL) {
							// Нормализуем URL
							normURL, _ := normalizeURL(absURL)
							if normURL != "" {
								// Для простоты считаем все ссылки a страницами (HTML)
								links = append(links, normURL)
								// Заменяем href на локальный путь (страница)
								localPath, _ := getLocalPath(normURL, c.config.OutputDir, true)
								relPath, _ := filepath.Rel(filepath.Dir(getLocalPathForBase(baseURL, c.config.OutputDir)), localPath)
								relPath = filepath.ToSlash(relPath)
								n.Attr[i].Val = relPath
							}
						}
						break
					}
				}
			case "img":
				for i, attr := range n.Attr {
					if attr.Key == "src" {
						src := attr.Val
						absURL := resolveURL(base, src)
						if absURL == "" {
							continue
						}
						if sameDomain(baseURL, absURL) {
							normURL, _ := normalizeURL(absURL)
							if normURL != "" {
								resources = append(resources, normURL)
								localPath, _ := getLocalPath(normURL, c.config.OutputDir, false)
								relPath, _ := filepath.Rel(filepath.Dir(getLocalPathForBase(baseURL, c.config.OutputDir)), localPath)
								relPath = filepath.ToSlash(relPath)
								n.Attr[i].Val = relPath
							}
						}
						break
					}
				}
			case "script":
				for i, attr := range n.Attr {
					if attr.Key == "src" {
						src := attr.Val
						absURL := resolveURL(base, src)
						if absURL == "" {
							continue
						}
						if sameDomain(baseURL, absURL) {
							normURL, _ := normalizeURL(absURL)
							if normURL != "" {
								resources = append(resources, normURL)
								localPath, _ := getLocalPath(normURL, c.config.OutputDir, false)
								relPath, _ := filepath.Rel(filepath.Dir(getLocalPathForBase(baseURL, c.config.OutputDir)), localPath)
								relPath = filepath.ToSlash(relPath)
								n.Attr[i].Val = relPath
							}
						}
						break
					}
				}
			case "link":
				var rel, href string
				for _, attr := range n.Attr {
					if attr.Key == "rel" {
						rel = attr.Val
					}
					if attr.Key == "href" {
						href = attr.Val
					}
				}
				if rel == "stylesheet" && href != "" {
					absURL := resolveURL(base, href)
					if absURL == "" {
						break
					}
					if sameDomain(baseURL, absURL) {
						normURL, _ := normalizeURL(absURL)
						if normURL != "" {
							resources = append(resources, normURL)
							localPath, _ := getLocalPath(normURL, c.config.OutputDir, false)
							relPath, _ := filepath.Rel(filepath.Dir(getLocalPathForBase(baseURL, c.config.OutputDir)), localPath)
							relPath = filepath.ToSlash(relPath)
							// Обновляем атрибут href
							for i := range n.Attr {
								if n.Attr[i].Key == "href" {
									n.Attr[i].Val = relPath
									break
								}
							}
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// Рендерим изменённый HTML
	var buf bytes.Buffer
	err = html.Render(&buf, doc)
	if err != nil {
		return "", nil, nil, err
	}
	return buf.String(), links, resources, nil
}

// resolveURL преобразует относительный URL в абсолютный на основе базового
func resolveURL(base *url.URL, ref string) string {
	refURL, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	absURL := base.ResolveReference(refURL)
	return absURL.String()
}

// getLocalPathForBase возвращает локальный путь для базового URL (используется для вычисления относительных путей)
func getLocalPathForBase(rawURL, outputDir string) string {
	path, _ := getLocalPath(rawURL, outputDir, true)
	return path
}

// enqueueLinks добавляет ссылки на страницы в очередь, если глубина не превышена
func (c *Crawler) enqueueLinks(links []string, depth int) {
	if depth > c.config.MaxDepth && c.config.MaxDepth >= 0 {
		return
	}
	for _, link := range links {
		if _, ok := c.visited.Load(link); !ok {
			c.wg.Add(1)
			go func(url string, d int) {
				c.taskQueue <- Task{URL: url, IsPage: true, Depth: d}
			}(link, depth)
		}
	}
}

// enqueueResources добавляет ресурсы в очередь (глубина не важна)
func (c *Crawler) enqueueResources(resources []string) {
	for _, res := range resources {
		if _, ok := c.visited.Load(res); !ok {
			c.wg.Add(1)
			go func(url string) {
				c.taskQueue <- Task{URL: url, IsPage: false, Depth: 0}
			}(res)
		}
	}
}

// worker обрабатывает задачи из очереди
func (c *Crawler) worker() {
	for task := range c.taskQueue {
		err := c.downloadFile(task)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		}
		c.wg.Done()
	}
}

// Run запускает обход сайта
func (c *Crawler) Run(startURL string) error {
	// Нормализуем стартовый URL
	normStart, err := normalizeURL(startURL)
	if err != nil {
		return fmt.Errorf("invalid start URL: %w", err)
	}
	// Сохраняем хост для проверки домена
	u, _ := url.Parse(normStart)
	c.host = strings.TrimPrefix(u.Hostname(), "www.")
	c.config.Domain = c.host

	// Запускаем воркеры
	for i := 0; i < c.config.Concurrency; i++ {
		go c.worker()
	}

	// Отправляем первую задачу
	c.wg.Add(1)
	c.taskQueue <- Task{URL: normStart, IsPage: true, Depth: 0}

	// Ожидаем завершения всех задач
	c.wg.Wait()
	close(c.taskQueue)

	return nil
}

// main разбирает аргументы и запускает краулер
func main() {
	var (
		depth       = flag.Int("depth", 2, "максимальная глубина рекурсии (0 - только начальная страница, -1 - без ограничений)")
		concurrency = flag.Int("concurrency", 5, "количество одновременных загрузок")
		output      = flag.String("output", "./site", "директория для сохранения")
		timeoutSec  = flag.Int("timeout", 10, "таймаут HTTP-запроса в секундах")
	)
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <URL>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	startURL := flag.Arg(0)

	cfg := &Config{
		MaxDepth:    *depth,
		Concurrency: *concurrency,
		OutputDir:   *output,
		Timeout:     time.Duration(*timeoutSec) * time.Second,
	}

	crawler := NewCrawler(cfg)
	fmt.Printf("Starting mirror of %s (depth=%d, concurrency=%d, output=%s)\n",
		startURL, cfg.MaxDepth, cfg.Concurrency, cfg.OutputDir)

	if err := crawler.Run(startURL); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Mirroring completed.")
}
