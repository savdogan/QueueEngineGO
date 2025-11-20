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

func OnOutboundChannelLeave(message *ari.StasisEnd, cl ari.Client, h *ari.ChannelHandle, connectionId string) {

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
		clog(LevelWarn, "[%s] Call not found for outbound channel enter: %s", agentChannelID, agentChannelID)
		return
	}

	call.Lock()
	defer call.Unlock()

	// 2. Kilit Al ve Serbest Bırak

	// 3. Durum Kontrolü (Java: if (call.getState() == CALL_STATE.BRIDGED_WITH_AGENT))
	if call.State == CALL_STATE_BridgedWithAgent {
		clog(LevelInfo, "[%s] Got new outbound channel enter for already bridged call: %s. Terminating the new channel...", call.UniqueId, agentChannelID)

		// TO DO: ariFacade.hangup(call, channelId); eşleniği
		// Hata kontrolü eklenmelidir.
		_ = h.Hangup()
		return
	}

	// 5. Başarılı Atama ve Durum Güncelleme
	clog(LevelInfo, "[%s] Bridging client with agent channel %s", call.UniqueId, agentChannelID)

	// Java: attempt.setDialStatus(DIAL_STATUS.ANSWER);

	// Java: call.setState(CALL_STATE.BRIDGED_WITH_AGENT);
	call.State = CALL_STATE_BridgedWithAgent

	// Haritadaki struct'ı güncelle (Go map'leri struct'ları kopyalar)

	// 6. Harici Sistemleri Bilgilendirme (TO DO'lar)

	// queueLogger.onConnectAttemptSuccess(attempt);
	// TO DO: globalQueueLogger.OnConnectAttemptSuccess(attempt)

	// aidFacade.publishDistributionAttemptResult(...)
	// TO DO: globalAidFacade.PublishDistributionAttemptResult(call.ParentId, AID_DISTRIBUTION_STATE_Accepted, attempt.Username, call.QueueName)

	// callEventPublisher.publishCallQueueLeaveEvent(call);
	// TO DO: globalCallEventPublisher.PublishCallQueueLeaveEvent(call)

	// 7. Mevcut Zamanlanmış Eylemleri İptal Etme (cancelCurrentActions())
	globalScheduler.CancelByCallID(call.UniqueId)

	clog(LevelDebug, "[%s] All scheduled announcements cancelled.", call.UniqueId)

	// 8. Köprüleme İşlemi (bridgeCall(channelId))
	// CallManager metodu olarak çağrılır.
	go func() {
		if err := globalCallManager.bridgeCall(call, agentChannelID); err != nil {
			clog(LevelError, "[%s] Failed to bridge call: %v", call.UniqueId, err)
			return
		}
	}()

	// 9. Latecomer'ları Düşürme
	// if (call.getCurrentDistributionAttempts().size() > 1) { dropDistributionLatecomers(channelId); }
	/*	if len(call.CurrentDistributionAttempts) > 1 {
		globalCallManager.dropDistributionLatecomers(call, agentChannelID)
	} */
}

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
		clog(LevelWarn, "[%s] Call not found for outbound channel enter: %s", agentChannelID, agentChannelID)
		return
	}

	call.Lock()
	defer call.Unlock()

	// 2. Kilit Al ve Serbest Bırak

	// 3. Durum Kontrolü (Java: if (call.getState() == CALL_STATE.BRIDGED_WITH_AGENT))
	if call.State == CALL_STATE_BridgedWithAgent {
		clog(LevelInfo, "[%s] Got new outbound channel enter for already bridged call: %s. Terminating the new channel...", call.UniqueId, agentChannelID)

		// TO DO: ariFacade.hangup(call, channelId); eşleniği
		// Hata kontrolü eklenmelidir.
		_ = h.Hangup()
		return
	}

	// 5. Başarılı Atama ve Durum Güncelleme
	clog(LevelInfo, "[%s] Bridging client with agent channel %s", call.UniqueId, agentChannelID)

	// Java: attempt.setDialStatus(DIAL_STATUS.ANSWER);

	// Java: call.setState(CALL_STATE.BRIDGED_WITH_AGENT);
	call.State = CALL_STATE_BridgedWithAgent

	// Haritadaki struct'ı güncelle (Go map'leri struct'ları kopyalar)

	// 6. Harici Sistemleri Bilgilendirme (TO DO'lar)

	// queueLogger.onConnectAttemptSuccess(attempt);
	// TO DO: globalQueueLogger.OnConnectAttemptSuccess(attempt)

	// aidFacade.publishDistributionAttemptResult(...)
	// TO DO: globalAidFacade.PublishDistributionAttemptResult(call.ParentId, AID_DISTRIBUTION_STATE_Accepted, attempt.Username, call.QueueName)

	// callEventPublisher.publishCallQueueLeaveEvent(call);
	// TO DO: globalCallEventPublisher.PublishCallQueueLeaveEvent(call)

	// 7. Mevcut Zamanlanmış Eylemleri İptal Etme (cancelCurrentActions())
	globalScheduler.CancelByCallID(call.UniqueId)

	clog(LevelDebug, "[%s] All scheduled announcements cancelled.", call.UniqueId)

	// 8. Köprüleme İşlemi (bridgeCall(channelId))
	// CallManager metodu olarak çağrılır.
	go func() {
		if err := globalCallManager.bridgeCall(call, agentChannelID); err != nil {
			clog(LevelError, "[%s] Failed to bridge call: %v", call.UniqueId, err)

			return
		}
	}()

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
		clog(LevelError, "[%s] Cannot drop latecomers: ARI Client not found.", call.UniqueId)
		return
	}

	for currentChannelID := range call.CurrentDistributionAttempts {
		if currentChannelID != answeredChannelID {
			clog(LevelDebug, "[%s] Dropping latecomer channel: %s", call.UniqueId, currentChannelID)

			// Kanala erişim ve kapatma
			latecomerChannelHandle := ariClient.Channel().Get(currentChannelID)

			if err := latecomerChannelHandle.Hangup(); err != nil {
				clog(LevelError, "[%s] Failed to hangup latecomer %s: %v", call.UniqueId, currentChannelID, err)
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

	call.Lock()
	// Kilit altındaki verileri oku ve değiştir.
	bridgeID := BRIDGE_PREFIX + call.UniqueId
	call.Bridge = bridgeID
	call.BridgedChannel = agentChannelID
	call_ConnectionName := call.ConnectionName
	clientcall_ChannelID := call.UniqueId // Genellikle UniqueId, istemci kanal ID'si olarak kullanılır
	connectedAgentIdStr := fmt.Sprintf("%d", call.ConnectedAgentId)
	call.ConnectedAgentChannelID = agentChannelID
	//Kilidi Bırak
	call.Unlock()

	// 2. ARI Client'ı Bulma
	ariClient, found := globalClientManager.GetClient(call_ConnectionName)
	if !found {
		clog(LevelError, "[%s] ARI Client not found for connection ID: %s", clientcall_ChannelID, call_ConnectionName)
		return clearCallProperties(call, fmt.Errorf("ari client not found for connection id: %s", call_ConnectionName))
	}

	clog(LevelInfo, "Ari Client %+v", ariClient)
	clog(LevelInfo, "Ari Client Bridge  %+v", ariClient.Bridge())
	clog(LevelInfo, "Bridge ID :%s", bridgeID)

	bridgeKey := ari.NewKey(ari.BridgeKey, bridgeID)

	// 3. Köprü Oluşturma (ari.Client.Bridge().Create)
	bridge, err := ariClient.Bridge().Create(bridgeKey, "mixing", bridgeKey.ID)
	if err != nil {
		clog(LevelError, "[%s] Failed to create bridge %s: %v", clientcall_ChannelID, bridgeID, err)
		return clearCallProperties(call, err)
	}

	// 4. Kanalları Köprüye Ekleme (ari.Client.Channel().Get(id).AddToBridge)

	// a. İstemci Kanalını Köprüye Ekle
	clientChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, clientcall_ChannelID))

	if err := bridge.AddChannel(clientChannelHandle.Key().ID); err != nil {
		clog(LevelInfo, "failed to add channel to bridge , error : %+v", err)
		return clearCallProperties(call, err)
	}

	// b. Temsilci Kanalını Köprüye Ekle
	agentChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, agentChannelID))

	if err := bridge.AddChannel(agentChannelHandle.Key().ID); err != nil {
		clog(LevelInfo, "failed to add channel to bridge agent channel , error : %+v", err)
		return clearCallProperties(call, err)
	}

	globalCallManager.AddOutBoundCall(agentChannelID, agentChannelHandle)

	// 5. Kanal Değişkenini Ayarlama (ari.Client.Channel().Get(id).SetVariable)
	variableName := string(DIALPLAN_VARIABLE_QeQueueMember)

	// Değişkeni istemci kanalına ayarla
	if err := clientChannelHandle.SetVariable(variableName, connectedAgentIdStr); err != nil {
		clog(LevelError, "[%s] Failed to set variable %s to %s: %v", clientcall_ChannelID, variableName, connectedAgentIdStr, err)
		return clearCallProperties(call, err)
	}

	clog(LevelInfo, "[%s] Call successfully bridged: %s", clientcall_ChannelID, bridgeID)

	wg := new(sync.WaitGroup)

	wg.Add(1)

	go manageBridge(*redisClientManager.ctx, bridge, wg)

	wg.Wait()

	return nil // Başarılı
}

