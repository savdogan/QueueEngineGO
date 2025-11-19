package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/CyCoreSystems/ari/v6"
	"github.com/CyCoreSystems/ari/v6/ext/play"
)

// ÖNEMLİ VARSAYIM: Bu fonksiyon, agentChannelID'yi kullanarak Call nesnesini bulmalıdır.
// Bu, CallManager'a yeni bir helper fonksiyonu eklenmesini gerektirir.
// Ancak, handler yapınızı basitleştirmek için, bu fonksiyonun OnOutboundChannelEnter
// çağrısı sırasında uygun Call nesnesini bulduğunu varsayıyoruz.

// OnOutboundChannelEnter, temsilci kanalı stasis'e girdiğinde çağrılır.
func OnOutboundChannelEnter(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle, connectionId string) {

	agentChannelID := message.Channel.ID

	// 1. İlgili Call Nesnesini Bulma (Bu bir TO DO/PlaceHolder'dır)
	// NOTE: Bu, StasisStart mesajının Application'ına göre CallManager'da bir arama gerektirir.
	// Örnek: CallManager'daki tüm çağrıların CurrentDistributionAttempts haritasını tarayarak
	// bu agentChannelID'ye sahip Call'u bulmalıyız.

	splittedAppData := strings.Split(message.Channel.Dialplan.AppData, ",")
	call_UniqueuId := splittedAppData[1]

	// Şimdilik, sadece call.UniqueId'yi kullanarak çağrıyı bulduğunu varsayıyoruz.
	call, found := globalCallManager.GetCall(call_UniqueuId) // Channel.Name'in UniqueId olduğunu varsayalım
	if !found {
		CustomLog(LevelWarn, "[%s] Call not found for outbound channel enter: %s", agentChannelID, agentChannelID)
		return
	}

	// 2. Kilit Al ve Serbest Bırak

	// 3. Durum Kontrolü (Java: if (call.getState() == CALL_STATE.BRIDGED_WITH_AGENT))
	if call.State == CALL_STATE_BridgedWithAgent {
		CustomLog(LevelInfo, "[%s] Got new outbound channel enter for already bridged call: %s. Terminating the new channel...", call.UniqueId, agentChannelID)

		// TO DO: ariFacade.hangup(call, channelId); eşleniği
		// Hata kontrolü eklenmelidir.
		_ = h.Hangup()
		return
	}

	// 4. Dağıtım Denemesini Bulma (Java: call.getCurrentDistributionAttempts().get(channelId);)
	attempt, ok := call.CurrentDistributionAttempts[agentChannelID]
	if !ok {
		CustomLog(LevelWarn, "[%s] Got unknown outbound channel enter (No active attempt found): %s", call.UniqueId, agentChannelID)
		// TO DO: Eğer bu bir hata ise, h.Hangup() da gerekebilir.
		return
	}

	// 5. Başarılı Atama ve Durum Güncelleme
	CustomLog(LevelInfo, "[%s] Bridging client with agent channel %s", call.UniqueId, agentChannelID)

	// Java: attempt.setDialStatus(DIAL_STATUS.ANSWER);
	attempt.DialStatus = DIAL_STATUS_Answer

	// Java: call.setState(CALL_STATE.BRIDGED_WITH_AGENT);
	call.State = CALL_STATE_BridgedWithAgent

	// Java: call.setConnectedAgentId(attempt.getUserId());
	call.ConnectedAgentId = attempt.UserID

	// Haritadaki struct'ı güncelle (Go map'leri struct'ları kopyalar)
	call.CurrentDistributionAttempts[agentChannelID] = attempt

	// 6. Harici Sistemleri Bilgilendirme (TO DO'lar)

	// queueLogger.onConnectAttemptSuccess(attempt);
	// TO DO: globalQueueLogger.OnConnectAttemptSuccess(attempt)

	// aidFacade.publishDistributionAttemptResult(...)
	// TO DO: globalAidFacade.PublishDistributionAttemptResult(call.ParentId, AID_DISTRIBUTION_STATE_Accepted, attempt.Username, call.QueueName)

	// callEventPublisher.publishCallQueueLeaveEvent(call);
	// TO DO: globalCallEventPublisher.PublishCallQueueLeaveEvent(call)

	// 7. Mevcut Zamanlanmış Eylemleri İptal Etme (cancelCurrentActions())
	globalScheduler.CancelByCallID(call.UniqueId)
	CustomLog(LevelDebug, "[%s] All scheduled announcements cancelled.", call.UniqueId)

	// 8. Köprüleme İşlemi (bridgeCall(channelId))
	// CallManager metodu olarak çağrılır.
	if err := globalCallManager.bridgeCall(call, agentChannelID); err != nil {
		CustomLog(LevelError, "[%s] Failed to bridge call: %v", call.UniqueId, err)
		return
	}

	// 9. Latecomer'ları Düşürme
	// if (call.getCurrentDistributionAttempts().size() > 1) { dropDistributionLatecomers(channelId); }
	/*	if len(call.CurrentDistributionAttempts) > 1 {
		globalCallManager.dropDistributionLatecomers(call, agentChannelID)
	} */
}

