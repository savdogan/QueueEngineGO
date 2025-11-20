package main

import (
	"database/sql"
	"sync"

	"github.com/CyCoreSystems/ari/v6"
)

// ClientManager, tüm aktif ARI istemcilerini yönetir.
type ClientManager struct {
	sync.RWMutex
	clients map[string]ari.Client
}

// DBManager, tüm uygulama için tek bir SQL Server bağlantısını yönetir.
type DBManager struct {
	DB *sql.DB
}

// QueueCacheManager, DB bağlantısını ve Queue önbelleğini yönetir.
type QueueCacheManager struct {
	sync.RWMutex // Cache erişimi için kilit
	DB           *sql.DB
	Cache        map[string]*Queue // Key: queue_name, Value: *WbpQueue
}

type CallManager struct {
	mu          sync.RWMutex
	calls       map[string]*Call
	outChannels map[string]*ari.ChannelHandle
}
