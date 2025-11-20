package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var logChannel = make(chan string, 1000)
var currentLogFile *os.File
var currentLogDate string

// ChannelWriter: log.SetOutput'u kanala yönlendirir (Adım 2)
type ChannelWriter struct{}

func (writer *ChannelWriter) Write(p []byte) (n int, err error) {
	select {
	case logChannel <- string(p):
		return len(p), nil
	default:
		// Kanal doluysa atla, bekletme
		return len(p), nil
	}
}

var cfgLogDirectory string

func clog(level LogLevel, format string, v ...interface{}) {
	// 1. Kontrol: Log seviyesi minimum seviyeden düşük öncelikli mi?
	if level > cfgMinLogLevel {
		return
	}

	// 2. Seviye Etiketi Ekleme
	var levelTag string
	switch level {
	case LevelFatal:
		levelTag = "[FATAL] "
	case LevelError:
		levelTag = "[ERROR] "
	case LevelWarn:
		levelTag = "[WARN] "
	case LevelInfo:
		levelTag = "[INFO] "
	case LevelDebug:
		levelTag = "[DEBUG] "
	case LevelTrace:
		levelTag = "[TRACE] "
	default:
		levelTag = "[???]    "
	}

	// 3. Logu Basma
	message := levelTag + fmt.Sprintf(format, v...)

	// 2, clog'u çağıran fonksiyonu işaret eder.
	log.Output(2, message)

	// Fatal seviyesinde sistemden çıkış yapılması
	if level == LevelFatal {
		go func() {
			terminateApplication(5)
		}()
	}
}

// setLogFile: Dosya açma ve log çıkışını ayarlama işini yapar (Adım 3 ve 4 için hazırlık)
func setLogFile() error {
	// ... Günlük log döndürme mantığı burada yer alır ...

	today := time.Now().Format("2006-01-02")

	baseFileName := fmt.Sprintf("app-async-%s.log", today)

	var logFilePath string

	// 2. AppConfig'deki LogDirectory alanını kontrol et
	// Dizin ismini kontrol ederken trim ile baştaki/sondaki boşlukları temizlemek iyi bir uygulamadır.

	logDir := cfgLogDirectory // Varsayımsal global değişken

	if logDir != "" {
		// Dizin ismi mevcutsa, dizin ve dosya adını birleştir
		// filepath.Join, OS'e uygun ayırıcıları otomatik olarak kullanır.
		logFilePath = filepath.Join(logDir, baseFileName)

		// Önemli: Eğer dizin mevcut değilse, loglama başlamadan önce oluşturmalısınız.
		// os.MkdirAll(logDir, 0755) // Bu satırı loglama başlangıcına eklemeyi düşünün.

	} else {
		// Dizin ismi boşsa, sadece dosya adını kullan
		logFilePath = baseFileName
	}

	// Dosya açma (veya oluşturma) işlemi
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	if currentLogFile != nil {
		currentLogFile.Close()
	}
	currentLogFile = file
	currentLogDate = today

	// Çıktıyı ChannelWriter'a yönlendir (Adım 3)
	log.SetOutput(&ChannelWriter{})
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile | log.Lmicroseconds)

	return nil
}

var logDirectoryChanged = false

// startAsyncLogger: Kanaldan okuyup dosyaya yazan goroutine'i başlatır (Adım 4)
func startAsyncLogger() error {
	if err := setLogFile(); err != nil {
		log.Fatalf("Log sistemi başlatılamadı: %v", err)
		return err
	}

	go func() {
		// Bu goroutine I/O işlemini yapar. Ana akışı etkilemez.
		for logMsg := range logChannel {
			// Günlük döndürme kontrolü ve yazma işlemi
			today := time.Now().Format("2006-01-02")
			if today != currentLogDate || logDirectoryChanged {
				// Yeni gün gelince dosyayı döndür
				if err := setLogFile(); err != nil {
					log.Printf("[LOGGER ERROR] Yeni log dosyası oluşturulamadı: %v\n", err)
					continue
				}

				logDirectoryChanged = false
			}

			if currentLogFile != nil {
				_, err := currentLogFile.WriteString(logMsg)
				if err != nil {
					log.Printf("[LOGGER ERROR] Dosyaya yazma hatası: %v | Mesaj: %s\n", err, logMsg)
				}
			}
		}
	}()

	return nil
}
