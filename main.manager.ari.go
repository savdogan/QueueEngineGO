package main

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/CyCoreSystems/ari/v6"
	"github.com/CyCoreSystems/ari/v6/client/native"
)

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]ari.Client),
	}
}

func (cm *ClientManager) AddClient(connectiondId string, cl ari.Client) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clients[connectiondId] = cl
}

func (cm *ClientManager) GetClient(connectiondId string) (ari.Client, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	cl, ok := cm.clients[connectiondId]
	return cl, ok
}

// runApp, tek bir ARI bağlantısını kurar ve olayları dinler.
func runApp(ctx context.Context, cfg *AriConfig, manager *ClientManager, ariAppInfo AriAppInfo) error {
	CustomLog(LevelInfo, "Connecting to ARI: %s - %s - %s -%t", ariAppInfo.ConnectionName, ariAppInfo.InboundAppName, ariAppInfo.OutboundAppName, ariAppInfo.IsOutboundApplication)

	ariConnectionApplicationName := ""
	if ariAppInfo.IsOutboundApplication {
		ariConnectionApplicationName = ariAppInfo.OutboundAppName
	} else {
		ariConnectionApplicationName = ariAppInfo.InboundAppName
	}

	ariSlogLogger := NewSlogLogger(ariConnectionApplicationName)

	options := &native.Options{
		Application:  ariConnectionApplicationName,
		Logger:       ariSlogLogger, // Standart log kullanıldığı için nil bırakılabilir
		Username:     cfg.Username,
		Password:     cfg.Password,
		URL:          cfg.RestURL,
		WebsocketURL: cfg.WebsocketURL,
	}

	DebugARIInfo(options.URL, options.Username, options.Password)

	// native.Connect ile ARI bağlantısı kurulur
	cl, err := native.Connect(options)

	if err != nil {
		CustomLog(LevelError, "Native.Connect Error %+v", err)
		return err
	}

	asteriskInfo, err := cl.Asterisk().Info(nil) // nil: varsayılan seçenekler
	if err != nil {
		CustomLog(LevelError, "Error %+v , AsterixInfo : %+v", err, asteriskInfo)
		return err
	}

	// İstemciyi Yöneticiye Kaydet
	manager.AddClient(ariAppInfo.ConnectionName, cl)
	CustomLog(LevelInfo, "Client registered and listening: %s", ariAppInfo.ConnectionName)

	// Olay dinlemesini başlat (bloklamaz)
	go listenApp(ctx, cl, ariAppInfo)

	return nil
}

func DebugARIInfo(url, user, pass string) {
	req, _ := http.NewRequest("GET", url+"/asterisk/info", nil)
	req.SetBasicAuth(user, pass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("HTTP error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Println("Status:", resp.Status)
	log.Println("Raw Body:", string(body)) // 🔥 HTML ise burada görünecek
}

// handleAriEvent, gelen tüm ARI Event arayüzlerini işler. (Java'daki onSuccess eşleniği)
func handleAriEvent(msg ari.Event, cl ari.Client, ariAppInfo AriAppInfo) {

	CustomLog(LevelInfo, "Ari Event : %+v", msg)

	appName := msg.GetApplication()

	// Channel verisini güvenli bir şekilde çekme (GetChannel() metodu olan event'ler için)
	var channelID string
	if chGetter, ok := msg.(ChannelGetter); ok {
		if chData := chGetter.GetChannel(); chData != nil {
			channelID = chData.ID
		}
	}

	// switch type ile olayın tipine göre işlem yapma
	switch v := msg.(type) {

	case *ari.StasisStart:
		CustomLog(LevelInfo, "[%s] StasisStart: Channel %s entered. Args: %v", appName, channelID, v.Args)
		// Kanala özgü işleyiciyi başlat
		h := cl.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID))
		go handleStasisStartMessage(msg.(*ari.StasisStart), cl, h, ariAppInfo)
	case *ari.StasisEnd:
		CustomLog(LevelInfo, "[%s] StasisEnd: Channel %s left.", appName, channelID)
		h := cl.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID))
		go handleStasisEndMessage(msg.(*ari.StasisEnd), cl, h, ariAppInfo)
	case *ari.ChannelEnteredBridge:
		CustomLog(LevelDebug, "[%s] ChannelEnteredBridge: Channel %s joined bridge %s", appName, channelID, v.Bridge.ID)

	case *ari.ChannelLeftBridge:
		CustomLog(LevelDebug, "[%s] ChannelLeftBridge: Channel %s left bridge %s", appName, channelID, v.Bridge.ID)

	case *ari.PlaybackStarted:
		CustomLog(LevelDebug, "[%s] PlaybackStarted: Playback %s started on %s", appName, v.Playback.ID, channelID)

	case *ari.PlaybackFinished:
		CustomLog(LevelDebug, "[%s] PlaybackFinished: Playback %s finished on %s", appName, v.Playback.ID, channelID)

	case *ari.ChannelDtmfReceived:
		CustomLog(LevelInfo, "[%s] ChannelDtmfReceived: Channel %s, Digit: %s", appName, channelID, v.Digit)

	case *ari.Dial:
		peerID := ""
		// Kontrol: Peer'ın ID alanı boş string değilse, Peer var demektir.
		// Yapı (struct) olduğu için nil kontrolü yapılmaz.
		if v.Peer.ID != "" {
			peerID = v.Peer.ID
		}
		CustomLog(LevelInfo, "[%s] Dial: Status: %s, Peer: %s", appName, v.Dialstatus, peerID)
		go handleDialMessage(msg.(*ari.Dial))

	case *ari.ChannelVarset:
		CustomLog(LevelTrace, "[%s] ChannelVarset: Channel %s, Var: %s, Value: %s", appName, channelID, v.Variable, v.Value)

	case *ari.ChannelStateChange:
		CustomLog(LevelDebug, "[%s] ChannelStateChange: Channel %s is now %s", appName, v.Channel.ID, v.Channel.State)

	case *ari.ChannelDialplan:
		CustomLog(LevelTrace, "[%s] ChannelDialplan: Channel %s entered app %s", appName, channelID, v.DialplanApp)

	default:
		CustomLog(LevelTrace, "[%s] Unhandled Event Type: %s", appName, msg.GetType())
	}
}

