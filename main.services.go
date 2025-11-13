package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

func startHttpEnabled(manager *ClientManager) {
	go func() {
		listenAddr := fmt.Sprintf(":%d", AppConfig.HttpPort)
		CustomLog(LevelInfo, "Listening for HTTP requests on %s", listenAddr)

		// HTTP Handler'ı tanımlama
		http.Handle("/call", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			appName := r.URL.Query().Get("app")
			if appName == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Error: 'app' query parameter is required (e.g., /call?app=gbqe-client1)"))
				return
			}

			cl, ok := manager.GetClient(appName)
			if !ok {
				CustomLog(LevelWarn, "Client not found for app: %s", appName)
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("ARI Client not found for app: " + appName))
				return
			}

			h, err := createCall(cl)
			if err != nil {
				CustomLog(LevelError, "Failed to create call via %s: %v", appName, err)
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte("Failed to create call: " + err.Error()))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Call initiated on " + appName + " with ID: " + h.ID()))
		}))

		// 🔹 HTTP üzerinden event alma (/event)
		http.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
			var e HttpEvent
			if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			CustomLog(LevelInfo, "[HTTP] Event received: %+v", e)
			//go m.handleEvent(e)
			w.WriteHeader(http.StatusOK)
		})

		http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {

			clientAddr := r.RemoteAddr

			// Sadece IP adresini almak isterseniz:
			ip, _, err := net.SplitHostPort(clientAddr)
			if err == nil {
				CustomLog(LevelInfo, "İstek IP: %s", ip)
			} else {
				CustomLog(LevelInfo, "İstek Adresi: %s", clientAddr)
			}

			// URL Query (sorgu) parametrelerini al
			q := r.URL.Query()
			cmd := q.Get("cmd")
			valueStr := q.Get("value") // Konfigürasyon değeri

			if cmd == "" {
				http.Error(w, "cmd parameter is missing", http.StatusBadRequest)
				return
			}

			CustomLog(LevelInfo, "[COMMAND] New command received: %s, value: %s", cmd, valueStr)

			// Komuta göre farklı işlemleri yap
			switch strings.ToLower(cmd) {
			case "loglevelinfo", "logleveldebug", "loglevelwarn", "loglevelerror", "logleveltrace":
				handleLogLevel(w, cmd)
				/*
					case "metrics":
						metrics := m.handleMetrics()
						now := time.Now()
						for k, v := range metrics {
							v.LastUpdated = now // Kopyayı güncelledik
							metrics[k] = v      // Güncellenmiş kopyayı map'e geri yazdık
						}
						w.Header().Set("Content-Type", "application/json")
						if err := json.NewEncoder(w).Encode(metrics); err != nil {
							http.Error(w, "JSON serialization error", http.StatusInternalServerError)
							return
						}
					case "getalldata":
						m.mu.RLock()

						// 2. Kilidi serbest bırakmayı (Unlock) ertele (defer)
						// Böylece, fonksiyon sona erdiğinde (başarıyla veya hatayla), kilit otomatik olarak açılır.
						defer m.mu.RUnlock()

						// 3. Yanıt başlığını ayarla
						w.Header().Set("Content-Type", "application/json")

						// 4. Veriyi JSON olarak kodla (encode)
						// KRİTİK NOT: JSON dönüşümü kilit (RLock) altındayken yapılmalıdır,
						// böylece dönüştürme sırasında başka bir Go rutini veriyi değiştiremez.
						if err := json.NewEncoder(w).Encode(m); err != nil {
							// Eğer JSON dönüşümü hata verirse, kilidi serbest bırakmak için 'defer' çalışacaktır.
							http.Error(w, "JSON serialization error", http.StatusInternalServerError)
							return
						}
				*/
			case "setlogdirectory":
				handleLogDirectory(w, valueStr) // Yeni handler'ı çağır
			case "getconfig":
				// AppConfig nesnesinin kilitlenmesini sağlayarak güvenli bir şekilde okuma yapılır.
				configCopy := AppConfig // Yapının kopyasını al
				w.Header().Set("Content-Type", "application/json")
				// Kopyalanan config verisi JSON'a serileştirilir
				if err := json.NewEncoder(w).Encode(configCopy); err != nil {
					http.Error(w, "JSON serialization error", http.StatusInternalServerError)
					return
				}
			case "setloadsnapshotonstart":
				handleBooleanConfig(w, "LoadSnapshotOnStart", valueStr)
			case "setrejectedcallminwaittime":
				handleIntConfig(w, "RejectedCallMinWaitTime", valueStr)
			case "setrejectedcallcleanupcount":
				handleIntConfig(w, "RejectedCallCleanupCount", valueStr)
			case "setrejectedcallprocessinginterval":
				handleIntConfig(w, "RejectedCallProcessingInterval", valueStr)
			case "setrejectedcallwaitingminqueuelength":
				handleIntConfig(w, "RejectedCallWaitingMinQueueLength", valueStr)

			default:
				http.Error(w, fmt.Sprintf("Unknown Command : %s", cmd), http.StatusBadRequest)
			}
		})

		// 🔹 Konsol Servisi (/console)
		http.HandleFunc("/console", func(w http.ResponseWriter, r *http.Request) {
			// 1. HTML dosyasını diskten oku
			htmlContent, err := os.ReadFile("console.html")

			if err != nil {
				// Dosya bulunamazsa veya okuma hatası olursa 500 hatası döndür
				http.Error(w, fmt.Sprintf("Console file is not found: %s. check file path.", "console.html"), http.StatusInternalServerError)
				CustomLog(LevelError, "Console file reading error: %v", err)
				return
			}

			// 2. Başlıkları ayarla ve içeriği yaz
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(htmlContent)
		})

		// 🔹 Konsol Servisi (/console)
		http.HandleFunc("/viewmetrics", func(w http.ResponseWriter, r *http.Request) {
			// 1. HTML dosyasını diskten oku
			htmlContent, err := os.ReadFile("metric.view.console.html")

			if err != nil {
				// Dosya bulunamazsa veya okuma hatası olursa 500 hatası döndür
				http.Error(w, fmt.Sprintf("Console file is not found: %s. check file path.", "metric.view.console.html"), http.StatusInternalServerError)
				CustomLog(LevelError, "Console file reading error: %v", err)
				return
			}

			// 2. Başlıkları ayarla ve içeriği yaz
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(htmlContent)
		})

		http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {

			// 1. Headers'ı Ayarla
			w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

			// 2. Kilitli Bölgeyi Hızlıca Geç: Tüm metrik stringini topla
			//metricsPayload := collectPrometheusMetrics(m) // collectPrometheusMetrics kilitli bölgeyi yönetir.

			// 3. Kilitsiz Bölge: Toplu I/O işlemini gerçekleştir
			//if _, err := w.Write([]byte(metricsPayload)); err != nil {
			//	CustomLog(LevelError, "[PROMETHEUS] Yanıt yazılırken hata oluştu: %v", err)
			//}
		})

		CustomLog(LevelInfo, "[HTTP] Dinleniyor :%d (Metrics)", AppConfig.HttpPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", AppConfig.HttpPort), nil); err != nil && err != http.ErrServerClosed {
			CustomLog(LevelError, "Http server (port : %d) error: %v", AppConfig.HttpPort, err)
		}
	}()
}
