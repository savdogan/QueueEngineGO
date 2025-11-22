package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var cfgMinLogLevel LogLevel

type GlobalState struct {
	DB    *sql.DB
	QCM   *QueueCacheManager
	ACM   *ClientManager
	CM    *CallManager
	SM    *ScheduleManager
	RPubs *redis.ClusterClient
	RSubs *redis.ClusterClient
	Ctx   context.Context
	Cfg   *Config
}

var (
	// Global değişkenimiz
	g *GlobalState
	// Sadece bir kez çalışmasını garanti eden yapı
	once sync.Once
)

// InitGlobalState: Global durumu başlatır ve g değişkenine atar.
// Hata alırsak uygulamayı durdurmak (panic) genellikle başlangıç aşamasında tercih edilir.
func InitGlobalState(cfg *Config, version int) {
	once.Do(func() {

		// --- 1. ADIM: CONFIG DOĞRULAMA ---
		// Eğer config hatalıysa log.Fatal ile uygulamayı hemen durduruyoruz.
		// Böylece yanlış DB stringi ile bağlanmaya çalışıp vakit kaybetmez.
		if err := cfg.Validate(); err != nil {
			log.Fatalf("BAŞLANGIÇ HATASI (Config): %v", err)
		}

		log.Printf("Konfigürasyon doğrulandı. Ortam: %s", cfg.Environment)

		cfgPublisherHostName := getHostname()
		cfgLogDirectory = cfg.LogDirectory
		cfgMinLogLevel = cfg.MinLogLevel
		cfg.PublisherHostName = cfgPublisherHostName
		cfg.Version = version

		var err error

		// 1. Ana Context Oluşturma (Uygulama yaşam döngüsü için)
		// Not: Struct içinde *context.Context yerine context.Context tutmanızı öneririm.
		// Ancak yapınıza sadık kalarak pointer olarak alıyorum.
		bgCtx := context.Background()

		// 2. Veritabanı Bağlantısı (SQL Server)
		log.Println("SQL Server bağlantısı kuruluyor...")
		db, err := sql.Open("sqlserver", cfg.DBConnectingString)
		if err != nil {
			log.Fatalf("Veritabanı sürücü hatası: %v", err)
		}

		// Bağlantı havuzu ayarları (Performans için önemli)
		db.SetMaxOpenConns(100) // Aynı anda açık maksimum bağlantı
		db.SetMaxIdleConns(10)  // Boşta bekleyecek bağlantı
		db.SetConnMaxLifetime(time.Hour)

		// Bağlantıyı test et (Ping)
		ctx, cancel := context.WithTimeout(bgCtx, 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("Veritabanına erişilemedi: %v", err)
		}

		// 3. Redis Bağlantıları (Pub ve Sub için ayrı)
		log.Println("Redis Cluster bağlantıları kuruluyor...")

		// Publisher Client
		rPubs := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.RedisAddresses,
			Password: cfg.RedisPassword,
			// Diğer timeout ayarları buraya eklenebilir
		})
		if err := rPubs.Ping(ctx).Err(); err != nil {
			log.Fatalf("Redis Pubs bağlantı hatası: %v", err)
		}

		// Subscriber Client (Ayrı olması şarttır)
		rSubs := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.RedisAddresses,
			Password: cfg.RedisPassword,
		})
		if err := rSubs.Ping(ctx).Err(); err != nil {
			log.Fatalf("Redis Subs bağlantı hatası: %v", err)
		}

		// 4. Manager'ların Başlatılması (Dependency Injection)
		// Burada DB ve Redis'i manager'ların içine gönderiyoruz.

		// Örnek: Queue Cache Manager başlatılıyor
		qcm := NewQueueCacheManager(db)

		// Örnek: Call Manager (Hem DB hem Redis'e ihtiyacı olabilir)
		cm := NewCallManager()

		// Örnek: Schedule Manager
		sm := NewSchedulerManager()

		// Örnek: Client Manager
		acm := NewClientManager()

		log.Println("Tüm bileşenler başarıyla başlatıldı.")

		// 5. Global Değişkene Atama
		g = &GlobalState{
			DB:    db,
			QCM:   qcm,
			ACM:   acm,
			CM:    cm,
			SM:    sm,
			RPubs: rPubs,
			RSubs: rSubs,
			Ctx:   bgCtx,
			Cfg:   cfg,
		}
	})
}

