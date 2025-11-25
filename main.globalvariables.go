package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var cfgMinLogLevel LogLevel

type GlobalState struct {
	DB            *sql.DB
	QCM           *QueueCacheManager
	ACM           *ClientManager
	CM            *CallManager
	SM            *ScheduleManager
	RPubs         *redis.ClusterClient
	RSubs         *redis.ClusterClient
	Ctx           context.Context
	Cfg           *Config
	ServerManager *ServerManager
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
		ctx, cancel := context.WithTimeout(bgCtx, 10*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("Veritabanına erişilemedi: %v", err)
		}

		configMap, err := getAppConfigsAsMap(db)
		if err != nil {
			log.Fatalf("QEConfig could not loaded from database : %v", err)
		}
		cfg.QEAppConfig = configMap

		serverManager := NewServerManager()

		mediaServers, err := getAllServersWithType(db, 1)
		if err != nil {
			log.Fatalf("Media servers could not loaded : %v", err)
		}
		serverManager.MediaServers = mediaServers

		// 7. ARI Connections (Asterisk) Kontrolü
		if len(serverManager.MediaServers) == 0 {
			log.Fatalf("There are no media server definition")
		}

		registrarServers, err := getAllServersWithType(db, 2)
		if err != nil {
			log.Fatalf("Media servers could not loaded : %v", err)
		}
		serverManager.RegistrarServers = registrarServers

		// 3. Redis Bağlantıları (Pub ve Sub için ayrı)
		log.Println("Redis Cluster bağlantıları kuruluyor...")

		// Publisher Client
		rPubs := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.getRedisServers(),
			Password: cfg.getConfigValue("redis.password", true),
			// Diğer timeout ayarları buraya eklenebilir
		})
		if err := rPubs.Ping(ctx).Err(); err != nil {
			log.Fatalf("Redis Pubs bağlantı hatası: %v", err)
		}

		// Subscriber Client (Ayrı olması şarttır)
		rSubs := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.getRedisServers(),
			Password: cfg.getConfigValue("redis.password", true),
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
			DB:            db,
			QCM:           qcm,
			ACM:           acm,
			CM:            cm,
			SM:            sm,
			RPubs:         rPubs,
			RSubs:         rSubs,
			Ctx:           bgCtx,
			Cfg:           cfg,
			ServerManager: serverManager,
		}
	})
}

// Validate: Konfigürasyonun tutarlılığını kontrol eder.
func (c *Config) Validate() error {
	// 1. Environment Kontrolü
	if c.Environment == "" {
		return fmt.Errorf("konfigürasyon hatası: 'Environment' boş olamaz")
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

	return nil
}
