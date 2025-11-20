package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// InitQueueManager: Sadece QueueManager nesnesini başlatır.
func InitQueueManager() {
	// DB bağlantısı zaten globalDB'de olmalı.
	if globalDB == nil {
		clog(LevelFatal, "QueueManager başlatılamadı: Global DB bağlantısı mevcut değil.")
		return
	}

	// Queue Manager'ı globalDB bağlantısını kullanarak başlat
	globalQueueManager = &QueueCacheManager{
		DB:    globalDB, // Artık globalDB'yi kullanıyoruz
		Cache: make(map[string]*Queue),
		// Mutex, struct tanımında kalır.
	}
	clog(LevelInfo, "Queue önbellek yöneticisi başlatıldı.")
	loadingIsOkForQueueDefinition = true
}

func InitAriConnection(ctx context.Context) {

	globalClientManager = NewClientManager()

	clog(LevelInfo, "Ari connections is starting...")
	// 3. ARI Bağlantılarını Başlat
	for _, ariCfg := range AppConfig.AriConnections {

		for _, instanceId := range AppConfig.InstanceIDs {

			applicationNameInbound := fmt.Sprintf("%s%s", INBOUND_ARI_APPLICATION_PREFIX, instanceId)
			applicationNameOutbound := fmt.Sprintf("%s%s", OUTBOUND_ARI_APPLICATION_PREFIX, instanceId)
			connectionName1 := fmt.Sprintf("%s-%s-%s", ariCfg.Id, applicationNameInbound, instanceId)
			connectionName2 := fmt.Sprintf("%s-%s-%s", ariCfg.Id, applicationNameOutbound, instanceId)

			ariAppInfoInbound := AriAppInfo{ConnectionName: connectionName1, InboundAppName: applicationNameInbound, OutboundAppName: applicationNameOutbound, IsOutboundApplication: false, InstanceID: instanceId}
			ariAppInfoOutbound := AriAppInfo{ConnectionName: connectionName2, InboundAppName: applicationNameInbound, OutboundAppName: applicationNameOutbound, IsOutboundApplication: true, InstanceID: instanceId}

			go func(ariCfg AriConfig, ariAppInfoInbound AriAppInfo) {
				if err := runApp(ctx, &ariCfg, globalClientManager, ariAppInfoInbound); err != nil {
					clog(LevelError, "ARI application failed to start for %+v: %v", ariAppInfoOutbound, err)
				}
			}(ariCfg, ariAppInfoInbound)

			go func(ariCfg AriConfig, ariAppInfoOutbound AriAppInfo) {
				if err := runApp(ctx, &ariCfg, globalClientManager, ariAppInfoOutbound); err != nil {
					clog(LevelError, "ARI application failed to start for %+v: %v", ariAppInfoOutbound, err)
				}
			}(ariCfg, ariAppInfoOutbound)
		}
	}

}

func InitRedisManager(ctx context.Context) {

	redisClientManager = struct {
		Pubs *redis.ClusterClient
		Subs *redis.ClusterClient
		ctx  *context.Context
	}{}

	redisClientManager.ctx = &ctx

	// Bu redis clinet sadece publish işlemleri için kullanılır
	AppConfig.RLock()
	redisAddresses := AppConfig.RedisAddresses
	redisPassword := AppConfig.RedisPassword
	instanceIds := AppConfig.InstanceIDs
	AppConfig.RUnlock()

	redisClientManager.Subs = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    redisAddresses,
		Password: redisPassword,
	})

	if redisClientManager.Subs == nil {
		clog(LevelFatal, "[REDIS SUBSCRIBE] Redis istemcisi atanmamış (rdb is nil)")
		return
	}

	redisClientManager.Pubs = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    redisAddresses,
		Password: redisPassword,
	})

	if redisClientManager.Pubs == nil {
		clog(LevelFatal, "[REDIS PUBLISHER] Redis istemcisi atanmamış (rdb is nil)")
		return
	}

	handleRedisSubsMessages(ctx, instanceIds)
}

func InitSchedulerManager() {
	globalScheduler = NewScheduler()
	clog(LevelInfo, "Scheduler Manager is started.")
}

// InitDBConnection, SQL Server bağlantısını kurar ve globalDB'yi ayarlar.
func InitDBConnection() error {

	// Windows kimlik doğrulaması (Integrated Security) için bağlantı dizesi

	AppConfig.RLock()
	defer AppConfig.RUnlock()

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

	clog(LevelInfo, "[DB] SQL Server connection established successfully.")
	loadingIsOkForDBManager = true
	return nil
}

func InitCallManager() {
	globalCallManager = NewCallManager()
	clog(LevelInfo, "Çağrı yöneticisi başlatıldı.")
}

func InitHttpServer() {
	if AppConfig.HttpServerEnabled {
		startHttpEnabled()
	} else {
		clog(LevelInfo, "HTTP Server is disabled via config.")
	}
}

func WaitForServicesReady(ctx context.Context) {

	clog(LevelInfo, "Hizmetlerin hazır olması bekleniyor...")

	for {
		// 1. Koşul Kontrolü
		if loadingIsOkForDBManager && loadingIsOkForQueueDefinition {
			clog(LevelInfo, "\n✅ Tüm gereklilikler (HTTP, DB, QueueDef) sağlandı!")
			break // Döngüden çık
		}

		// 2. Durum Raporu (İsteğe bağlı)
		clog(LevelInfo, "Bekleniyor... DB: %t, QueueDef: %t\n",
			loadingIsOkForDBManager, loadingIsOkForQueueDefinition)

		// 3. Duraklama
		// 200 milisaniye bekle
		time.Sleep(200 * time.Millisecond)
	}

}
