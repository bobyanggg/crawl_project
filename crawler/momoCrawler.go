package crawler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"dev/crawl_project/sql"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly"
	"github.com/pkg/errors"
)

type MomoQuery struct {
	*Query
}

const absoluteURL string = "https://m.momoshop.com.tw/"

const productsPerPageMomo = 20

func NewMomoQuery(keyword string) *MomoQuery {
	return &MomoQuery{newQuery(Momo, keyword)}
}

func (q *MomoQuery) GetQuerySrc() *Query {
	return q.Query
}

func (q *MomoQuery) Crawl(ctx context.Context, page int, finishQuery chan bool, newProducts chan *sql.Product, wgJob *sync.WaitGroup) {

	qSrc := q.GetQuerySrc()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://m.momoshop.com.tw/search.momo", nil)
	if err != nil {
		log.Println(errors.Wrap(err, "Can not generate request"))
		wgJob.Done()
		return
	}
	query := request.URL.Query()
	query.Add("searchKeyword", qSrc.Keyword)
	query.Set("curPage", fmt.Sprintf("%d", page))
	request.URL.RawQuery = query.Encode()
	startUrl := request.URL.String()

	c := colly.NewCollector(
		colly.AllowedDomains("m.momoshop.com.tw", "www.m.momoshop.com.tw"),
	)

	c.OnHTML("li[class=goodsItemLi]", func(e *colly.HTMLElement) {
		tempProduct := sql.Product{}
		tempProduct.Name = e.ChildText("h3.prdName")
		tempProduct.Word = qSrc.Keyword
		tempPrice, err := strconv.Atoi(e.ChildText("b.price"))
		if err != nil {
			log.Println(errors.Wrapf(err, "failed to get price of %s", tempProduct.Name))
		}
		tempProduct.Price = tempPrice
		tempProduct.ProductURL = absoluteURL + e.ChildAttr("li[class=goodsItemLi] > a", "href")
		tempProduct.ImageURL = e.ChildAttr("img.goodsImg", "src")
		query, err := url.Parse(tempProduct.ProductURL)
		if err != nil {
			log.Println(errors.Wrapf(err, "failed to find Product Url of %s", tempProduct.Name))
		}
		querys := query.Query()
		if tempId, ok := querys["i_code"]; ok {
			tempProduct.ProductID = tempId[0]
		}
		if tempProduct.ProductID == "" {
			log.Println(errors.Wrapf(err, "failed to find Product Url of %s", tempProduct.Name))
		}
		newProducts <- &tempProduct
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting: ", r.URL.String())
	})

	err = c.Visit(startUrl)
	if err != nil {
		log.Println(errors.Wrapf(err, "failed to visit %s", startUrl))
	}
	wgJob.Done()
}

func (q *MomoQuery) FindMaxPage(ctx context.Context, totalWebProduct int) (int, error) {
	calPage := totalWebProduct / productsPerPageMomo
	totalPageResult := 0
	starturl := fmt.Sprintf("https://www.momoshop.com.tw/search/searchShop.jsp?keyword=%s&searchType=1&curPage=%d", q.GetQuerySrc().Keyword, 1)
	selector := "#BodyBase > div.bt_2_layout.searchbox.searchListArea.selectedtop > div.pageArea.topPage > dl > dt > span:nth-child(2)"
	sel := `document.querySelector("body")`

	html, err := GetHttpHtmlContent(ctx, starturl, selector, sel)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get html element")
	}

	dom, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return 0, errors.Wrap(err, "failed to go query")
	}

	dom.Find("#BodyBase > div.bt_2_layout.searchbox.searchListArea.selectedtop > div.pageArea.topPage > dl > dt > span:nth-child(2)").Each(func(i int, selection *goquery.Selection) {
		pageStr := strings.Split(selection.Text(), "/")
		totalPage, _ := strconv.Atoi(pageStr[1])
		totalPageResult = totalPage
	})
	log.Printf("total page of keyword %s in %s is: %d\n", q.Keyword, q.Web, totalPageResult)
	log.Printf("max page allowed: %d", calPage)

	if calPage < totalPageResult {
		totalPageResult = calPage
	}

	return totalPageResult, nil
}

func GetHttpHtmlContent(ctx context.Context, url string, selector string, sel interface{}) (string, error) {
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true), // debug using
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`),
	}
	//Initialization parameters, first pass an empty data
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...)

	c, _ := chromedp.NewExecAllocator(ctx, options...)

	// create context
	chromeCtx, _ := chromedp.NewContext(c, chromedp.WithLogf(log.Printf))
	//Execute an empty task to create a chrome instance in advance
	if err := chromedp.Run(chromeCtx, make([]chromedp.Action, 0, 1)...); err != nil {
		return "", errors.Wrap(err, "failed to create a chrome instance")
	}

	//Create a context with a timeout of 40s
	timeoutCtx, cancel := context.WithTimeout(chromeCtx, 40*time.Second)
	defer cancel()

	var htmlContent string
	err := chromedp.Run(timeoutCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(selector),
		chromedp.OuterHTML(sel, &htmlContent, chromedp.ByJSPath),
	)
	if err != nil {
		fmt.Printf("Run err : %v\n", err)
		return "", err
	}

	return htmlContent, nil
}