/*
// dropDistributionLatecomers, OnOutboundChannelEnter içinden Lock altında çağrılır.
func (cm *CallManager) dropDistributionLatecomers(call *Call, answeredChannelID string) {
	// Burada kilit zaten OnOutboundChannelEnter tarafından tutuluyor olmalı.

	// Temsilcinin ARI istemcisine erişmek için ConnectionId kullanılır.
	ariClient, found := globalClientManager.GetClient(call.ConnectionName)
	if !found {
		CustomLog(LevelError, "[%s] Cannot drop latecomers: ARI Client not found.", call.UniqueId)
		return
	}

	for currentChannelID := range call.CurrentDistributionAttempts {
		if currentChannelID != answeredChannelID {
			CustomLog(LevelDebug, "[%s] Dropping latecomer channel: %s", call.UniqueId, currentChannelID)

			// Kanala erişim ve kapatma
			latecomerChannelHandle := ariClient.Channel().Get(currentChannelID)

			if err := latecomerChannelHandle.Hangup(); err != nil {
				CustomLog(LevelError, "[%s] Failed to hangup latecomer %s: %v", call.UniqueId, currentChannelID, err)
				// Hata olsa bile devam et.
			}

			// Haritadan silme (Zaten Lock altındayız)
			delete(call.CurrentDistributionAttempts, currentChannelID)
		}
	}
} */

const BRIDGE_PREFIX = "queue-" // Varsayımsal köprü ön eki

