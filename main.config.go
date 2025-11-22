package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func saveConfigToDisk() error {
	// 1. Cfg'i JSON formatına serileştir
	data, err := json.MarshalIndent(g.Cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config serileştirme hatasi: %w", err)
	}

	// 2. JSON verisini dosyaya yaz
	// os.WriteFile kullanılır. 0644 dosya izinlerini belirtir.
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		return fmt.Errorf("config dosyasına yazma hatasi: %w", err)
	}

	return nil
}

func handleLogDirectory(w http.ResponseWriter, valueStr string) {

	newDir := strings.TrimSpace(valueStr)

	// 1. YAZMA İZNİ KONTROLÜ (Dizin boş değilse)
	if newDir != "" {
		// Geçici bir deneme dosyası yolu oluştur
		testFilePath := filepath.Join(newDir, fmt.Sprintf("write_test_%d.tmp", time.Now().UnixNano()))

		// Dizin mevcut değilse, oluştur
		if err := os.MkdirAll(newDir, 0755); err != nil {
			errMsg := fmt.Sprintf("HATA: Log dizini oluşturulamadı/erişilemiyor (%s). Hata: %v", newDir, err)
			clog(LevelError, "%s", errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Geçici deneme dosyasına yazmayı dene
		err := os.WriteFile(testFilePath, []byte("write test"), 0644)

		// Temizlik: Geçici dosyayı defer ile her durumda sil
		defer os.Remove(testFilePath)

		if err != nil {
			errMsg := fmt.Sprintf("HATA: Dizin yazma izni yok veya dosya sistemi hatası (%s). Hata: %v", newDir, err)
			clog(LevelError, "%s", errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
	}

	// 2. GÜVENLİ YAZMA: Cfg değerini güncelle
	g.Cfg.Lock()
	g.Cfg.LogDirectory = newDir
	g.Cfg.Unlock()

	logDirectoryChanged = true

	if err := saveConfigToDisk(); err != nil {
		http.Error(w, fmt.Sprintf("[CHANGE LOG DIRECTORY] Konfigürasyon güncellendi, ancak diske yazma hatası: %v", err), http.StatusInternalServerError)
		clog(LevelError, "[CONFIG SAVE ERROR] LogDirectory degeri %s olarak guncellendi, ancak diske yazma hatasi: %v", newDir, err)
		return
	}

	// 4. Başarılı yanıt
	clog(LevelInfo, "[CHANGE LOG DIRECTORY] LogDirectory değişmesi için sinyal yollandı: %s", newDir)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "[CHANGE LOG DIRECTORY] LogDirectory değişmesi iiçin sinyal yollandı : %s", newDir)
}

func handleLogLevel(w http.ResponseWriter, cmd string) {

	logLevelTag := ""

	configUpdated := true

	g.Cfg.Lock()

	switch strings.ToLower(cmd) { // Komutu küçük harfe çevirerek karşılaştır
	case "loglevelinfo":
		logLevelTag = "LevelInfo"
		g.Cfg.MinLogLevel = LevelInfo
	case "logleveldebug":
		logLevelTag = "LevelDebug"
		g.Cfg.MinLogLevel = LevelDebug
	case "loglevelerror":
		logLevelTag = "LevelError"
		g.Cfg.MinLogLevel = LevelError
	case "loglevelwarn":
		logLevelTag = "LevelWarn"
		g.Cfg.MinLogLevel = LevelWarn
	case "logleveltrace":
		logLevelTag = "LevelTrace"
		g.Cfg.MinLogLevel = LevelTrace
	default:
		configUpdated = false
		http.Error(w, fmt.Sprintf("Bilinmeyen Loglevel: %s", cmd), http.StatusBadRequest)
	}

	g.Cfg.Unlock()

	if configUpdated {

		w.WriteHeader(http.StatusOK)
		err := saveConfigToDisk()
		if err != nil {
			fmt.Fprintf(w, "[LOGL_EVEL_CHANGED] Set to %s, bu not saved to config file", logLevelTag)
			clog(LevelInfo, "[LOGL_EVEL_CHANGED] Set to %s, bu not saved to config file, err : %+v", logLevelTag, err)
		} else {
			fmt.Fprintf(w, "[LOGL_EVEL_CHANGED] Set to : %s", logLevelTag)
			clog(LevelInfo, "[LOGL_EVEL_CHANGED] Set to %s", logLevelTag)
		}

	} else {
		http.Error(w, "Konfigürasyon parametresi bulunamadi veya eşleştirilemedi.", http.StatusInternalServerError)
	}

}

// Boolean konfigürasyon parametrelerini işleyen yardımcı fonksiyon
func handleBooleanConfig(w http.ResponseWriter, paramName string, valueStr string) {
	// 1. value parametresi kontrolü
	if valueStr == "" {
		http.Error(w, fmt.Sprintf("%s komutu için 'value' parametresi eksik.", paramName), http.StatusBadRequest)
		return
	}

	// 2. Değeri boolean'a çevirme
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		// Sadece "true" veya "false" kabul edilmeli
		http.Error(w, fmt.Sprintf("Geçersiz değer: '%s'. Beklenen 'true' veya 'false' değeridir.", valueStr), http.StatusBadRequest)
		return
	}

	var configUpdated bool

	switch paramName {
	case "LoadSnapshotOnStart":
		g.Cfg.Lock()
		g.Cfg.LoadSnapshotOnStart = value
		g.Cfg.Unlock()
		configUpdated = true
		// Eğer gelecekte başka boolean ayarlarınız olursa buraya eklenecektir.
	}

	if configUpdated {
		// 4. BAŞARILI GÜNCELLEME SONRASI DISKE YAZMA İŞLEMİ
		if err := saveConfigToDisk(); err != nil {
			http.Error(w, fmt.Sprintf("Konfigürasyon güncellendi, ancak diske yazma hatası: %v", err), http.StatusInternalServerError)
			clog(LevelError, "[CONFIG SAVE ERROR] %s degeri %t olarak guncellendi, ancak diske yazma hatasi: %v", paramName, value, err)
			return
		}

		// 5. Başarılı yanıt
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s degeri %t olarak guncellendi ve config.json dosyasina yazildi.", paramName, value)
		clog(LevelInfo, "[CONFIG UPDATE] %s degeri %t olarak guncellendi ve diske kaydedildi.", paramName, value)
	} else {
		http.Error(w, "Konfigürasyon parametresi bulunamadi veya eşleştirilemedi.", http.StatusInternalServerError)
	}
}

func handleIntConfig(w http.ResponseWriter, paramName string, valueStr string) {
	// 1. value parametresi kontrolü
	if valueStr == "" {
		http.Error(w, fmt.Sprintf("%s komutu için 'value' parametresi eksik.", paramName), http.StatusBadRequest)
		return
	}

	// 2. Değeri integer'a çevirme
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Geçersiz değer: '%s'. Beklenen tam sayıdır.", valueStr), http.StatusBadRequest)
		return
	}

	// 3. Cfg yapısındaki ilgili alanı güncelleme (Kilitleme zorunlu)
	var configUpdated bool

	g.Cfg.Lock()

	switch paramName {

	default:
		clog(LevelInfo, "%d", value)
	}

	g.Cfg.Unlock()

	if configUpdated {
		// Başarılı yanıt
		w.WriteHeader(http.StatusOK)
		err := saveConfigToDisk()
		if err != nil {
			fmt.Fprintf(w, "%s degeri %d olarak guncellendi, fakat dosyaya kalıcı olarak yazılamadı!!!", paramName, value)
			clog(LevelInfo, "[CONFIG UPDATE] %s degeri %d olarak guncellendi. Error : %+v", paramName, value, err)
		} else {
			fmt.Fprintf(w, "%s degeri %d olarak guncellendi.", paramName, value)
			clog(LevelInfo, "[CONFIG UPDATE] %s degeri %d olarak guncellendi.", paramName, value)
		}

	} else {
		http.Error(w, "Konfigürasyon parametresi bulunamadi veya eşleştirilemedi.", http.StatusInternalServerError)
	}
}

func loadConfig(path string) *Config {
	var cfg Config

	// Dosyayı oku
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("[ERROR] :config dosyası okunamadi: %+v", err)
	}

	// JSON verisini struct'a çözümle
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("[ERROR] :config JSON çözümlenemedi: %+v", err)
	}

	log.Printf("Successed to load config")

	return &cfg
}
