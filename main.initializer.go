package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// InitQueueManager: Sadece QueueManager nesnesini başlatır.
func InitQueueManager() {
	// DB bağlantısı zaten globalDB'de olmalı.
	if globalDB == nil {
		CustomLog(LevelFatal, "QueueManager başlatılamadı: Global DB bağlantısı mevcut değil.")
		return
	}

	// Queue Manager'ı globalDB bağlantısını kullanarak başlat
	globalQueueManager = &QueueCacheManager{
		DB:    globalDB, // Artık globalDB'yi kullanıyoruz
		Cache: make(map[string]*Queue),
		// Mutex, struct tanımında kalır.
	}
	CustomLog(LevelInfo, "Queue önbellek yöneticisi başlatıldı.")
	loadingIsOkForQueueDefinition = true
}

func InitAriConnection(ctx context.Context) {

	globalClientManager = NewClientManager()

	CustomLog(LevelInfo, "Ari connections is starting...")
	// 3. ARI Bağlantılarını Başlat
	for _, ariCfg := range AppConfig.AriConnections {
		go func(ariCfg AriConfig) {
			if err := runApp(ctx, &ariCfg, globalClientManager); err != nil {
				CustomLog(LevelError, "ARI application failed to start for %s: %v", ariCfg.Application, err)
			}
		}(ariCfg)
	}

}

func InitSchedulerManager() {
	globalScheduler = NewScheduler()
	CustomLog(LevelInfo, "Scheduler Manager is started.")
}

// InitDBConnection, SQL Server bağlantısını kurar ve globalDB'yi ayarlar.
func InitDBConnection() error {

	// Windows kimlik doğrulaması (Integrated Security) için bağlantı dizesi

	AppConfig.Mu.RLock()
	defer AppConfig.Mu.RUnlock()

	if AppConfig.DBConnectingString == "" {
		return fmt.Errorf("DBConnectingString")
	}

	connString := AppConfig.DBConnectingString

	// !!! SÜRÜCÜ ADI "mssql" OLARAK DEĞİŞTİ !!!
	db, err := sql.Open("mssql", connString)
	if err != nil {
		return fmt.Errorf("[DB] failed to connect to SQL Server: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("[DB] SQL Server connection test is failed : %w", err)
	}

	// Temel bağlantıyı global alana atama
	globalDB = db

	CustomLog(LevelInfo, "[DB] SQL Server connection established successfully.")
	loadingIsOkForDBManager = true
	return nil
}

func InitCallManager() {
	globalCallManager = NewCallManager()
	CustomLog(LevelInfo, "Çağrı yöneticisi başlatıldı.")
}

func InitHttpServer() {
	if AppConfig.HttpServerEnabled {
		startHttpEnabled()
	} else {
		CustomLog(LevelInfo, "HTTP Server is disabled via config.")
	}
}

func WaitForServicesReady(ctx context.Context) {

	CustomLog(LevelInfo, "Hizmetlerin hazır olması bekleniyor...")

	for {
		// 1. Koşul Kontrolü
		if loadingIsOkForDBManager && loadingIsOkForQueueDefinition {
			CustomLog(LevelInfo, "\n✅ Tüm gereklilikler (HTTP, DB, QueueDef) sağlandı!")
			break // Döngüden çık
		}

		// 2. Durum Raporu (İsteğe bağlı)
		CustomLog(LevelInfo, "Bekleniyor... DB: %t, QueueDef: %t\n",
			loadingIsOkForDBManager, loadingIsOkForQueueDefinition)

		// 3. Duraklama
		// 200 milisaniye bekle
		time.Sleep(200 * time.Millisecond)
	}

}
