package main

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
)

var loadingIsOkForDBManager bool
var loadingIsOkForQueueDefinition bool

var cfgMinLogLevel LogLevel
var AppConfig Config

var globalDB *sql.DB                      // Tüm uygulama için tek ve ana bağlantı havuzu
var globalQueueManager *QueueCacheManager // Sadece kuyruk önbelleği ve mantığı için
var globalClientManager *ClientManager    // Ari Clientlaır için
var globalCallManager *CallManager        // Çağrı yönetimi için
var globalScheduler *Scheduler            // İş zamanlayıcı için
var redisClientManager struct {
	Pubs *redis.ClusterClient // Redis abonelikleri için küme istemcisi
	Subs *redis.ClusterClient
	ctx  *context.Context
}
