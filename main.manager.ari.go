package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/CyCoreSystems/ari/v6"
	"github.com/CyCoreSystems/ari/v6/client/native"
)

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]WrapedAriClient),
	}
}

func (cm *ClientManager) AddClient(connectiondId string, cancelFunc context.CancelFunc, cl ari.Client) {
	cm.Lock()
	defer cm.Unlock()
	cm.clients[connectiondId] = WrapedAriClient{
		client:     &cl,
		cancelFunc: cancelFunc,
	}
}

func (cm *ClientManager) GetClient(connectiondId string) (ari.Client, bool) {
	cm.RLock()
	defer cm.RUnlock()
	cl, ok := cm.clients[connectiondId]
	return *cl.client, ok
}

func (cm *ClientManager) GetWrapedAriClient(connectiondId string) (WrapedAriClient, bool) {
	cm.RLock()
	defer cm.RUnlock()
	wal, ok := cm.clients[connectiondId]
	return wal, ok
}

func (cm *ClientManager) RemoveClient(connectiondId string) {
	cm.Lock()
	defer cm.Unlock()
	delete(cm.clients, connectiondId)
}

// runApp, tek bir ARI bağlantısını kurar ve olayları dinler.
func runApp(ctx context.Context, cfg *AriConfig, manager *ClientManager, ariAppInfo AriAppInfo) error {
	clog(LevelInfo, "Connecting to ARI: %s - %s - %s -%t", ariAppInfo.ConnectionName, ariAppInfo.InboundAppName, ariAppInfo.OutboundAppName, ariAppInfo.IsOutboundApplication)

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
		clog(LevelError, "Native.Connect Error %+v", err)
		return err
	}

	asteriskInfo, err := cl.Asterisk().Info(nil) // nil: varsayılan seçenekler
	if err != nil {
		clog(LevelError, "Error %+v , AsterixInfo : %+v", err, asteriskInfo)
		return err
	}

	customExitCtx, cancelCustom := context.WithCancel(context.Background())
	// İstemciyi Yöneticiye Kaydet
	manager.AddClient(ariAppInfo.ConnectionName, cancelCustom, cl)
	clog(LevelInfo, "Client registered and listening: %s", ariAppInfo.ConnectionName)

	// Olay dinlemesini başlat (bloklamaz)
	go listenApp(ctx, customExitCtx, cl, ariAppInfo)

	return nil
}

func InitAriConnection() {

	clog(LevelInfo, "Ari connections are starting...")

	for _, server := range g.ServerManager.MediaServers {
		for _, instanceId := range g.Cfg.InstanceIDs {
			server.InitAriMediaServer(instanceId)
		}
	}
}

func (asm *WbpServer) InitAriMediaServer(instanceId string) {

	ariCfg := &AriConfig{
		Id:           asm.ID,
		WebsocketURL: fmt.Sprintf("ws://%s:8088/ari/events", asm.IP),
		Username:     *asm.AriUsername,
		RestURL:      fmt.Sprintf("%sari", *asm.AriURL),
		Password:     *asm.AriPassword,
	}

	appInbound := fmt.Sprintf("%s%s", INBOUND_ARI_APPLICATION_PREFIX, instanceId)
	appOutbound := fmt.Sprintf("%s%s", OUTBOUND_ARI_APPLICATION_PREFIX, instanceId)

	// 2. App Bilgilerini Oluştur (Struct Literal kullanarak daha okunaklı hale getirildi)
	inboundInfo := AriAppInfo{
		ConnectionName:        fmt.Sprintf("%d-%s-%s", asm.ID, appInbound, instanceId),
		InboundAppName:        appInbound,
		OutboundAppName:       appOutbound,
		IsOutboundApplication: false,
		InstanceID:            instanceId,
		MediaServerId:         asm.ID,
	}

	outboundInfo := AriAppInfo{
		ConnectionName:        fmt.Sprintf("%d-%s-%s", asm.ID, appOutbound, instanceId),
		InboundAppName:        appInbound,
		OutboundAppName:       appOutbound,
		IsOutboundApplication: true,
		InstanceID:            instanceId,
		MediaServerId:         asm.ID,
	}

	// 3. Uygulamaları Başlat (Tekrarlayan kod Helper fonksiyona taşındı)
	startAriApp(g.Ctx, *ariCfg, inboundInfo)
	startAriApp(g.Ctx, *ariCfg, outboundInfo)

}

