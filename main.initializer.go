package main

import (
	"database/sql"
)

func NewQueueCacheManager(DB *sql.DB) *QueueCacheManager {
	return &QueueCacheManager{
		DB:    DB, // Artık g.DB'yi kullanıyoruz
		Cache: make(map[string]*Queue),
		// Mutex, struct tanımında kalır.
	}
}

func InitHttpServer() {
	if g.Cfg.HttpServerEnabled {
		startHttpEnabled()
	} else {
		clog(LevelInfo, "HTTP Server is disabled via config.")
	}
}
