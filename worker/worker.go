package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	keyword     string
	page        int
	wgJob       *sync.WaitGroup
	newProducts chan *sql.Product
}

var webs = []crawler.Web{crawler.Momo, crawler.Pchome}

// Queue creates job Chan and newProduct Chan.
// Workers are started in this function.
// Queue also listen to new product channel and send product information back to server.
// Queue listens to ctx from server, if ctx timeout, Queue calls cleanupCancel to cleanup
func Queue(ctx context.Context, keyWord string, pProduct chan pb.UserResponse) {
	fmt.Println("---------------start-------------")

	jsonFile, err := os.Open("../config/worker.json")
	if err != nil {
		log.Fatal("faile to open json fail for creating worker: ", err)
	}
	log.Println("successfully opened worker config")

	// defer closes jsonFile after parsing, if not closed, future parsing will fail
	defer jsonFile.Close()

	var workerConfig WorkerConfig
	if err := json.NewDecoder(jsonFile).Decode(&workerConfig); err != nil {
		log.Fatal(err, "failed to decode worker config")
		return
	}

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
	wgSend.Add(len(webs))
	for _, web := range webs {
		go func(web crawler.Web) {
			var wc crawler.Crawler
			switch web {
			case crawler.Momo:
				wc = crawler.NewMomoQuery(keyWord)
			case crawler.Pchome:
				wc = crawler.NewPChomeQuery(keyWord)
			}

			jobsChan := make(chan *Job, workerConfig.WorkerNum)
			// responsible for start worker
			go startWorker(cleanupCtx, wc, jobsChan, workerConfig)

			err := send(ctx, wc, wgJob, newProducts, jobsChan, workerConfig)
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
func send(ctx context.Context, wc crawler.Crawler, wgJob *sync.WaitGroup, newProducts chan *sql.Product, jobsChan chan *Job, workerConfig WorkerConfig) error {

	webNum := len(webs)
	totalWebProduct := workerConfig.MaxProduct / webNum

	qSrc := wc.GetQuerySrc()

	log.Println("peparing to find max page: ", wc.GetQuerySrc().Web)
	maxPage, err := wc.FindMaxPage(ctx, totalWebProduct)
	if err != nil {
		return errors.Wrapf(err, "failed to get max page")
	}

	for i := 1; i <= maxPage; i++ {
		wgJob.Add(1)
		input := &Job{qSrc.Web, qSrc.Keyword, i, wgJob, newProducts}
		fmt.Println("In queue", input)
		jobsChan <- input
		log.Println("already send input value:", input)
	}

	return nil
}

// worker starts workers that listen to jobsChan in background
func worker(ctx context.Context, wc crawler.Crawler, num int, jobsChan chan *Job, sleepTime int) {

	log.Println("start the worker", num, wc.GetQuerySrc().Web)

	for {
		select {
		case job := <-jobsChan:
			// n := getRandomTime()
			finishQuery := make(chan bool)
			log.Printf("%d starting on %v, Sleeping %d seconds...\n", num, job, sleepTime)

			go wc.Crawl(ctx, job.page, finishQuery, job.newProducts, job.wgJob)
			log.Println("finished", job.web, job.page)
			time.Sleep(time.Duration(sleepTime) * time.Second)
			// close workers
		case <-ctx.Done():
			if ctx.Err() != context.Canceled {
				log.Println("context err: ", ctx.Err())
			}
			log.Println("closing worker.....", num, wc.GetQuerySrc().Web)
			return
		}
	}
}

// startWorker opens worker.json config, generates worker and jobs channel
func startWorker(ctx context.Context, wc crawler.Crawler, jobsChan chan *Job, workerConfig WorkerConfig) {
	totalWorker := workerConfig.WorkerNum
	sleepTime := workerConfig.SleepTime

	// generate workers for each web
	for i := 0; i < totalWorker; i++ {
		go worker(ctx, wc, i, jobsChan, sleepTime)
	}

}