// bridgeCall, çağrı kanallarını yeni bir köprüye ekler ve değişkenleri ayarlar.
// Bu fonksiyon, kendisini çağıran (OnOutboundChannelEnter) fonksiyonun zaten
// call nesnesi üzerinde yazma kilidi tuttuğunu varsayar.
func (cm *CallManager) bridgeCall(call *Call, agentChannelID string) error {

	// Kilit altındaki verileri oku ve değiştir.
	bridgeID := BRIDGE_PREFIX + call.UniqueId
	call.Bridge = bridgeID
	call.BridgedChannel = agentChannelID

	// Gerekli değerleri kopyala/oku (Kilit hala tutuluyor)
	clientConnectionID := call.ConnectionName
	clientChannelID := call.UniqueId // Genellikle UniqueId, istemci kanal ID'si olarak kullanılır
	connectedAgentIdStr := fmt.Sprintf("%d", call.ConnectedAgentId)

	// 1. Redis Güncelleme
	// redisCallAdapter.updateBaseData(call); eşleniği (TO DO: globalRedisCallAdapter'ı kullan)
	// globalRedisCallAdapter.UpdateBaseData(call)

	// 2. ARI Client'ı Bulma
	ariClient, found := globalClientManager.GetClient(clientConnectionID)
	if !found {
		CustomLog(LevelError, "[%s] ARI Client not found for connection ID: %s", call.UniqueId, clientConnectionID)
		return fmt.Errorf("ari client not found for connection id: %s", clientConnectionID)
	}

	CustomLog(LevelInfo, "Ari Client %+v", ariClient)
	CustomLog(LevelInfo, "Ari Client Bridge  %+v", ariClient.Bridge())
	CustomLog(LevelInfo, "Bridge ID :%s", bridgeID)

	bridgeKey := ari.NewKey(ari.BridgeKey, bridgeID)

	// 3. Köprü Oluşturma (ari.Client.Bridge().Create)
	bridge, err := ariClient.Bridge().Create(bridgeKey, "mixing", bridgeKey.ID)
	if err != nil {
		CustomLog(LevelError, "[%s] Failed to create bridge %s: %v", call.UniqueId, bridgeID, err)
		return err
	}

	// 4. Kanalları Köprüye Ekleme (ari.Client.Channel().Get(id).AddToBridge)

	// a. İstemci Kanalını Köprüye Ekle
	clientChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, clientChannelID))

	if err := bridge.AddChannel(clientChannelHandle.Key().ID); err != nil {
		CustomLog(LevelInfo, "failed to add channel to bridge , error : %+v", err)
		return err
	}

	// Silinecek
	//	if err := clientChannelHandle.Move(bridgeID, ""); err != nil {
	//		CustomLog(LevelError, "[%s] Failed to add client channel %s to bridge: %v", call.UniqueId, clientChannelID, err)
	//		return err
	//	}

	// b. Temsilci Kanalını Köprüye Ekle
	agentChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, agentChannelID))

	if err := bridge.AddChannel(agentChannelHandle.Key().ID); err != nil {
		CustomLog(LevelInfo, "failed to add channel to bridge agent channel , error : %+v", err)
		return err
	}

	//if err := agentChannelHandle.Move(bridgeID, ""); err != nil {
	//	CustomLog(LevelError, "[%s] Failed to add agent channel %s to bridge: %v", call.UniqueId, agentChannelID, err)
	//	return err
	//}

	// 5. Kanal Değişkenini Ayarlama (ari.Client.Channel().Get(id).SetVariable)
	variableName := string(DIALPLAN_VARIABLE_QeQueueMember)

	// Değişkeni istemci kanalına ayarla
	if err := clientChannelHandle.SetVariable(variableName, connectedAgentIdStr); err != nil {
		CustomLog(LevelError, "[%s] Failed to set variable %s to %s: %v", call.UniqueId, variableName, connectedAgentIdStr, err)
		return err
	}

	CustomLog(LevelInfo, "[%s] Call successfully bridged: %s", call.UniqueId, bridgeID)

	wg := new(sync.WaitGroup)

	wg.Add(1)

	go manageBridge(*redisClientManager.ctx, bridge, wg)

	wg.Wait()

	return nil // Başarılı
}

func manageBridge(ctx context.Context, h *ari.BridgeHandle, wg *sync.WaitGroup) {
	// Delete the bridge when we exit
	defer h.Delete() //nolint:errcheck

	destroySub := h.Subscribe(ari.Events.BridgeDestroyed)
	defer destroySub.Cancel()

	enterSub := h.Subscribe(ari.Events.ChannelEnteredBridge)
	defer enterSub.Cancel()

	leaveSub := h.Subscribe(ari.Events.ChannelLeftBridge)
	defer leaveSub.Cancel()

	wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-destroySub.Events():
			CustomLog(LevelDebug, "bridge destroyed")
			return
		case e, ok := <-enterSub.Events():
			if !ok {
				CustomLog(LevelError, "channel entered subscription closed")
				return
			}

			v := e.(*ari.ChannelEnteredBridge)

			CustomLog(LevelDebug, "channel entered bridge %+v", v.Channel.Name)

			go func() {
				if err := play.Play(ctx, h, play.URI("sound:confbridge-join")).Err(); err != nil {
					CustomLog(LevelError, "failed to play join sound %s", err)
				}
			}()
		case e, ok := <-leaveSub.Events():
			if !ok {
				CustomLog(LevelError, "channel left subscription closed")
				return
			}

			v := e.(*ari.ChannelLeftBridge)

			CustomLog(LevelDebug, "channel left bridge , %s", v.Channel.Name)

			go func() {
				if err := play.Play(ctx, h, play.URI("sound:confbridge-leave")).Err(); err != nil {
					CustomLog(LevelError, "failed to play leave sound %+v", err)
				}
			}()
		}
	}
}

