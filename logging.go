package main

import (
	"io"
	"log"
	"os"
	"sync"
	"time"
)

func initLogger() {
	file, err := os.OpenFile("bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("open log file: %v", err)
	}

	mw := io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func logInfo(format string, args ...interface{}) {
	log.Printf("INFO "+format, args...)
}

func logWarn(format string, args ...interface{}) {
	log.Printf("WARN "+format, args...)
}

func logError(format string, args ...interface{}) {
	log.Printf("ERROR "+format, args...)
}

var (
	alertLast = map[string]time.Time{}
	alertMu   sync.Mutex
)

func canAlert(k string, d time.Duration) bool {
	alertMu.Lock()
	defer alertMu.Unlock()

	if t, ok := alertLast[k]; ok && time.Since(t) < d {
		return false
	}
	alertLast[k] = time.Now()
	logWarn("alert allowed: key=%s cooldown=%s", k, d)
	return true
}
