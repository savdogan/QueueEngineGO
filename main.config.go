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

func saveAppConfigToDisk() error {
	// 1. AppConfig'i JSON formatına serileştir
	data, err := json.MarshalIndent(AppConfig, "", "  ")
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
			CustomLog(LevelError, "%s", errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Geçici deneme dosyasına yazmayı dene
		err := os.WriteFile(testFilePath, []byte("write test"), 0644)

		// Temizlik: Geçici dosyayı defer ile her durumda sil
		defer os.Remove(testFilePath)

		if err != nil {
			errMsg := fmt.Sprintf("HATA: Dizin yazma izni yok veya dosya sistemi hatası (%s). Hata: %v", newDir, err)
			CustomLog(LevelError, "%s", errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
	}

	// 2. GÜVENLİ YAZMA: AppConfig değerini güncelle
	AppConfig.mu.Lock()
	AppConfig.LogDirectory = newDir
	AppConfig.mu.Unlock()

	logDirectoryChanged = true

	if err := saveAppConfigToDisk(); err != nil {
		http.Error(w, fmt.Sprintf("[CHANGE LOG DIRECTORY] Konfigürasyon güncellendi, ancak diske yazma hatası: %v", err), http.StatusInternalServerError)
		CustomLog(LevelError, "[CONFIG SAVE ERROR] LogDirectory degeri %s olarak guncellendi, ancak diske yazma hatasi: %v", newDir, err)
		return
	}

	// 4. Başarılı yanıt
	CustomLog(LevelInfo, "[CHANGE LOG DIRECTORY] LogDirectory değişmesi için sinyal yollandı: %s", newDir)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "[CHANGE LOG DIRECTORY] LogDirectory değişmesi iiçin sinyal yollandı : %s", newDir)
}

func handleLogLevel(w http.ResponseWriter, cmd string) {

	logLevelTag := ""

	configUpdated := true

	AppConfig.mu.Lock()

	switch strings.ToLower(cmd) { // Komutu küçük harfe çevirerek karşılaştır
	case "loglevelinfo":
		logLevelTag = "LevelInfo"
		AppConfig.MinLogLevel = LevelInfo
	case "logleveldebug":
		logLevelTag = "LevelDebug"
		AppConfig.MinLogLevel = LevelDebug
	case "loglevelerror":
		logLevelTag = "LevelError"
		AppConfig.MinLogLevel = LevelError
	case "loglevelwarn":
		logLevelTag = "LevelWarn"
		AppConfig.MinLogLevel = LevelWarn
	case "logleveltrace":
		logLevelTag = "LevelTrace"
		AppConfig.MinLogLevel = LevelTrace
	default:
		configUpdated = false
		http.Error(w, fmt.Sprintf("Bilinmeyen Loglevel: %s", cmd), http.StatusBadRequest)
	}

	AppConfig.mu.Unlock()

	if configUpdated {

		w.WriteHeader(http.StatusOK)
		err := saveAppConfigToDisk()
		if err != nil {
			fmt.Fprintf(w, "[LOGL_EVEL_CHANGED] Set to %s, bu not saved to config file", logLevelTag)
			CustomLog(LevelInfo, "[LOGL_EVEL_CHANGED] Set to %s, bu not saved to config file, err : %+v", logLevelTag, err)
		} else {
			fmt.Fprintf(w, "[LOGL_EVEL_CHANGED] Set to : %s", logLevelTag)
			CustomLog(LevelInfo, "[LOGL_EVEL_CHANGED] Set to %s", logLevelTag)
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
		AppConfig.mu.Lock()
		AppConfig.LoadSnapshotOnStart = value
		AppConfig.mu.Unlock()
		configUpdated = true
		// Eğer gelecekte başka boolean ayarlarınız olursa buraya eklenecektir.
	}

	if configUpdated {
		// 4. BAŞARILI GÜNCELLEME SONRASI DISKE YAZMA İŞLEMİ
		if err := saveAppConfigToDisk(); err != nil {
			http.Error(w, fmt.Sprintf("Konfigürasyon güncellendi, ancak diske yazma hatası: %v", err), http.StatusInternalServerError)
			CustomLog(LevelError, "[CONFIG SAVE ERROR] %s degeri %t olarak guncellendi, ancak diske yazma hatasi: %v", paramName, value, err)
			return
		}

		// 5. Başarılı yanıt
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s degeri %t olarak guncellendi ve config.json dosyasina yazildi.", paramName, value)
		CustomLog(LevelInfo, "[CONFIG UPDATE] %s degeri %t olarak guncellendi ve diske kaydedildi.", paramName, value)
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

	// 3. AppConfig yapısındaki ilgili alanı güncelleme (Kilitleme zorunlu)
	var configUpdated bool

	AppConfig.mu.Lock()

	switch paramName {

	default:
		CustomLog(LevelInfo, "%d", value)
	}

	AppConfig.mu.Unlock()

	if configUpdated {
		// Başarılı yanıt
		w.WriteHeader(http.StatusOK)
		err := saveAppConfigToDisk()
		if err != nil {
			fmt.Fprintf(w, "%s degeri %d olarak guncellendi, fakat dosyaya kalıcı olarak yazılamadı!!!", paramName, value)
			CustomLog(LevelInfo, "[CONFIG UPDATE] %s degeri %d olarak guncellendi. Error : %+v", paramName, value, err)
		} else {
			fmt.Fprintf(w, "%s degeri %d olarak guncellendi.", paramName, value)
			CustomLog(LevelInfo, "[CONFIG UPDATE] %s degeri %d olarak guncellendi.", paramName, value)
		}

	} else {
		http.Error(w, "Konfigürasyon parametresi bulunamadi veya eşleştirilemedi.", http.StatusInternalServerError)
	}
}

func loadConfig(path string) (Config, error) {
	var cfg Config

	// Dosyayı oku
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[ERROR] :config dosyası okunamadi: %+v", err)
		return cfg, err
	}

	// JSON verisini struct'a çözümle
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Printf("[ERROR] :config JSON çözümlenemedi: %+v", err)
		return cfg, err
	}

	return cfg, nil
}
