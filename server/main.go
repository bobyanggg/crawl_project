package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"dev/crawl_project/model"
	pb "dev/crawl_project/product"
	"dev/crawl_project/sql"
	"dev/crawl_project/worker"

	"google.golang.org/grpc"
)

// Log setting initialize from /grpc_service/model
// SQL setting initialize from /grpc_service/sql

type Server struct {
}

type ProductGRPC struct {
	Products      chan pb.UserResponse
	FinishRequest chan int
}

func (s *Server) GetUserInfo(in *pb.UserRequest, stream pb.UserService_GetUserInfoServer) error {
	log.Println("Search for", in.KeyWord)

	// Search in the database.
	products, err := sql.Select(in.KeyWord)
	if err != nil {
		return err
	}

	var p ProductGRPC
	p.Products = make(chan pb.UserResponse, 1000)
	p.FinishRequest = make(chan int, 1)

	ctx, _ := context.WithTimeout(stream.Context(), 100*time.Second)

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
			worker.Queue(ctx, in.KeyWord, p.Products)
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
			return nil
		}
	}
}

func main() {
	log.Println("---------- Service started ---------")

	// Read the grpc config.
	grpcConfig, err := model.OpenJson("../config/grpc.json")
	if err != nil {
		log.Fatal(err)
	}

	// Start the GRPC service.
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &Server{})
	listen, err := net.Listen("tcp", fmt.Sprintf(":%v", grpcConfig["port"]))
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Serve(listen)
}
