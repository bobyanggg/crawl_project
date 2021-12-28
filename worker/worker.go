package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"dev/crawl_project/crawler"
	pb "dev/crawl_project/product"

	"dev/crawl_project/sql"

	"github.com/pkg/errors"
)

type WorkerConfig struct {
	MaxProduct int `json:"maxProduct"`
	WorkerNum  int `json:"workerNum"`
	SleepTime  int `json:"sleepTime"`
}
type Job struct {
	web         crawler.Web
	webCrawler  crawler.Crawler
	keyword     string
	page        int
	wgJob       *sync.WaitGroup
	newProducts chan *sql.Product
}

var Webs = []crawler.Web{crawler.Momo, crawler.Pchome}

// Queue creates job Chan and newProduct Chan.
// Workers are started in this function.
// Queue also listen to new product channel and send product information back to server.
// Queue listens to ctx from server, if ctx timeout, Queue calls cleanupCancel to cleanup
func Queue(ctx context.Context, keyWord string, pProduct chan pb.UserResponse, workerConfig WorkerConfig, jobsChan map[crawler.Web]chan *Job) {
	fmt.Println("---------------start-------------")

	newProducts := make(chan *sql.Product, workerConfig.MaxProduct)
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer func() {
		cleanupCancel()
		close(newProducts)

	}()

	// listen to ctx from server, if timeout, call cleanup function
	go func() {
		for {
			select {
			case product := <-newProducts:
				// Insert the data to the database.
				product.Word = keyWord
				if err := sql.Insert(*product); err != nil {
					log.Println(err)
				}
				// Push the data to grpc output.
				pProduct <- pb.UserResponse{
					Name:       product.Name,
					Price:      int32(product.Price),
					ImageURL:   product.ImageURL,
					ProductURL: product.ProductURL,
				}
			case <-ctx.Done():
				if ctx.Err() != context.Canceled {
					cleanupCancel()
				}
				return
			case <-cleanupCtx.Done():
				return
			}
		}
	}()

	wgJob := &sync.WaitGroup{}
	// call send tp send jobs
	anyResponse := false

	wgSend := &sync.WaitGroup{}
	wgSend.Add(len(Webs))
	for _, web := range Webs {
		go func(web crawler.Web) {
			var wc crawler.Crawler
			switch web {
			case crawler.Momo:
				wc = crawler.NewMomoQuery(keyWord)
			case crawler.Pchome:
				wc = crawler.NewPChomeQuery(keyWord)
			}

			err := send(ctx, wc, wgJob, newProducts, jobsChan[web], workerConfig)
			if err == nil {
				anyResponse = true
			} else {
				log.Printf("Failed to get response from %s: %v", web, err)
			}
			wgSend.Done()

		}(web)
	}
	wgSend.Wait() //avoid finish before send is finished
	if !anyResponse {
		cleanupCancel()
		return
	}
	wgJob.Wait()
}

// send function gets the maximum page and puts job into jobchan while looping through pages
func send(ctx context.Context, wc crawler.Crawler, wgJob *sync.WaitGroup, newProducts chan *sql.Product, jobChan chan *Job, workerConfig WorkerConfig) error {

	webNum := len(Webs)
	totalWebProduct := workerConfig.MaxProduct / webNum

	qSrc := wc.GetQuerySrc()

	log.Println("preparing to find max page: ", wc.GetQuerySrc().Web)
	maxPage, err := wc.FindMaxPage(ctx, totalWebProduct)
	if err != nil {
		return errors.Wrapf(err, "failed to get max page")
	}

	for i := 1; i <= maxPage; i++ {
		wgJob.Add(1)
		input := &Job{
			web:         qSrc.Web,
			webCrawler:  wc,
			keyword:     qSrc.Keyword,
			page:        i,
			wgJob:       wgJob,
			newProducts: newProducts,
		}
		fmt.Println("In queue", input)
		jobChan <- input
		log.Println("already send input value:", input)
	}

	return nil
}

// worker starts workers that listen to jobsChan in background
func worker(ctx context.Context, num int, web crawler.Web, jobChan chan *Job, sleepTime int) {

	log.Println("start the worker", num, web)

	for {
		select {
		case job := <-jobChan:
			// n := getRandomTime()
			finishQuery := make(chan bool)
			log.Printf("%d starting on %v, Sleeping %d seconds...\n", num, job, sleepTime)

			job.webCrawler.Crawl(ctx, job.page, finishQuery, job.newProducts, job.wgJob)
			log.Println("finished", job.web, job.page)
			time.Sleep(time.Duration(sleepTime) * time.Second)
			// close workers
		case <-ctx.Done():
			log.Println("closing worker.....", num, web)
			return
		}
	}
}

// startWorker opens worker.json config, generates worker and jobs channel
func StartWorker(ctx context.Context, jobsChan map[crawler.Web]chan *Job, workerConfig WorkerConfig) {
	totalWorker := workerConfig.WorkerNum
	sleepTime := workerConfig.SleepTime

	// generate workers for each web
	for _, web := range Webs {
		for i := 0; i < totalWorker; i++ {
			go worker(ctx, i, web, jobsChan[web], sleepTime)
		}
	}
}
