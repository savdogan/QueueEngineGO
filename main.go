package main

import (
	"log"
	"os"
	"runtime"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

func terminateApplication(sleepTime time.Duration) {
	log.Printf("Uygulama için exit komutu çağırıldı.....")
	time.Sleep(sleepTime * time.Second)
	os.Exit(1)
}

// Global Log Seviyesi Değişkeni: clog'un erişimi için config'den buraya aktarılacak

func main() {

	version := 1

	runtime.SetBlockProfileRate(1)

	log.Printf("QueueEngineGO version:%d is starting...", version)

	cfg := loadConfig("config.json")

	InitGlobalState(cfg, version)

	startAsyncLogger()

	go startRedisListener()

	go InitHttpServer()

	go InitAriConnection()

	for range make(chan struct{}) {
	} // Sonsuza kadar çalış
	time.Sleep(5 * time.Second)
	clog(LevelInfo, "All services shut down. Main exiting.")
}
