package main

import (
	"context"
	"log"
	"sync"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// Global Log Seviyesi Değişkeni: CustomLog'un erişimi için config'den buraya aktarılacak
var cfgMinLogLevel LogLevel
var AppConfig Config

func main() {

	version := 1

	log.Printf("QueueEngineGO version:%d is starting...", version)

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. KONFİGÜRASYONU Yükle
	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}
	AppConfig := cfg

	log.Printf("Successed to load config")

	cfgPublisherHostName := getHostname()
	AppConfig.mu = &sync.RWMutex{}
	cfgLogDirectory = AppConfig.LogDirectory
	cfgMinLogLevel = AppConfig.MinLogLevel
	//redisAddresses := AppConfig.RedisAddresses
	//redisPassword := AppConfig.RedisPassword
	//LoadSnapshotOnStart := AppConfig.LoadSnapshotOnStart
	AppConfig.mu.Lock()
	AppConfig.PublisherHostName = cfgPublisherHostName
	AppConfig.Version = version
	log.Printf("PublisherHostName : %s", AppConfig.PublisherHostName)
	log.Printf("Version : %d", AppConfig.Version)
	AppConfig.mu.Unlock()

	log.Printf("Async logging is starting...")

	startAsyncLogger()
	time.Sleep(5 * time.Second)

	manager := NewClientManager()
	var wg sync.WaitGroup

	CustomLog(LevelInfo, "Ari connections is starting...")
	// 3. ARI Bağlantılarını Başlat
	for _, ariCfg := range AppConfig.AriConnections {
		wg.Add(1)

		go func(ariCfg AriConfig) {
			defer wg.Done()
			if err := runApp(ctx, ariCfg, manager); err != nil {
				CustomLog(LevelError, "ARI application failed to start for %s: %v", ariCfg.Application, err)
			}
		}(ariCfg)
	}

	// 4. HTTP Sunucusunu Başlat (Config'den portu kullanarak)
	if AppConfig.HttpServerEnabled {
		startHttpEnabled(manager)
	} else {
		CustomLog(LevelInfo, "HTTP Server is disabled via config.")
	}

	// 1. SQL Server Bağlantısını KUR (Hata olursa burada durur)
	sqlInstance := "GAVWSQLTST01.global-bilgi.entp"
	sqlDB := "gbWebPhone_test"
	sqlUser := "[GLOBAL-BILGI\\savdogan]" // Kullanıcı bilgisi gerekli değil ancak loglamada tutulabilir

	if err := InitDBConnection(sqlInstance, sqlDB, sqlUser); err != nil {
		CustomLog(LevelFatal, "Veritabanı bağlantısı kurulamadı: %v", err)
		return
	}
	defer CloseDBConnection() // Uygulama sonlandığında bağlantıyı kapat

	InitQueueManager()

	go func() {

		time.Sleep(5 * time.Second)

		// Kullanım Örneği (Örneğin StasisStart geldikten sonra)
		queueName := "Yuktesti" // Varsayımsal kuyruk adı
		queueDef, err := globalQueueManager.GetQueueByName(queueName)

		if err != nil {
			// Kuyruk tanımı bulunamadı veya DB hatası var
			CustomLog(LevelError, "Kuyruk tanımı alınamadı: %v, %d", err, queueDef.ID)
			return
		}

	}()

	// Uygulamanın çalışmasını sağla
	wg.Wait()
	CustomLog(LevelInfo, "All services shut down. Main exiting.")
}

