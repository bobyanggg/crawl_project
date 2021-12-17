package crawler

import (
	"context"
	"math"
	"testing"
)

func TestFindMaxPchomePage(t *testing.T) {
	ctx := context.Background()
	keyword := "iphone13"
	q := NewPChomeQuery(keyword)
	page, err := q.FindMaxPage(ctx, math.MaxInt32)
	if err != nil {
		t.Log("err: ", err)
	}
	t.Log("page: ", page)
	if page != 100 {
		t.Error("fail")
	} else {
		t.Log("success")
	}
}
