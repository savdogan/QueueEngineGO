package main

import (
	"context"
	"fmt"
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

	runtime.SetBlockProfileRate(1)

	version := 1

	log.Printf("QueueEngineGO version:%d is starting...", version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. KONFİGÜRASYONU Yükle
	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}
	AppConfig = cfg

	log.Printf("Successed to load config")

	cfgPublisherHostName := getHostname()
	cfgLogDirectory = AppConfig.LogDirectory
	cfgMinLogLevel = AppConfig.MinLogLevel
	//redisAddresses := AppConfig.RedisAddresses
	//redisPassword := AppConfig.RedisPassword
	//LoadSnapshotOnStart := AppConfig.LoadSnapshotOnStart
	AppConfig.Lock()
	AppConfig.PublisherHostName = cfgPublisherHostName
	AppConfig.Version = version
	log.Printf("PublisherHostName : %s", AppConfig.PublisherHostName)
	AppConfig.Unlock()

	//------------LOG Bölümü Başlangıç
	log.Printf("Async logging is starting, you can now follow it in the log file. ")
	err = startAsyncLogger()
	if err != nil {
		log.Printf("Logging starting is failed.")
		terminateApplication(5)
	}
	log.Printf("Async logging is started")
	//------------LOG Bölümü Başlangıç

	//------------DB Conncetion Bölümü Başlangıç
	if err := InitDBConnection(); err != nil {
		clog(LevelFatal, "Veritabanı bağlantısı kurulamadı: %v", err)
		terminateApplication(5)
		return
	}
	defer CloseDBConnection() // Uygulama sonlandığında bağlantıyı kapat
	//------------DB Conncetion Bölümü Bitiş

	InitSchedulerManager()

	InitQueueManager()

	InitCallManager()

	InitHttpServer()

	InitRedisManager(ctx)

	WaitForServicesReady(ctx)

	fmt.Println("[ARI CONNECTION] is starting...")

	InitAriConnection(ctx)

	go func() {

		// Kullanım Örneği (Örneğin StasisStart geldikten sonra)
		queueName := "Yuktesti" // Varsayımsal kuyruk adı
		queueDef, err := globalQueueManager.GetQueueByName(queueName)

		if err != nil {
			// Kuyruk tanımı bulunamadı veya DB hatası var
			clog(LevelError, "Kuyruk tanımı alınamadı: %v, %d", err, queueDef.ID)
			return
		}

	}()

	for range make(chan struct{}) {
	} // Sonsuza kadar çalış
	time.Sleep(5 * time.Second)
	clog(LevelInfo, "All services shut down. Main exiting.")
}