// buildAgentUri: Temsilcinin URI'sini oluşturur (Örnek)
// Java'daki User sınıfının karşılığı olan UserDist struct'ını kullanır.
func buildAgentUri(user UserDist) string {
	// Java mantığına göre bir URI oluşturur (Örn: "PJSIP/John@trunk")
	return fmt.Sprintf("SIP/%s-%d@${REGAGENT%s}", user.Username, user.ID, user.Username)
}

// getCdrVariableName: CDR değişken adını alır (Örnek)
// Java'da Enum'u string'e çeviriyor.
func getCdrVariableName(variable DIALPLAN_VARIABLE) string {
	return fmt.Sprintf("CDR(%s)", variable) // Varsayılan CDR ön eki
}

// addSipHeaderVariable: SIP başlık değişkenini variables map'ine ekler.
func addSipHeaderVariable(variables map[string]string, header SIPHeader, value string) {
	// SIP başlıkları genellikle 'SIPADDHEADER' formatında değişken olarak ayarlanır.
	// Örneğin: SIPADDHEADER(X-Header-Name)=Value
	variableName := fmt.Sprintf("__SIPADDHEADER(%s)", header.SequenceValue)
	variables[variableName] = value
}

// OriginateAgentChannel, tüm değişkenleri ayarlayarak temsilciye yönelik çağrıyı başlatır.
// Java'daki Mono<Call> yerine, temel başlatma fonksiyonundan dönen hatayı döndürür.
func OriginateAgentChannel(call *Call, user UserDist, dialTimeout int, newChannelsUniqueId string) error {

	// 1. Gerekli tüm değişkenleri tutacak map'i başlat
	variables := make(map[string]string)

	// --- Kilitlenme ile gerekli alanları oku (Call'un eşzamanlı okunması) ---
	call.mu.RLock()
	call_UniqueId := call.UniqueId
	attemptNumber := call.DistributionAttemptNumber
	queueName := call.QueueName
	callDialContext := call.DialContext
	call_OutBoundApplicationName := call.OutBoundApplicationName
	call.mu.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(queueName)
	if err != nil {
		errMsg := fmt.Sprintf("Queue not found: %s", queueName)
		CustomLog(LevelError, "[%s] %s %+v", call_UniqueId, errMsg, err)
		return fmt.Errorf("Queue Error: %s", errMsg)
	}

	currentQueue.mu.RLock()
	queueID := queueName // Queue tipi string olduğu için burada hata olabilir. String olarak varsayıyorum.
	musicClassOnHold := currentQueue.MusicClassOnHold
	agentAnnounceFile := currentQueue.Announce // Varsayımsal alan
	currentQueue.mu.RUnlock()

	// 2. Zorunlu Değişkenleri Ekle

	// DIALPLAN_VARIABLE.dialUri
	variables[string(DIALPLAN_VARIABLE_DialUri)] = buildAgentUri(user)

	// DIALPLAN_VARIABLE.dialTimeout
	variables[string(DIALPLAN_VARIABLE_DialTimeout)] = fmt.Sprintf("%d", dialTimeout)

	// CDR değişkeni (call_type)
	variables[getCdrVariableName(DIALPLAN_VARIABLE_CallType)] = AGENT_LEG_CALL_TYPE

	// 3. SIP Başlıklarını Ekle

	// Dağıtım Girişimi Sayısı (DISTRIBUTION_ATTEMPT_NUMBER)
	addSipHeaderVariable(variables, SIP_HEADERS["DISTRIBUTION_ATTEMPT_NUMBER"], fmt.Sprintf("%d", attemptNumber))

	// Kuyruk Adı (QUEUE_NAME)
	addSipHeaderVariable(variables, SIP_HEADERS["QUEUE_NAME"], queueName)

	// Kuyruk ID (QUEUE_ID)
	addSipHeaderVariable(variables, SIP_HEADERS["QUEUE_ID"], queueID)

	// MOH Sınıfı (HOLD_MOH_CLASS)
	if strings.TrimSpace(musicClassOnHold) != "" {
		addSipHeaderVariable(variables, SIP_HEADERS["HOLD_MOH_CLASS"], musicClassOnHold)
	}

	// 4. Temsilci Anonsu (Agent Announce)
	if strings.TrimSpace(agentAnnounceFile) != "" {
		variables[string(DIALPLAN_VARIABLE_AgentAnnounce)] = agentAnnounceFile
	}

	// 5. Dial Context Belirleme
	// (StringUtils.isNotBlank(call.getDialContext())) ? call.getDialContext() : DIALPLAN_CONTEXT.qe_dial_agent.toString();

	dialContext := string(DIALPLAN_CONTEXT_QeDialAgent) // Varsayılan değer
	if strings.TrimSpace(callDialContext) != "" {
		dialContext = callDialContext
	}

	// 6. Başlatma Hedefini Oluşturma
	endpoint := fmt.Sprintf("Local/s@%s", dialContext)

	// 7. Temel Outbound Fonksiyonunu Çağırma
	// return originateOutboundChannel(...)

	outBoundChannel, err := originateOutboundChannel(call,
		endpoint,
		newChannelsUniqueId,
		variables,
		call_OutBoundApplicationName) // AriFacade'ın outboundAppName alanını kullanır

	if err != nil {
		CustomLog(LevelError, "[%s] Originate failed via context %s: %v", call.UniqueId, dialContext, err)
	}

	CustomLog(LevelInfo, "[%s] Outbound channel originated: %s", call.UniqueId, outBoundChannel.ID())

	return err
}