func clearCallProperties(call *Call, err error) error {

	call.Lock()
	call_ConnectionName := call.ConnectionName
	call_Bridge := call.Bridge
	call_BridgedChannel := call.BridgedChannel
	call_ConnectedAgentChannelID := call.ConnectedAgentChannelID

	call.Bridge = ""
	call.IsCallInDistribution = false
	call.BridgedChannel = ""

	call.State = CALL_STATE_InQueue
	call.ConnectedAgentId = 0

	call.Unlock()

	ariClient, found := globalClientManager.GetClient(call_ConnectionName)
	if found {
		if call_BridgedChannel != "" {
			agentChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, call_BridgedChannel))
			if agentChannelHandle.ID() != "" {
				agentChannelHandle.Hangup()
			}
		}
		if call_Bridge != "" {
			bridgeKey := ari.NewKey(ari.BridgeKey, call_Bridge)
			bridgeHandle := ariClient.Bridge().Get(bridgeKey)
			if bridgeHandle.ID() != "" {
				bridgeHandle.Delete()
			}
		}
		if call_ConnectedAgentChannelID != "" && call_ConnectedAgentChannelID != call_BridgedChannel {
			agentChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, call_ConnectedAgentChannelID))
			if agentChannelHandle.ID() != "" {
				agentChannelHandle.Hangup()
			}
		}
	}

	globalCallManager.RemoveOutBoundCall(call_BridgedChannel)

	// 2. ARI Client'ı Bulma

	return err

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
			clog(LevelDebug, "bridge destroyed")
			return
		case e, ok := <-enterSub.Events():
			if !ok {
				clog(LevelError, "channel entered subscription closed")
				return
			}

			v := e.(*ari.ChannelEnteredBridge)

			clog(LevelDebug, "channel entered bridge %+v", v.Channel.Name)

			go func() {
				if err := play.Play(ctx, h, play.URI("sound:confbridge-join")).Err(); err != nil {
					clog(LevelError, "failed to play join sound %s", err)
				}
			}()
		case e, ok := <-leaveSub.Events():
			if !ok {
				clog(LevelError, "channel left subscription closed")
				return
			}

			v := e.(*ari.ChannelLeftBridge)

			clog(LevelDebug, "channel left bridge , %s", v.Channel.Name)

			go func() {
				if err := play.Play(ctx, h, play.URI("sound:confbridge-leave")).Err(); err != nil {
					clog(LevelError, "failed to play leave sound %+v", err)
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
	call.Lock()
	call_UniqueId := call.UniqueId
	attemptNumber := call.DistributionAttemptNumber
	queueName := call.QueueName
	callDialContext := call.DialContext
	call_OutBoundApplicationName := call.OutBoundApplicationName
	call.ConnectedAgentChannelID = newChannelsUniqueId
	call.Unlock()

	currentQueue, err := globalQueueManager.GetQueueByName(queueName)
	if err != nil {
		errMsg := fmt.Sprintf("Queue not found: %s", queueName)
		clog(LevelError, "[%s] %s %+v", call_UniqueId, errMsg, err)
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
		clog(LevelError, "[%s] Originate failed via context %s: %v", call_UniqueId, dialContext, err)
	}

	clog(LevelInfo, "[%s] Outbound channel originated: %s", call_UniqueId, outBoundChannel.ID())

	return err
}

// originateOutboundChannel, belirtilen hedef (endpoint) için yeni bir kanal başlatır.
// Java'daki Mono<Call> yerine, başlatılan kanalın handle'ını ve hata (error) döndürür.
func originateOutboundChannel(call *Call, endpoint string, newChannelId string, variables map[string]string, stasisApp string) (*ari.ChannelHandle, error) {

	call.RLock()
	call_UniqueId := call.UniqueId
	call_ConnectionName := call.ConnectionName
	call.RUnlock()

	// 1. ARI İstemcisini Bulma
	ariClient, found := globalClientManager.GetClient(call_ConnectionName)
	if !found {
		errMsg := fmt.Sprintf("ARI Client not found for Connection ID: %s", call_ConnectionName)
		clog(LevelInfo, "Could not originate agent channel, connection name: %s, error : %s", call_ConnectionName, fmt.Errorf("%s", errMsg))
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Java'daki .setTimeout(CALL_MAX_TTL_SEC) -> Go'da sadece ARI'ya bildirilir.
	// Go ARI kütüphanesi doğrudan bir Timeout alanı sağlamaz,
	// bu genellikle değişkenler veya dialplan üzerinden yönetilir.
	// Ancak bu TTL, ARI'nin kendisi tarafından yönetilen bir durumdur ve burada doğrudan tanımlanmaz.
	//	}

	orgRequest := ari.OriginateRequest{
		Endpoint:   endpoint,
		App:        stasisApp,
		AppArgs:    call_UniqueId,
		ChannelID:  newChannelId,
		Originator: call_UniqueId,
		Timeout:    -1, //To DO: Buraya ne gelecek bulalaım .
		Variables:  variables,
	}

	// ARI Kütüphanesi Originate() metodunu kullanır.
	ch, err := ariClient.Channel().Originate(nil, orgRequest)
	if err != nil {
		clog(LevelError, "Could not originate outbound channel call_UniqueId : %s, error: %+v", call_UniqueId, err)
		return nil, fmt.Errorf("originate failed: %w", err)
	}

	// 4. Başarılı Loglama (Java'daki .doOnNext eşleniği)
	clog(LevelInfo, "[%s] Originated channel: %s", call_UniqueId, newChannelId)

	return ch, nil
}

//handleStasisEndMessage

func handleStasisEndMessage(message *ari.StasisEnd, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {

	clog(LevelInfo, "[%s] StasisEnd (connectionId : %s, application: %s, channel: %v)", ARI_MESSAGE_LOG_PREFIX, ariAppInfo.ConnectionName, message.Application, message.Channel)

	if message.Channel.ID == "" {
		log.Printf("ERROR: StasisStart message has no channel: %v", message)
		return // Kanal yoksa işlemi durdur
	}

	// Client Uygulamasına Giriş
	if isInboundApplication(message.Application) {
		OnClientChannelLeave(message, cl, h, ariAppInfo)
	} else if isOutboundApplication(message.Application) {
		OnOutboundChannelLeave(message, cl, h, ariAppInfo.ConnectionName)
	} else {
		log.Printf("ERROR: Got StasisEnd for unknown ARI application: %s", message.Application)
	}
}

func handleStasisStartMessage(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {

	clog(LevelInfo, "[%s] StasisStart (connectionId : %s, application: %s, channel: %v, args: %v)", ARI_MESSAGE_LOG_PREFIX, ariAppInfo.ConnectionName, message.Application, message.Channel, message.Args)

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

func handleDialMessage(message *ari.Dial) {

	clog(LevelInfo, "[%s] Dial (connectionId : %s, application: %s, channel: %v)", ARI_MESSAGE_LOG_PREFIX, message.Application, message.Dialstatus, message.GetChannelIDs())

	clog(LevelInfo, "message.Application %s", message.Application)
	clog(LevelInfo, "message.Caller.ID %s", message.Caller.ID)
	clog(LevelInfo, "message.Caller.State %s", message.Caller.State)
	clog(LevelInfo, "message.Dialstatus %s", message.Dialstatus)
	clog(LevelInfo, "message.Dialstring %s", message.Dialstring)
	clog(LevelInfo, "message.Peer.ID %s", message.Peer.ID)
	clog(LevelInfo, "message.Caller.State %s ", message.Caller.State)
	clog(LevelInfo, "message.Peer.State %s ", message.Peer.State)

	if message.Dialstatus == "" {
		return
	}

	if message.Peer.ID == "" {
		return
	}

	if message.Dialstatus != string(DIAL_STATUS_Answer) && message.Dialstatus != string(DIAL_STATUS_Progress) && message.Dialstatus != string(DIAL_STATUS_Ringing) {
		onOutboundChannelNoAnswer(message.Peer.ID, message.Caller.ID, message.Dialstatus)
	}

}

func onOutboundChannelNoAnswer(peerId string, callerId string, dialStatus string) {

	clog(LevelDebug, "Call onOutboundChannelNoAnswer, peerId : %s , caller Id : %s , dialStastus : %s", peerId, callerId, dialStatus)

	splittedPeerId := strings.Split(peerId, "-")

	call_UniqueuId := fmt.Sprintf("%s-%s", splittedPeerId[0], splittedPeerId[1])

	call, found := globalCallManager.GetCall(call_UniqueuId)

	if !found {
		clog(LevelDebug, "Call not found in onOutboundChannelNoAnswer, peerId : %s , caller Id : %s", peerId, callerId)
		return
	}

	clearCallProperties(call, nil)

	call.Lock()
	call_UniqueId := call.UniqueId
	call_InstanceId := call.InstanceID
	call_QueueName := call.QueueName

	call.Unlock()

	go func() {
		PublishInteractionStateMessage(InteractionState{
			InteractionID:        call_UniqueId,
			InstanceID:           call_InstanceId,
			RelatedAgentUsername: "",
			State:                "REJECTED",
			QueueName:            call_QueueName,
		})
	}()

}

func OnClientChannelLeave(message *ari.StasisEnd, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {
	clog(LevelInfo, "Inbound kanalından çıkış yapıldı..")

	clog(LevelInfo, "------------------ message.Channel.ID : %s", message.Channel.ID)

	call, found := globalCallManager.GetCall(message.Channel.ID)

	if !found {
		clog(LevelInfo, "Call is not found : %s", message.Channel.ID)
		return
	}

	call.RLock()
	if call.State == CALL_STATE_Terminated {
		clog(LevelInfo, "Call terminated before : %s", message.Channel.ID)
		return
	}
	call.RUnlock()

	go globalCallManager.terminateCall(call, CALL_TERMINATION_REASON_ClientHangup)

}

func OnClientChannelEnter(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {
	clog(LevelInfo, "Inbound kanalından giriş yapıldı..")

	call := CreateCall(message, message.Channel.ID, message.Channel.Name, message.Args[0], 1)

	//To DO:  Önce ARI bağlantısını kontrol et ,  uygun değilse çağrıyı reddet

	//To DO: Call boş mu

	if call == nil {

		//To DO: Kanalı reject et ...

		clog(LevelInfo, "Call is empty : %s", message.Channel.ID)
		return
	}

	key := h.Key()

	if key == nil {
		clog(LevelError, "[OnClientChannelEnter] Channel Key is nil for Channel ID : %s ", message.Channel.ID)
		return
	}

	call.ChannelKey = key
	call.Application = message.Application
	call.ConnectionName = ariAppInfo.ConnectionName
	call.OutBoundApplicationName = ariAppInfo.OutboundAppName
	call.InstanceID = ariAppInfo.InstanceID

	//To DO: AID bağlantısını kontrol et  , uygun değilse çağrıyı reddet

	//To DO: Redis Bağlatısını kontrol et  , uygun değilse çağrıyı reddet

	clog(LevelInfo, "[CALL_CREATED] %+v", call)
	go globalCallManager.processNewInboundCall(call)
}
