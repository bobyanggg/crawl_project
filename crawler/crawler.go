package crawler

import (
	"context"
	"dev/crawl_project/sql"
	"sync"
)

type Crawler interface {
	// Find product information from the website
	Crawl(ctx context.Context, page int, finishQuery chan bool, newProducts chan *sql.Product, wgJob *sync.WaitGroup) error
}