// originateOutboundChannel, belirtilen hedef (endpoint) için yeni bir kanal başlatır.
// Java'daki Mono<Call> yerine, başlatılan kanalın handle'ını ve hata (error) döndürür.
func originateOutboundChannel(call *Call, endpoint string, newChannelsUniqueId string, variables map[string]string, stasisApp string) (*ari.ChannelHandle, error) {

	// 1. ARI İstemcisini Bulma
	ariClient, found := globalClientManager.GetClient(call.ConnectionName)
	if !found {
		errMsg := fmt.Sprintf("ARI Client not found for Connection ID: %s", call.ConnectionName)
		// Java'daki onError("Could not originate...") eşleniği
		onError("Could not originate agent channel", call.UniqueId, fmt.Errorf("%s", errMsg))
		return nil, fmt.Errorf("%s", errMsg)
	}

	// 2. ARI Originate Request (Kanalları başlatma)
	// Java'daki builder zincirini (originateWithId().setApp().setVariables()...) tek bir struct'a çeviriyoruz.
	//	request := ari.ChannelCreateRequest{
	//		Endpoint:   endpoint,
	//		App:        stasisApp,
	//		AppArgs:    call.UniqueId,       // Java'daki .setAppArgs(call.getUniqueId())
	//		ChannelID:  newChannelsUniqueId, // .originateWithId(newChannelsUniqueId)
	//		Originator: call.UniqueId,       // .setOriginator(call.getUniqueId())
	//Variables:  variables,

	// Java'daki .setTimeout(CALL_MAX_TTL_SEC) -> Go'da sadece ARI'ya bildirilir.
	// Go ARI kütüphanesi doğrudan bir Timeout alanı sağlamaz,
	// bu genellikle değişkenler veya dialplan üzerinden yönetilir.
	// Ancak bu TTL, ARI'nin kendisi tarafından yönetilen bir durumdur ve burada doğrudan tanımlanmaz.
	//	}

	orgRequest := ari.OriginateRequest{
		Endpoint:   endpoint,
		App:        stasisApp,
		AppArgs:    call.UniqueId,       // Java'daki .setAppArgs(call.getUniqueId())
		ChannelID:  newChannelsUniqueId, // .originateWithId(newChannelsUniqueId)
		Originator: call.UniqueId,
		Timeout:    -1,
		Variables:  variables,
	}

	//	newAriChannel := ariClient.Channel()

	//	for key, value := range variables {
	//		variableKey := ari.NewKey(ari.VariableKey, key)
	//	newAriChannel.SetVariable(variableKey, key, value)
	//}

	// ARI Kütüphanesi Create() metodunu kullanır.
	ch, err := ariClient.Channel().Originate(nil, orgRequest)

	// 3. Hata Kontrolü (Java'daki .doOnError eşleniği)
	if err != nil {
		// Java'daki onError("Could not originate...") eşleniği
		onError("Could not originate outbound channel", call.UniqueId, err)
		return nil, fmt.Errorf("originate failed: %w", err)
	}

	// 4. Başarılı Loglama (Java'daki .doOnNext eşleniği)
	CustomLog(LevelInfo, "[%s] Originated channel: %s", call.UniqueId, newChannelsUniqueId)

	return ch, nil
}

