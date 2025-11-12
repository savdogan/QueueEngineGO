package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

var AppConfig Config

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
			// Görev çalıştığında CallID'yi kullanır
			//fmt.Printf("--- %d nolu işlem %s: %d saniye sonra planlanan iş çalıştı. ---\n", i, callID, delay/time.Second)
			CustomLog(LevelInfo, "--- %d nolu işlem %s: %d saniye sonra planlanan iş çalıştı. ---\n", i, callID, delay/time.Second)
		})
	}

	planningDuration := time.Since(startTime)
	fmt.Printf("Planlama Tamamlandı: %s sürdü.\n", planningDuration)
	fmt.Println("50000 görev için 20 saniye bekleniyor...")

	// 7 saniye bekleyip programdan çıkmak yerine, 8 saniye bekleyelim ki görevlerin çoğu bitsin
	time.Sleep(30 * time.Second)

	fmt.Println("\nProgram sonlandı.")
}

/*
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
