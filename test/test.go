package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type WorkerConfig struct {
	WorkerNum    int `json:"workerNum"`
	TotalProduct int `json:"totalProduct`
}

func main() {
	jsonFile, err := os.Open("../config/worker.json")
	if err != nil {
		log.Fatal("faile to open json fail for creating worker: ", err)
	}
	log.Println("successfully opened worker config")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	var workerConfig WorkerConfig
	if err := json.NewDecoder(jsonFile).Decode(&workerConfig); err != nil {
		log.Fatal(err, "failed to decode worker config")
		return
	}
	fmt.Println(workerConfig)

}