// --- EKLENMESİ GEREKEN YARDIMCI FONKSİYONLAR ---

// onError: Java'daki onError metodunun Go eşleniği (sadece loglama ve hata yönetimi yapar)
func onError(message string, uniqueId string, err error) {
	level := LevelError

	CustomLog(level, "[%s] %s. Error: %v", uniqueId, message, err)

	// Bu noktada CallManager'dan çağrıyı sonlandırmak (TerminateCall) gerekebilir.
}

// handleStasisStartMessage, Java kodunun Go dilindeki karşılığıdır.

func handleStasisStartMessage(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {

	CustomLog(LevelInfo, "[%s] StasisStart (connectionId : %s, application: %s, channel: %v, args: %v)", ARI_MESSAGE_LOG_PREFIX, ariAppInfo.ConnectionName, message.Application, message.Channel, message.Args)

	if message.Channel.ID == "" {
		log.Printf("ERROR: StasisStart message has no channel: %v", message)
		return // Kanal yoksa işlemi durdur
	}

	// Args kontrolü. Go'da boş string dizisi kontrolü: len(message.Args) == 0
	if len(message.Args) == 0 {
		log.Printf("ERROR: StasisStart message has no arguments: %v", message)
		return // Argüman yoksa işlemi durdur
	}

	// Client Uygulamasına Giriş
	if isInboundApplication(message.Application) {
		OnClientChannelEnter(message, cl, h, ariAppInfo)
	} else if isOutboundApplication(message.Application) {
		OnOutboundChannelEnter(message, cl, h, ariAppInfo.ConnectionName)
	} else {
		log.Printf("ERROR: Got StasisStart for unknown ARI application: %s", message.Application)
	}
}

// Not: Bu kodun çalışması için, ari.Channel gibi tipleri
// kullanılan Go ARI kütüphanesine göre doğru bir şekilde ayarlamanız gerekir.

func OnClientChannelEnter(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {
	CustomLog(LevelInfo, "Inbound kanalından giriş yapıldı..")

	call := CreateCall(message, message.Channel.ID, message.Channel.Name, message.Args[0], 1)

	//To DO:  Önce ARI bağlantısını kontrol et ,  uygun değilse çağrıyı reddet

	//To DO: Call boş mu

	if call == nil {

		//To DO: Kanalı reject et ...

		CustomLog(LevelInfo, "Call is empty : %s", message.Channel.ID)
		return
	}

	key := h.Key()

	if key == nil {
		CustomLog(LevelError, "Channel Key is nil for Channel ID : %s ", message.Channel.ID)
		return
	}

	call.ChannelKey = key
	call.Application = message.Application
	call.ConnectionName = ariAppInfo.ConnectionName
	call.OutBoundApplicationName = ariAppInfo.OutboundAppName
	call.InstanceID = ariAppInfo.InstanceID

	//To DO: AID bağlantısını kontrol et  , uygun değilse çağrıyı reddet

	//To DO: Redis Bağlatısını kontrol et  , uygun değilse çağrıyı reddet

	CustomLog(LevelInfo, "[CALL_CREATED] %+v", call)
	go globalCallManager.AddCall(call)
}
