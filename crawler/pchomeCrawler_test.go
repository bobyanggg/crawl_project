package crawler

import (
	"context"
	"testing"
)

func TestFindMaxPchomePage(t *testing.T) {
	keyword := "iphone13"
	ctx := context.Background()
	page, err := FindMaxPchomePage(ctx, keyword)
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
