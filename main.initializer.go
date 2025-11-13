package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
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

func InitAriConnection(wg *sync.WaitGroup, ctx context.Context) {

	globalClientManager = NewClientManager()

	CustomLog(LevelInfo, "Ari connections is starting...")
	// 3. ARI Bağlantılarını Başlat
	for _, ariCfg := range AppConfig.AriConnections {
		wg.Add(1)

		go func(ariCfg AriConfig) {
			defer wg.Done()
			if err := runApp(ctx, ariCfg, globalClientManager); err != nil {
				CustomLog(LevelError, "ARI application failed to start for %s: %v", ariCfg.Application, err)
			}
		}(ariCfg)
	}

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
		return fmt.Errorf("SQL Server'a bağlanırken hata: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("SQL Server bağlantı testi başarısız: %w", err)
	}

	// Temel bağlantıyı global alana atama
	globalDB = db

	CustomLog(LevelInfo, "SQL Server bağlantısı başarılı.")
	loadingIsOkForDBManager = true
	return nil
}
