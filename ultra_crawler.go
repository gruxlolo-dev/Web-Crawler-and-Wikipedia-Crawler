package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"golang.org/x/net/html"
)

// NOTE: You can configure system
// TODO: Create mysql database
// TODO: Configure database in env file
// TODO: Configure Settings in code
// TODO: Configure start links
// TODO: IF YOU NOT HAVE GOLANG THEN INSTALL
// NOTE: You can use webcrawler for everthing

// NOTE: How is work
// 1. Start start urls
// 2. Typing what is theme of url
// 3. Save in database
// 4. Timeout
// 5. Next Url

// configuration
const (
	maxURLs    = 1000000000000000000 // max urls to stop bot
	numWorkers = 100                 // workers number for wikipedia i prefer 5 workers
	timeout    = 5 * time.Second     // time cooldown
)

type UltraCrawler struct {
	db       *sql.DB
	urlQueue chan string
	seen     sync.Map
	total    int64
	start    time.Time
	wg       sync.WaitGroup
}

func main() {
	godotenv.Load()

	fmt.Printf("Starting crawler with %d workers...\n", numWorkers)
	crawler := &UltraCrawler{
		urlQueue: make(chan string, 100000),
		start:    time.Now(),
	}

	crawler.db = initDB()
	defer crawler.db.Close()

	startURLs := []string{
		"https://en.wikipedia.org/wiki/Special:WhatLinksHere?target=Programing&namespace=&limit=599",
	}

	// start workers
	for i := 0; i < numWorkers; i++ {
		crawler.wg.Add(1)
		go crawler.worker()
	}

	// add start url
	for _, url := range startURLs {
		crawler.addURL(url)
	}
	fmt.Printf("Added %d start URLs\n", len(startURLs))

	// speed monitor
	go crawler.monitor()

	// wait
	done := make(chan bool)
	go func() {
		crawler.wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		total := atomic.LoadInt64(&crawler.total)
		elapsed := time.Since(crawler.start).Seconds()
		fmt.Printf("FINISHED: %d URLs in %.1fs (%.1f/s)\n", total, elapsed, float64(total)/elapsed)
	case <-time.After(10 * time.Minute):
		total := atomic.LoadInt64(&crawler.total)
		elapsed := time.Since(crawler.start).Seconds()
		fmt.Printf("TIMEOUT: %d URLs in %.1fs (%.1f/s)\n", total, elapsed, float64(total)/elapsed)
	}

	close(crawler.urlQueue)
}

func initDB() *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxOpenConns(2000)
	db.SetMaxIdleConns(1000)
	db.SetConnMaxLifetime(time.Hour)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id INT AUTO_INCREMENT PRIMARY KEY,
		url VARCHAR(767) UNIQUE,
		title VARCHAR(500),
		category VARCHAR(200)
	)`)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func (c *UltraCrawler) addURL(url string) {
	if _, loaded := c.seen.LoadOrStore(url, true); !loaded {
		select {
		case c.urlQueue <- url:
		default:
		}
	}
}

func (c *UltraCrawler) worker() {
	defer c.wg.Done()

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     30 * time.Second,
			DisableKeepAlives:   false,
			TLSHandshakeTimeout: 500 * time.Millisecond,
		},
	}

	for url := range c.urlQueue {
		if atomic.LoadInt64(&c.total) >= maxURLs {
			return
		}

		title, category, links := c.processURL(client, url)
		if title != "" {
			c.saveURL(url, title, category)
			for _, link := range links {
				if isValidURL(link) {
					c.addURL(link)
				}
			}
		}
	}
}

func (c *UltraCrawler) processURL(client *http.Client, pageURL string) (string, string, []string) {
	fmt.Printf("DEBUG: Processing URL: %s\n", pageURL)
	req, _ := http.NewRequest("GET", pageURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; UltraCrawler/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: %v for %s\n", err, pageURL)
		return "", "", nil
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", "", nil
	}

	title := getTitle(doc)
	category := getCategory(pageURL)
	links := getLinks(doc)

	return title, category, links
}

func (c *UltraCrawler) saveURL(pageURL, title, category string) {
	_, err := c.db.Exec("INSERT IGNORE INTO urls(url, title, category) VALUES(?, ?, ?)", pageURL, title, category)
	if err == nil {
		atomic.AddInt64(&c.total, 1)
	}
}

func (c *UltraCrawler) monitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		total := atomic.LoadInt64(&c.total)
		if total >= maxURLs {
			break
		}
		elapsed := time.Since(c.start).Seconds()
		speed := float64(total) / elapsed
		fmt.Printf("[%d] Speed: %.1f/s\n", total, speed)
	}
}

func getTitle(doc *html.Node) string {
	var title string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = strings.TrimSpace(n.FirstChild.Data)
			title = strings.Replace(title, " - Wikipedia", "", 1)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return title
}

func getCategory(pageURL string) string {
	lower := strings.ToLower(pageURL)
	if strings.Contains(lower, "programming") || strings.Contains(lower, "language") {
		return "Programming Languages"
	}
	if strings.Contains(lower, "algorithm") || strings.Contains(lower, "data_structure") {
		return "Algorithms"
	}
	if strings.Contains(lower, "database") || strings.Contains(lower, "sql") {
		return "Databases"
	}
	if strings.Contains(lower, "web") || strings.Contains(lower, "framework") {
		return "Web Development"
	}
	if strings.Contains(lower, "machine_learning") || strings.Contains(lower, "ai") {
		return "AI/ML"
	}
	return "Programming"
}

func getLinks(doc *html.Node) []string {
	var links []string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.HasPrefix(attr.Val, "/wiki/") {
					if !strings.Contains(attr.Val, ":") && !strings.Contains(attr.Val, "#") {
						links = append(links, "https://en.wikipedia.org"+attr.Val)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return links
}

func isValidURL(link string) bool {
	u, err := url.Parse(link)
	return err == nil && u.Host == "en.wikipedia.org" && strings.HasPrefix(u.Path, "/wiki/") &&
		!strings.Contains(u.Path, ":") && !strings.Contains(link, "#")
}
