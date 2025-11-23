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

type CallManager struct {
	sync.RWMutex
	calls           map[string]*Call
	outChannels     map[string]*ari.ChannelHandle
	agentCalltoCall map[string]string
}
