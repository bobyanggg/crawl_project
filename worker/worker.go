package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	pb "dev/crawl_project/product"

	"dev/crawl_project/sql"
)

type WorkerConfig struct {
	MaxProduct int `json:"maxProduct"`
	WorkerNum  int `json:"workerNum"`
	SleepTime  int `json:"sleepTime"`
}
type Job struct {
	web         string
	keyword     string
	page        int
	wgJob       *sync.WaitGroup
	newProducts chan *sql.Product
}

type Crawler interface {
	// Find product information from the website
	Crawl(page int, finishQuery chan bool, newProducts chan *sql.Product, wgJob *sync.WaitGroup)
}

var webs = []string{"momo", "pchome"}

// Queue creates job Chan and newProduct Chan.
// Workers are started in this function.
// Queue also listen to new product channel and send product information back to server.
// Queue listens to ctx from server, if ctx timeout, Queue calls cleanupCancel to cleanup
func Queue(ctx context.Context, keyWord string, pProduct chan pb.UserResponse) {
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	// cleanupCtx, cleanupCancel = context.WithDeadline(cleanupCtx, cleanupCtx.Deadline())
	defer cleanupCancel()
	// load worker config
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

	jobsChan := make(map[string]chan *Job)
	newProducts := make(chan *sql.Product, workerConfig.MaxProduct)

	// generate job channel for each web
	for _, val := range webs {
		jobsChan[val] = make(chan *Job, workerConfig.WorkerNum)
	}

	// responsible for start worker
	go startWorker(cleanupCtx, jobsChan, workerConfig)

	// listen to newProducts channel
	go func() {
		for product := range newProducts {
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
		}
	}()

	// listen to ctx from server, if timeout, call cleanup function
	go func() {
		for {
			select {
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
	for _, web := range webs {
		send(ctx, web, keyWord, wgJob, newProducts, jobsChan, workerConfig)
	}
	wgJob.Wait()
	close(newProducts)
}

// send function gets the maximum page and puts job into jobchan while looping through pages
func send(ctx context.Context, web, keyWord string, wgJob *sync.WaitGroup, newProducts chan *sql.Product, jobsChan map[string]chan *Job, workerConfig WorkerConfig) {
	var maxPage int
	webNum := len(webs)
	totalWebProduct := workerConfig.MaxProduct / webNum

	// TODO : make a interface or merge existing?
	switch web {
	case "momo":
		calPage := totalWebProduct/20 + 1
		maxMomo := FindMaxMomoPage(keyWord)
		if calPage > maxMomo {
			maxPage = maxMomo
		} else {
			maxPage = calPage
		}
	case "pchome":
		calPage := totalWebProduct/20 + 1
		maxPchome := FindMaxPchomePage(keyWord)
		if calPage > maxPchome {
			maxPage = maxPchome
		} else {
			maxPage = calPage
		}
	}

	go func(maxPage int) {
		for i := 1; i <= maxPage; i++ {
			wgJob.Add(1)
			input := &Job{web, keyWord, i, wgJob, newProducts}
			fmt.Println("In queue", input)
			jobsChan[web] <- input
			log.Println("already send input value:", input)
		}
	}(maxPage)
}

// process creates query instance, then calls crawl function
func process(num int, job Job, newProducts chan *sql.Product, sleepTime int) {

	// n := getRandomTime()
	var crawler Crawler
	finishQuery := make(chan bool)
	log.Printf("%d starting on %v, Sleeping %d seconds...\n", num, job, sleepTime)

	switch job.web {
	case "momo":
		crawler = NewMomoQuery(job.keyword)
	case "pchome":
		crawler = NewPChomeQuery(job.keyword)
	}
	go crawler.Crawl(job.page, finishQuery, newProducts, job.wgJob)
	log.Println("finished", job.web, job.page)
	time.Sleep(time.Duration(sleepTime) * time.Second)
}

// worker starts workers that listen to jobsChan in background
func worker(ctx context.Context, num int, web string, jobsChan map[string]chan *Job, sleepTime int) {

	log.Println("start the worker", num, web)

	for {
		select {
		case job := <-jobsChan[web]:
			process(num, *job, job.newProducts, sleepTime)
			// close workers
		case <-ctx.Done():
			if ctx.Err() != context.Canceled {
				log.Println("context err: ", ctx.Err())
			}
			log.Println("closing worker.....", num, web)
			return
		}
	}
}

// startWorker opens worker.json config, generates worker and jobs channel
func startWorker(ctx context.Context, jobsChan map[string]chan *Job, workerConfig WorkerConfig) {
	fmt.Println("--------------start-------------")
	totalWorker := workerConfig.WorkerNum
	sleepTime := workerConfig.SleepTime

	// generate workers for each web
	go func() {
		for _, web := range webs {
			for i := 0; i < totalWorker; i++ {
				go worker(ctx, i, web, jobsChan, sleepTime)
			}
		}
	}()
}
