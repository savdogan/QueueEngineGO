package main

import "database/sql"

var globalDB *sql.DB                      // Tüm uygulama için tek ve ana bağlantı havuzu
var globalQueueManager *QueueCacheManager // Sadece kuyruk önbelleği ve mantığı için
