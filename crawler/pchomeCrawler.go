package crawler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"dev/crawl_project/sql"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type PChomeQuery struct {
	*Query
}

type PchomeResponse struct {
	Prods []Commodity `json:"prods"`
}

type Commodity struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
	PicS  string `json:"picS"`
	Id    string `json:"Id"`
}

type PchomeMaxPageResponse struct {
	MaxPage int `json:"totalPage"`
}

const productsPerPagePchome = 20

func NewPChomeQuery(keyword string) *PChomeQuery {
	return &PChomeQuery{newQuery(Pchome, keyword)}
}

func (q *PChomeQuery) GetQuerySrc() *Query {
	return q.Query
}

func (q *PChomeQuery) FindMaxPage(ctx context.Context, totalWebProduct int) (int, error) {
	calPage := totalWebProduct / productsPerPagePchome
	var client = &http.Client{Timeout: 10 * time.Second}

	request, err := http.NewRequest("GET", "http://ecshweb.pchome.com.tw/search/v3.3/all/results?sort=rnk", nil)
	if err != nil {
		fmt.Println("Can not generate request")
		fmt.Println(err)
	}

	query := request.URL.Query()
	query.Add("q", q.GetQuerySrc().Keyword)

	var maxPage PchomeMaxPageResponse
	request.URL.RawQuery = query.Encode()
	url := request.URL.String()

	response, err := client.Get(url)
	if err != nil {
		errors.Wrapf(err, "failed to get response from %s", url)
	}

	if err := json.NewDecoder(response.Body).Decode(&maxPage); err != nil {
		errors.Wrap(err, "failed to decode json")
	}

	defer response.Body.Close()

	log.Printf("total page of keyword %s in %s is: %d\n", q.Keyword, q.Web, maxPage.MaxPage)
	log.Printf("max page allowed: %d", calPage)

	if calPage < maxPage.MaxPage {
		maxPage.MaxPage = calPage
	}

	return maxPage.MaxPage, nil
}

func (q *PChomeQuery) Crawl(ctx context.Context, page int, finishQuery chan bool, newProducts chan *sql.Product, wgJob *sync.WaitGroup) {
	qSrc := q.GetQuerySrc()

	var client = &http.Client{Timeout: 10 * time.Second}

	request, err := http.NewRequestWithContext(ctx, "GET", "http://ecshweb.pchome.com.tw/search/v3.3/all/results?sort=rnk", nil)
	if err != nil {
		fmt.Println(errors.Wrap(err, "Can not generate request"))
	}

	query := request.URL.Query()
	query.Add("q", qSrc.Keyword)

	var result PchomeResponse
	query.Set("page", fmt.Sprintf("%d", page))
	request.URL.RawQuery = query.Encode()
	url := request.URL.String()

	response, err := client.Get(url)
	if err != nil {
		log.Println(errors.Wrap(err, "can not get response form PChome"))
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		log.Println(errors.Wrapf(err, "can not decode JSON form PChome for %s", qSrc.Keyword))
	}

	defer response.Body.Close()

	for _, prod := range result.Prods {
		tempProduct := sql.Product{
			Word:       qSrc.Keyword,
			ProductID:  prod.Id,
			Name:       prod.Name,
			Price:      prod.Price,
			ImageURL:   "https://b.ecimg.tw" + prod.PicS,
			ProductURL: "https://24h.pchome.com.tw/prod/" + prod.Id,
		}
		newProducts <- &tempProduct
	}
	wgJob.Done()
}
