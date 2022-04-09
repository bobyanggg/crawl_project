// $ go run main.go iphone
package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http"
	"sort"

	pb "github.com/bobyanggg/crawl_project/product"
	"github.com/julienschmidt/httprouter"

	"google.golang.org/grpc"
)

var tpl *template.Template

type searchResult struct {
	Title  string
	Result []*pb.UserResponse
}

func init() {
	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
	mux := httprouter.New()
	mux.GET("/", index)
	mux.POST("/search", search)
	http.ListenAndServe(":8080", mux)

}

func index(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	err := tpl.ExecuteTemplate(w, "index.gohtml", nil)
	HandleError(w, err)
}

func search(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	keyWord := req.FormValue("keyWord")

	// connect to GRPC service
	conn, err := grpc.Dial(":8081", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	getProductClient, err := client.GetProduct(context.Background(), &pb.UserRequest{KeyWord: keyWord})
	if err != nil {
		log.Fatal(err)
	}

	var result []*pb.UserResponse

	// receive
	for {
		reply, err := getProductClient.Recv()
		if err == io.EOF {
			log.Println("Done")
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("reply : %v\n", reply)
		result = append(result, reply)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Price < result[j].Price })

	data := &searchResult{
		Title:  keyWord,
		Result: result,
	}

	err = tpl.ExecuteTemplate(w, "result.gohtml", data)
	HandleError(w, err)
}

func HandleError(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatalln(err)
	}
}
