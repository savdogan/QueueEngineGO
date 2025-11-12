package main

import (
	"sync"
	"time"
)

// ScheduledTask: Zamanlanmış işi ve ilişkili verileri tutar.
type ScheduledTask struct {
	CallID    string    // Hangi çağrıya ait olduğunu gösteren ID
	ExecuteAt time.Time // İşin çalışacağı zaman (Heap için öncelik)
	Action    func()    // Çalıştırılacak fonksiyon
	index     int       // Heap içindeki pozisyonu (silme/güncelleme için kritik)
}

type LogLevel int

const (
	LevelFatal LogLevel = iota // 0: Kritik sistem hatası
	LevelError                 // 1: Hata durumları (Her zaman açık kalmalı)
	LevelWarn                  // 2: Uyarılar
	LevelInfo                  // 3: Genel Bilgi (Standart loglar)
	LevelDebug                 // 4: Detaylı hata ayıklama (Geliştirmede açık)
	LevelTrace                 // 5: Herşey
)

type Config struct {
	Environment                       string        `json:"Environment"`
	RedisAddresses                    []string      `json:"RedisAddresses"`
	RedisPassword                     string        `json:"RedisPassword"`
	LoadSnapshotOnStart               bool          `json:"LoadSnapshotOnStart"` // m.LoadSnapshot() kontrolü için
	MinLogLevel                       LogLevel      `json:"MinLogLevel"`
	RejectedCallMinWaitTime           int           `json:"RejectedCallMinWaitTime"`
	RejectedCallCleanupCount          int           `json:"RejectedCallCleanupCount"`
	RejectedCallProcessingInterval    int           `json:"RejectedCallProcessingInterval"`
	RejectedCallWaitingMinQueueLength int           `json:"RejectedCallWaitingMinQueueLength"`
	PublisherHostName                 string        `json:"publisherHostName"`
	LogDirectory                      string        `json:"logDirectory"`
	mu                                *sync.RWMutex // Yapılandırma okuma/yazma işlemleri için
	HttpServerEnabled                 bool          `json:"HttpServerEnabled"` // m.LoadSnapshot() kontrolü için
	Version                           int           `json:"Version"`
	HttpPort                          int           `json:"HttpPort"`
}
