# UltraCrawler

UltraCrawler is a fast, multi-threaded web crawler written in Go. It is mainly designed to crawl Wikipedia pages and store basic data such as URL, page title, and category in a MySQL database.

This project is intended to be simple, readable, and practical. It can be used for learning, experiments, or as a base for your own crawler.

---

## Features

- concurrent crawling using goroutines
- worker pool architecture
- URL deduplication
- HTML parsing (title and links)
- simple page categorization
- MySQL storage
- crawl speed monitoring
- HTTP timeouts

---

## Requirements

- Go 1.20 or newer
- MySQL or MariaDB
- Internet connection

---

## Installation

Clone the repository:

```bash
git clone https://github.com/your-username/ultracrawler.git
cd ultracrawler
```

Install dependencies:

```bash
go mod tidy
```

---

## Configuration

### Environment variables

Create a `.env` file in the project root directory.
You can start by copying the example file:

```bash
cp .env.example .env
```

Example `.env` file:

```
DB_HOST=localhost
DB_PORT=3306
DB_USER=your_database_user
DB_PASSWORD=your_database_password
DB_NAME=crawler
```

Make sure the database exists:

```sql
CREATE DATABASE crawler CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

---

## Database structure

The application creates the table automatically:

```sql
CREATE TABLE urls (
  id INT AUTO_INCREMENT PRIMARY KEY,
  url VARCHAR(767) UNIQUE,
  title VARCHAR(500),
  category VARCHAR(200)
);
```

---

## How it works

1. The crawler starts with a list of start URLs
2. Workers fetch pages in parallel
3. Each page is parsed to extract:
   - title
   - internal Wikipedia links
4. Data is saved to the database
5. New links are added to the queue
6. Duplicate URLs are ignored

The process continues until the limit or timeout is reached.

---

## Code configuration

Main settings are defined in constants:

```go
const (
  maxURLs    = 1000000000000000000
  numWorkers = 100
  timeout    = 5 * time.Second
)
```

- `maxURLs` – maximum number of stored URLs
- `numWorkers` – number of concurrent workers
- `timeout` – HTTP request timeout

Start URLs example:

```go
startURLs := []string{
  "https://en.wikipedia.org/wiki/Special:WhatLinksHere?target=Programing",
}
```

---

## Running the crawler

Start the application:

```bash
go run main.go
```

Example output:

```
[1200] Speed: 80.3/s
DEBUG: Processing URL: https://en.wikipedia.org/wiki/Go_(programming_language)
```

---

## Performance notes

- uses MySQL connection pooling
- reuses HTTP connections
- limits request time

Recommendations:
- do not set `numWorkers` too high
- be careful when crawling large sites

---

## Legal note

This crawler is intended for educational and testing purposes.

When crawling Wikipedia or other websites:
- follow their terms of use
- do not overload servers

The author is not responsible for misuse of this software.

---

[](url)
