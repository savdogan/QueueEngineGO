package main

import "database/sql"

var loadingIsOkForDBManager bool
var loadingIsOkForQueueDefinition bool

var cfgMinLogLevel LogLevel
var AppConfig Config

var globalDB *sql.DB                      // Tüm uygulama için tek ve ana bağlantı havuzu
var globalQueueManager *QueueCacheManager // Sadece kuyruk önbelleği ve mantığı için
var globalClientManager *ClientManager    // Ari Clientlaır için
var globalCallManager *CallManager        // Çağrı yönetimi için
var globalScheduler *Scheduler            // İş zamanlayıcı için
