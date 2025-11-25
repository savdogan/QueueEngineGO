package main

import (
	"sync"
)

type ServerManager struct {
	sync.RWMutex     `json:"-"`
	MediaServers     map[int64]*WbpServer
	RegistrarServers map[int64]*WbpServer
}

// NewCallManager, CallManager'ın güvenli bir örneğini oluşturur ve başlatır.
func NewServerManager() *ServerManager {
	return &ServerManager{
		MediaServers:     make(map[int64]*WbpServer),
		RegistrarServers: make(map[int64]*WbpServer),
	}
}

func (sm *ServerManager) AddMediaServer(server *WbpServer) error {
	sm.Lock()
	defer sm.Unlock()

	sm.MediaServers[server.ID] = server

	return nil
}

func (sm *ServerManager) RemoveMediaServer(serverId int64) error {
	sm.Lock()
	defer sm.Unlock()

	delete(sm.MediaServers, serverId)

	return nil
}