// Validate: Konfigürasyonun tutarlılığını kontrol eder.
func (c *Config) Validate() error {
	// 1. Environment Kontrolü
	if c.Environment == "" {
		return fmt.Errorf("konfigürasyon hatası: 'Environment' boş olamaz")
	}

	// 2. Redis Adresleri Kontrolü
	if len(c.RedisAddresses) == 0 {
		return fmt.Errorf("konfigürasyon hatası: 'RedisAddresses' listesi boş, en az bir node gerekli")
	}
	for _, addr := range c.RedisAddresses {
		if !strings.Contains(addr, ":") {
			return fmt.Errorf("konfigürasyon hatası: Redis adresi hatalı formatta (%s). Örn: 127.0.0.1:6379", addr)
		}
	}

	// 3. InstanceIDs Kontrolü (YENİ EKLENDİ)
	// 3.a. Boş mu kontrolü
	if len(c.InstanceIDs) == 0 {
		return fmt.Errorf("konfigürasyon hatası: 'InstanceIDs' listesi boş olamaz, en az bir ID gerekli")
	}

	// 3.b. Duplike (Mükerrer) kontrolü
	// Bir map oluşturup gördüğümüz ID'leri buraya not ediyoruz.
	seenInstances := make(map[string]bool)
	for _, id := range c.InstanceIDs {
		if id == "" {
			return fmt.Errorf("konfigürasyon hatası: 'InstanceIDs' içinde boş bir değer var")
		}
		if seenInstances[id] {
			return fmt.Errorf("konfigürasyon hatası: 'InstanceIDs' içinde mükerrer kayıt tespit edildi -> %s", id)
		}
		seenInstances[id] = true
	}

	// 4. Database String Kontrolü
	if c.DBConnectingString == "" {
		return fmt.Errorf("konfigürasyon hatası: 'DBConnectingString' boş olamaz")
	}

	// 5. HTTP Server Kontrolü
	if c.HttpServerEnabled {
		if c.HttpPort <= 0 || c.HttpPort > 65535 {
			return fmt.Errorf("konfigürasyon hatası: 'HttpPort' geçerli bir port aralığında değil (1-65535): %d", c.HttpPort)
		}
	}

	// 6. Log Dizini
	if c.LogDirectory == "" {
		return fmt.Errorf("konfigürasyon hatası: 'LogDirectory' belirtilmemiş")
	}

	// 7. ARI Connections (Asterisk) Kontrolü
	if len(c.AriConnections) == 0 {
		return fmt.Errorf("uyarı: Hiçbir 'ari_connections' tanımlanmamış. Çağrı yönetimi çalışmayabilir")
	}

	// ARI bağlantılarında duplike ID kontrolü için map (YENİ EKLENDİ)
	seenAriIDs := make(map[string]bool)

	for i, conn := range c.AriConnections {
		// ID Boşluk Kontrolü
		if conn.Id == "" {
			return fmt.Errorf("ari_connections[%d]: 'Id' alanı eksik", i)
		}

		// ID Duplike Kontrolü (YENİ EKLENDİ)
		if seenAriIDs[conn.Id] {
			return fmt.Errorf("konfigürasyon hatası: 'ari_connections' içinde mükerrer ID tespit edildi -> %s", conn.Id)
		}
		seenAriIDs[conn.Id] = true

		// Kullanıcı Adı
		if conn.Username == "" {
			return fmt.Errorf("ari_connections[%d] (ID: %s): 'Username' eksik", i, conn.Id)
		}

		// URL Validasyonları (RestUrl)
		if _, err := url.ParseRequestURI(conn.RestURL); err != nil {
			return fmt.Errorf("ari_connections[%d] (ID: %s): Geçersiz 'RestUrl' -> %s", i, conn.Id, conn.RestURL)
		}

		// URL Validasyonları (WebsocketURL)
		if _, err := url.ParseRequestURI(conn.WebsocketURL); err != nil {
			return fmt.Errorf("ari_connections[%d] (ID: %s): Geçersiz 'WebsocketURL' -> %s", i, conn.Id, conn.WebsocketURL)
		}
	}

	return nil
}