// startAriApp: Goroutine başlatma mantığını ve hata loglamayı tek bir yerde toplar.
func startAriApp(ctx context.Context, aricfg AriConfig, info AriAppInfo) {
	go func() {
		// runApp fonksiyonuna g.ACM'i buradan parametre olarak geçiyoruz
		if err := runApp(ctx, &aricfg, g.ACM, info); err != nil {
			clog(LevelError, "ARI application failed to start for Connection: %s, Instance: %s. Error: %v",
				info.ConnectionName, info.InstanceID, err)
		}
	}()
}

func (sm *ServerManager) addServer(serverId int64) error {

	_, found := g.ServerManager.MediaServers[serverId]

	newServer, err := getServer(g.DB, serverId)

	if err != nil {
		clog(LevelError, "Server not loaded, Server adding could not complete or server not enabled. Error : %+v", err)
		return err
	}

	if found {
		//find ari client
		for _, instanceId := range g.Cfg.InstanceIDs {

			appInbound := fmt.Sprintf("%s%s", INBOUND_ARI_APPLICATION_PREFIX, instanceId)
			appOutbound := fmt.Sprintf("%s%s", OUTBOUND_ARI_APPLICATION_PREFIX, instanceId)
			connectionNameInbound := fmt.Sprintf("%d-%s-%s", newServer.ID, appInbound, instanceId)
			connectionNameOutbound := fmt.Sprintf("%d-%s-%s", newServer.ID, appOutbound, instanceId)

			wali, foundi := g.ACM.GetWrapedAriClient(connectionNameInbound)
			if foundi {
				wali.cancelFunc()
				g.ACM.RemoveClient(connectionNameInbound)
			}

			walo, foundo := g.ACM.GetWrapedAriClient(connectionNameOutbound)
			if foundo {
				walo.cancelFunc()
				g.ACM.RemoveClient(connectionNameOutbound)
			}
		}

	} else {
		g.ServerManager.AddMediaServer(newServer)
	}

	for _, instanceId := range g.Cfg.InstanceIDs {
		newServer.InitAriMediaServer(instanceId)
	}

	return nil

}

