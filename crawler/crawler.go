package crawler

import (
	"context"
	"dev/crawl_project/sql"
	"sync"
)

type Crawler interface {
	// Find product information from the website
	Crawl(ctx context.Context, page int, finishQuery chan bool, newProducts chan *sql.Product, wgJob *sync.WaitGroup)
	FindMaxPage(ctx context.Context, totalWebProduct int) (int, error)
	GetQuerySrc() *Query
}

type Query struct {
	Web     Web
	Keyword string
}

type Web string

const (
	Pchome Web = "pchome"
	Momo       = "momo"
)

func newQuery(web Web, keyWord string) *Query {
	return &Query{Web: web, Keyword: keyWord}
}
