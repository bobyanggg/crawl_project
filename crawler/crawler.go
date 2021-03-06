package crawler

import (
	"context"
	"sync"

	"github.com/bobyanggg/crawl_project/sql"
)

type Crawler interface {
	// Find product information from the website
	Crawl(ctx context.Context, page int, newProducts chan *sql.Product, wgJob *sync.WaitGroup)
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
	Momo   Web = "momo"
)

func newQuery(web Web, keyWord string) *Query {
	return &Query{Web: web, Keyword: keyWord}
}