func (sm *ServerManager) deleteServer(serverId int64) error {

	_, found := g.ServerManager.MediaServers[serverId]

	if found {
		//find ari client
		for _, instanceId := range g.Cfg.InstanceIDs {

			appInbound := fmt.Sprintf("%s%s", INBOUND_ARI_APPLICATION_PREFIX, instanceId)
			appOutbound := fmt.Sprintf("%s%s", OUTBOUND_ARI_APPLICATION_PREFIX, instanceId)
			connectionNameInbound := fmt.Sprintf("%d-%s-%s", serverId, appInbound, instanceId)
			connectionNameOutbound := fmt.Sprintf("%d-%s-%s", serverId, appOutbound, instanceId)

			wali, foundi := g.ACM.GetWrapedAriClient(connectionNameInbound)
			if foundi {
				wali.cancelFunc()
				g.ACM.RemoveClient(connectionNameInbound)
				clog(LevelDebug, "Ari client is stopped and deleted in server : %d , ari client connection name : %s", serverId, connectionNameInbound)
			}

			walo, foundo := g.ACM.GetWrapedAriClient(connectionNameOutbound)
			if foundo {
				walo.cancelFunc()
				g.ACM.RemoveClient(connectionNameOutbound)
				clog(LevelDebug, "Ari client is stopped and deleted in server : %d , ari client connection name : %s", serverId, connectionNameOutbound)
			}
		}

	} else {
		clog(LevelDebug, "Server that will be deleted is not found in cache , server id : %d", serverId)
	}

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

	clog(LevelTrace, "Ari Event : %+v", msg)

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
		clog(LevelInfo, "[StasisStart] %s ,appName %s,  Args: %v", channelID, appName, v.Args)
		// Kanala özgü işleyiciyi başlat
		h := cl.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID))
		go handleStasisStartMessage(msg.(*ari.StasisStart), h, ariAppInfo)
	case *ari.StasisEnd:
		clog(LevelInfo, "[StasisEnd]  %s, appName %s", channelID, appName)
		go handleStasisEndMessage(msg.(*ari.StasisEnd), ariAppInfo)
	case *ari.ChannelEnteredBridge:
		clog(LevelInfo, "[%s] ChannelEnteredBridge: Channel %s joined bridge %s", appName, channelID, v.Bridge.ID)

	case *ari.ChannelLeftBridge:
		clog(LevelInfo, "[%s] ChannelLeftBridge: Channel %s left bridge %s", appName, channelID, v.Bridge.ID)

	case *ari.PlaybackStarted:
		clog(LevelInfo, "[%s] PlaybackStarted: Playback %s started on %s", appName, v.Playback.ID, channelID)

	case *ari.PlaybackFinished:
		clog(LevelInfo, "[%s] PlaybackFinished: Playback %s finished on %s", appName, v.Playback.ID, channelID)

	case *ari.ChannelDtmfReceived:
		clog(LevelInfo, "[%s] ChannelDtmfReceived: Channel %s, Digit: %s", appName, channelID, v.Digit)

	case *ari.Dial:
		peerID := ""
		// Kontrol: Peer'ın ID alanı boş string değilse, Peer var demektir.
		// Yapı (struct) olduğu için nil kontrolü yapılmaz.
		if v.Peer.ID != "" {
			peerID = v.Peer.ID
		}
		clog(LevelInfo, "[%s] Dial: Status: %s, Peer: %s", appName, v.Dialstatus, peerID)
		go handleDialMessage(msg.(*ari.Dial))

	case *ari.ChannelVarset:
		clog(LevelTrace, "[%s] ChannelVarset: Channel %s, Var: %s, Value: %s", appName, channelID, v.Variable, v.Value)

	case *ari.ChannelStateChange:
		clog(LevelInfo, "[%s] ChannelStateChange: Channel %s is now %s", appName, v.Channel.ID, v.Channel.State)

	case *ari.ChannelDialplan:
		clog(LevelInfo, "[%s] ChannelDialplan: Channel %s entered app %s", appName, channelID, v.DialplanApp)

	default:
		clog(LevelInfo, "[%s] Unhandled Event Type: %s", appName, msg.GetType())
	}
}

// listenApp fonksiyonuna yeni bir 'customExitCtx' parametresi ekledik
func listenApp(ctx context.Context, customExitCtx context.Context, cl ari.Client, ariAppInfo AriAppInfo) {

	clog(LevelInfo, "Listen App %s , connectionId : %s", cl.ApplicationName(), ariAppInfo.ConnectionName)

	allEvents := cl.Bus().Subscribe(nil, "StasisStart", "StasisEnd", "ChannelEnteredBridge", "ChannelLeftBridge", "PlaybackStarted", "PlaybackFinished", "ChannelVarset", "ChannelStateChange", "ChannelDialplan", "ChannelDtmfReceived", "Dial")
	defer allEvents.Cancel()

	for {
		select {
		// 1. ARI Eventleri
		case e := <-allEvents.Events():
			if e == nil {
				clog(LevelTrace, "Event boş")
				continue
			}
			go handleAriEvent(e, cl, ariAppInfo)

		// 2. Uygulama Kapanışı (Genel Shutdown)
		case <-ctx.Done():
			clog(LevelInfo, "Listener shutting down (App Context)...")
			return

		// 3. YENİ EKLENEN: Özel Olay Çıkışı
		case <-customExitCtx.Done():
			clog(LevelInfo, "Listener shutting down (WebPhone event fired about the server)...")
			// Gerekirse burada temizlik işlemleri yapılabilir
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

	clog(LevelInfo, "Running channel handler for channel %s , appname : %s", h.ID(), cl.ApplicationName())

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
	clog(LevelInfo, "Channel %s hung up.", h.ID())
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
				clog(LevelError, "ARI Heartbeat failed. Reconnecting...")

				// Yeniden bağlanma mantığınızı burada çağırın
				newClient, reconnErr := ReconnectAri(client)
				if reconnErr == nil {
					// Yeniden bağlandı, global istemciyi güncelleyin ve bu Goroutine'den çıkın.
					// (veya yeni istemciyi dinleyecek yeni bir Goroutine başlatın)
					g.ACM.ReplaceClient(newClient)
					return
				}
				clog(LevelError, "Failed to reconnect: %v. Retrying...", reconnErr)
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
