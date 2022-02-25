package readfile

import (
	"log"
	"os"
)

func init() {
	// Set storage path and format.
	logPath := "../log/logfile.log"
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("file open error : %v", err)
	}
	log.SetOutput(f)
}
