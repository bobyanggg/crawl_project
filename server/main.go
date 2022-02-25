package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/bobyanggg/crawl_project/crawler"
	pb "github.com/bobyanggg/crawl_project/product"
	"github.com/bobyanggg/crawl_project/readfile"
	"github.com/bobyanggg/crawl_project/sql"
	"github.com/bobyanggg/crawl_project/worker"

	"google.golang.org/grpc"
)

// Log setting initialize from /grpc_service/model
// SQL setting initialize from /grpc_service/sql

type Server struct {
	jobsChan     map[crawler.Web]chan *worker.Job
	workerConfig worker.WorkerConfig
}

type ProductGRPC struct {
	Products      chan pb.UserResponse
	FinishRequest chan int
}

func (s *Server) GetProduct(in *pb.UserRequest, stream pb.UserService_GetProductServer) error {
	log.Println("Search for", in.KeyWord)

	// Search in the database.
	products, err := sql.Select(in.KeyWord)
	if err != nil {
		return err
	}

	var p ProductGRPC
	p.Products = make(chan pb.UserResponse, 1000)
	p.FinishRequest = make(chan int, 1)

	ctx := context.Background()

	// Output it directly, if there are data in the database, otherwise search for data on the internet.
	go func() {
		if len(products) > 0 {
			// Push the data to grpc output buffer from the database.
			for _, product := range products {
				p.Products <- pb.UserResponse{
					Name:       product.Name,
					Price:      int32(product.Price),
					ImageURL:   product.ImageURL,
					ProductURL: product.ProductURL,
				}
			}
		} else {

			// Search for keyword in webs, then push the data to grpc output buffer.
			worker.Queue(ctx, in.KeyWord, p.Products, s.workerConfig, s.jobsChan)
		}
		// Check all products have been send, then finish this grpc request.
		for {
			select {
			case <-stream.Context().Done():
				log.Println("..........ctx canceled...........", stream.Context().Err())
				return
			default:
				if len(p.Products) == 0 {
					p.FinishRequest <- 1
					return
				}
			}
		}
	}()

	// Output the data to the client.
	for {
		select {
		case product := <-p.Products:
			err := stream.Send(&product)
			if err != nil {
				log.Println("client closed")
			}
		case <-p.FinishRequest:
			log.Println("Done!")
			return nil
		case <-ctx.Done():
			log.Println("Time out")
			return errors.New("time out")
		}
	}
}

func main() {
	log.Println("---------- Service started ---------")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// open workerConfig
	var workerConfig worker.WorkerConfig
	if err := readfile.OpenJsonEncodeStruct("../config/worker.json", &workerConfig); err != nil {
		log.Fatal("failed to get data from worker.json", err)
	}

	// Start the workers
	jobsChan := make(map[crawler.Web]chan *worker.Job)
	// Make jobsChan
	for _, web := range worker.Webs {
		jobsChan[web] = make(chan *worker.Job, 10)
	}
	worker.StartWorker(ctx, jobsChan, workerConfig)

	// Read the grpc config.
	grpcConfig, err := readfile.OpenJson("../config/grpc.json")
	if err != nil {
		log.Fatal(err)
	}

	// Start the GRPC service.
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &Server{jobsChan: jobsChan, workerConfig: workerConfig})
	listen, err := net.Listen("tcp", fmt.Sprintf(":%v", grpcConfig["port"]))
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Serve(listen)
}
