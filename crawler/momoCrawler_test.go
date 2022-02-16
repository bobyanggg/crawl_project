package crawler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"

	"dev/crawl_project/sql"
)

func Test_Crawl_Ipad(t *testing.T) {
	ctx := context.Background()
	m := NewMomoQuery("ipad")
	page := 1
	newProducts := make(chan *sql.Product)
	wgJob := &sync.WaitGroup{}
	results := []sql.Product{}
	wgJob.Add(1)
	go func() {
		for product := range newProducts {
			results = append(results, *product)
		}

	}()

	m.Crawl(ctx, page, newProducts, wgJob)
	fmt.Println(results)
	if len(results) == 0 {
		t.Error("error in crawl")
	}
}
func Test_FindMaxMomoPage_Ipad(t *testing.T) {
	ctx := context.Background()
	keyword := "ipad"
	q := NewMomoQuery(keyword)
	maxPage, err := q.FindMaxPage(ctx, math.MaxInt32)
	if err != nil {
		t.Error("error in find momopage")
	}
	if maxPage < 50 {
		t.Error("error in find momopage,page=", maxPage)
	}
}