/*

package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

var AppConfig Config

var pc int64

func main() {

	version := 1

	// 2. 🔥 KRİTİK ADIM: CONFIG DOSYASINI OKUMA VE YÜKLEME
	cfg, err := loadConfig("config.json")
	if err != nil {
		// Hata durumunda net bir mesaj basılır ve sistem log.Fatalf ile durdurulur.
		// log.Fatalf çağrısı, log basıldıktan sonra os.Exit(1) çağırarak programı sonlandırır.
		log.Fatalf("[FATAL ERROR] Konfigürasyon yüklenemedi: %v", err)
		// Bu noktadan sonra kod çalışmaya devam etmez.
	}

	AppConfig = cfg
	AppConfig.mu = &sync.RWMutex{}

	log.Printf("[SETUP] Konfigürasyon başarıyla yüklendi. Ortam: %s , %+v", AppConfig.Environment, cfg)

	// 1. Asenkron loglama sistemini başlat
	startAsyncLogger()
	log.Printf("[VERSION] : %d [SETUP] Asenkron loglama aktif edildi.", version)

	fmt.Println("=== Go Gecikmeli İş Scheduler Başlatıldı ===")
	scheduler := NewScheduler()

	// Simüle edilecek çağrı sayısı
	const callCount = 50000

	time.Sleep(3 * time.Second)

	// Planlama işleminin başlangıç süresi
	startTime := time.Now()

	fmt.Printf("Başlangıç: %d adet planlı görev oluşturuluyor...\n", callCount)

	for i := 0; i < callCount; i++ {

		// DÖNGÜ DEĞİŞKENİNİ KOPYALA:
		// i değişkeni, döngü her döndüğünde değişir.
		// Goroutine/Task'ın doğru CallID'yi görmesi için kopyalanmalıdır.
		// Eğer kopyalamazsak, tüm task'lar son 'i' değerini (49999) görür.
		callID := fmt.Sprintf("Call-%d", i)

		// Örnek: Task'ları rastgele veya sabit bir süre sonra planlayabiliriz.
		// Bu örnekte, basitlik için tüm görevler 7 saniye sonra planlanıyor.
		delay := 1 * time.Second

		scheduler.ScheduleTask(callID, delay, func() {
			atomic.AddInt64(&pc, 1)
			// Görev çalıştığında CallID'yi kullanır
			//fmt.Printf("--- %d nolu işlem %s: %d saniye sonra planlanan iş çalıştı. ---\n", i, callID, delay/time.Second)
			//currentTimeMilli := time.Now().Format("2006/01/02 15:04:05.000")
			CustomLog(LevelInfo, "--- nolu işlem %d: %s saniye sonra planlanan iş çalıştı. ---%s\n", i, callID, delay/time.Second)
		})
	}

	planningDuration := time.Since(startTime)
	fmt.Printf("Planlama Tamamlandı: %s sürdü.\n", planningDuration)
	fmt.Println("50000 görev için 20 saniye bekleniyor...")

	for i := 0; i < 20; i++ {
		time.Sleep(5 * time.Second)
		fmt.Printf("Başlangıç: %d adet planlı çalıştırıldı...\n", pc)
	}

	// 7 saniye bekleyip programdan çıkmak yerine, 8 saniye bekleyelim ki görevlerin çoğu bitsin
	log.Printf("[SERVER] SYSTEM IS ACTIVE NOW")
	select {} // Sonsuza kadar çalış
}


func main() {
	fmt.Println("=== Go Gecikmeli İş Scheduler Başlatıldı ===")
	scheduler := NewScheduler()

	// 1. İş: 4 saniye sonra çalışacak (Call 1)
	scheduler.ScheduleTask("Call-123", 4*time.Second, func() {
		fmt.Println("\n--- Call-123: 4 saniye sonra planlanan iş çalıştı. ---")
	})

	// 1. İş: 4 saniye sonra çalışacak (Call 1)
	scheduler.ScheduleTask("Call-123", 2*time.Second, func() {
		fmt.Println("\n--- Call-123: 2 saniye sonra planlanan iş çalıştı. ---")
	})

	// 2. İş: 10 saniye sonra çalışacak (Call 456)
	scheduler.ScheduleTask("Call-456", 10*time.Second, func() {
		fmt.Println("\n--- Call-456: 10 saniye sonra planlanan iş çalıştı. ---")
	})

	// 3. İş: 7 saniye sonra çalışacak (Call 123'e ait 2. iş)
	scheduler.ScheduleTask("Call-123", 7*time.Second, func() {
		fmt.Println("\n--- Call-123: 7 saniye sonra planlanan iş çalıştı. ---")
	})



	fmt.Println("\n3 saniye bekliyoruz ve Call-123'ü iptal ediyoruz (Bu, 4s ve 7s işlerini siler).")
	time.Sleep(8 * time.Second)

	// Call-123'e ait tüm işleri iptal et
	scheduler.CancelByCallID("Call-123")

	fmt.Println("10 saniyelik işin çalışmasını bekliyoruz.")

	// Programın hemen bitmemesi için bekleyin (11 saniye, 10 saniyelik işin çalışması için)
	time.Sleep(9 * time.Second)

	fmt.Println("\n=== Scheduler Kapatılıyor. ===")
} */
