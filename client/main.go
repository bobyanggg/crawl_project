// $ go run main.go iphone
package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sort"

	pb "github.com/bobyanggg/crawl_project/product"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"

	"google.golang.org/grpc"
)

var tpl *template.Template

const session = "session"

type sortMethod string

const (
	lessFirst   sortMethod = "lessFirst"
	higherFirst sortMethod = "higherFirst"
)

type searchResult struct {
	Title  string
	Result []*pb.UserResponse
}

var dbResult = map[string]*searchResult{}

func init() {
	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {
	mux := httprouter.New()
	mux.GET("/", index)
	mux.POST("/search", search)
	mux.POST("/sorter", sorter)
	http.ListenAndServe(":8080", mux)

}

func index(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// create session
	sID := uuid.NewV4()
	sess := &http.Cookie{
		Name:  "session",
		Value: sID.String(),
	}
	http.SetCookie(w, sess)

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
		Title:  "Search result: " + keyWord,
		Result: result,
	}

	err = tpl.ExecuteTemplate(w, "result.gohtml", data)
	HandleError(w, err)

	sess, err := req.Cookie(session)
	if err != nil {
		log.Fatalln(err)
	}

	dbResult[sess.Value] = data
}

func sorter(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	sorter := req.FormValue("sorter")
	fmt.Println(sorter)

	sess, err := req.Cookie(session)
	if err != nil {
		log.Fatalln(err)
	}

	data := dbResult[sess.Value]

	switch sortMethod(sorter) {
	case lessFirst:
		sort.Slice(data.Result, func(i, j int) bool { return data.Result[i].Price < data.Result[j].Price })
	case higherFirst:
		sort.Slice(data.Result, func(i, j int) bool { return data.Result[i].Price > data.Result[j].Price })
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
