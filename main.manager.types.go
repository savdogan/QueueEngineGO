package main

import (
	"database/sql"
	"sync"

	"github.com/CyCoreSystems/ari/v6"
)

// ClientManager, tüm aktif ARI istemcilerini yönetir.
type ClientManager struct {
	clients map[string]ari.Client
	mu      sync.RWMutex
}

// DBManager, tüm uygulama için tek bir SQL Server bağlantısını yönetir.
type DBManager struct {
	DB *sql.DB
}

// QueueCacheManager, DB bağlantısını ve Queue önbelleğini yönetir.
type QueueCacheManager struct {
	DB    *sql.DB
	Cache map[string]*Queue // Key: queue_name, Value: *WbpQueue
	mu    sync.RWMutex      // Cache erişimi için kilit
}

type CallManager struct {
	// Calls haritası: Anahtar (string) UniqueId'dir, Değer ise *model.Call işaretçisidir.
	// İşaretçi kullanıyoruz ki, haritadan çektiğimizde orijinal nesneyi güncelleyebilelim.
	calls       map[string]*Call
	outChannels map[string]*ari.ChannelHandle
	bridges     map[string]*Call

	// Eşzamanlı okuma/yazma güvenliği için RWMutex
	// Okumalar (Get) eşzamanlı yapılabilir (RLock), Yazmalar (Add/Remove) bloke edilir (Lock).
	mu sync.RWMutex
}
