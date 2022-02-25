package readfile

import (
	"encoding/json"
	"log"
	"os"
)

func OpenJson(filePath string) (map[string]interface{}, error) {
	// Open the file.
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	// Parse
	var data map[string]interface{}
	if err = json.NewDecoder(jsonFile).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func OpenJsonEncodeStruct(filePath string, data interface{}) error {
	jsonFile, err := os.Open("../config/worker.json")
	if err != nil {
		log.Fatal("failed to open json fail for creating worker: ", err)
	}
	log.Println("successfully opened worker config")

	// defer closes jsonFile after parsing, if not closed, future parsing will fail
	defer jsonFile.Close()

	if err := json.NewDecoder(jsonFile).Decode(&data); err != nil {
		return err
	}
	return nil
}