// listenApp, tüm ARI olaylarını dinleyen sonsuz döngüyü çalıştırır.
func listenApp(ctx context.Context, cl ari.Client, ariAppInfo AriAppInfo) {

	CustomLog(LevelInfo, "Listen App %s , connectionId : %s", cl.ApplicationName(), ariAppInfo.ConnectionName)

	allEvents := cl.Bus().Subscribe(nil, "StasisStart", "StasisEnd", "ChannelEnteredBridge", "ChannelLeftBridge", "PlaybackStarted", "PlaybackFinished", "ChannelVarset", "ChannelStateChange", "ChannelDialplan", "ChannelDtmfReceived", "Dial")
	defer allEvents.Cancel()

	for {

		select {
		case e := <-allEvents.Events():
			if e == nil {
				CustomLog(LevelTrace, "Event boş")
				continue
			}

			// Her olayı ayrı bir goroutine'de işle
			go handleAriEvent(e, cl, ariAppInfo)

		case <-ctx.Done():
			CustomLog(LevelInfo, "Listener shutting down...")
			return
		}
	}
}

// makeCall, ARI üzerinden bir kanal oluşturur (çağrı başlatır).
func makeCall(cl ari.Client) (h *ari.ChannelHandle, err error) {
	h, err = cl.Channel().Create(nil, ari.ChannelCreateRequest{
		Endpoint: "Local/1000",
		App:      "example", // Bu, Asterisk'teki Stasis uygulamasına yönlendirir
	})
	return
}

// channelHandler, Stasis'e giren her bir kanal için tetiklenir ve o kanalı yönetir.
/*
func channelHandler(cl ari.Client, h *ari.ChannelHandle) {

	CustomLog(LevelInfo, "Running channel handler for channel %s , appname : %s", h.ID(), cl.ApplicationName())

	stateChange := h.Subscribe(ari.Events.ChannelStateChange)
	defer stateChange.Cancel()

	// Basit bir örnek: Kanalı cevapla ve bir süre bekle/işle (Bu kısım özelleştirilebilir)
	h.Answer() //nolint:errcheck

	// Genellikle burada DTMF alımı, medya oynatma gibi ARI işlemleri yapılır
	// Şimdilik sadece olayları işleyip hangup'ı bekleyelim
	for range stateChange.Events() {
		// Kanala ait olayları burada işleyin
	}

	// İşlem bittiğinde kapat
	h.Hangup() //nolint:errcheck
	CustomLog(LevelInfo, "Channel %s hung up.", h.ID())
}
*/

/*

func startHeartbeatMonitor(client ari.Client) {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			// 30 saniyede bir Asterisk'ten basit bir bilgi iste
			_, err := client.Asterisk().Info(nil)

			if err != nil {
				// Hata varsa, bağlantı büyük ihtimalle koptu
				CustomLog(LevelError, "ARI Heartbeat failed. Reconnecting...")

				// Yeniden bağlanma mantığınızı burada çağırın
				newClient, reconnErr := ReconnectAri(client)
				if reconnErr == nil {
					// Yeniden bağlandı, global istemciyi güncelleyin ve bu Goroutine'den çıkın.
					// (veya yeni istemciyi dinleyecek yeni bir Goroutine başlatın)
					globalClientManager.ReplaceClient(newClient)
					return
				}
				CustomLog(LevelError, "Failed to reconnect: %v. Retrying...", reconnErr)
			}
			// ... Uygulama kapatıldığında çıkış sinyali de buraya eklenebilir
		}
	}
}

// Tahmini Yeniden Bağlanma Fonksiyonu
func ReconnectAri(oldClient ari.Client) (ari.Client, error) {
    // 1. Eski bağlantıyı temizle
    if oldClient != nil {
        oldClient.Close() // veya Shutdown()
    }

    // 2. Yeni bağlantıyı kur (main.InitAriConnection içindeki mantığı kullan)
    newClient, err := ari.Connect(...) // Doğru parametrelerle yeniden bağlantı
    if err != nil {
        return nil, err
    }
    return newClient, nil
}

*/
