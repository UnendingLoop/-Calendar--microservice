package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

func LogCollector(wg *sync.WaitGroup, logMode string, ch <-chan *EventEntry) {
	out := &os.File{}
	var err error

	logMode = strings.ToLower(strings.TrimSpace(logMode))

	switch logMode {
	case "":
		log.Println("logMode is not specified. Using stdout as default.")
		out = os.Stdout
	case "stdout":
		out = os.Stdout
	default:
		filename := time.Now().Format("2006-01-02") + logMode
		out, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Println("failed to open/create log-file. Using stdout as default.")
			out = os.Stdout
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for logEntry := range ch {
			fmt.Fprintln(out, logEntry.TimeStamp, logEntry.Level, logEntry.Msg, logEntry.Err, logEntry.Fields)
		}
		log.Println("Log channel closed. Exiting LogCollector")
	}()
}
