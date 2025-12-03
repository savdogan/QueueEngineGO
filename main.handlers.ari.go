package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/CyCoreSystems/ari/v6"
)

func OnOutboundChannelLeave(message *ari.StasisEnd, cl ari.Client, h *ari.ChannelHandle, connectionId string) {

	clog(LevelInfo, "Outbound kanalından çıkış yapıldı..")

	clog(LevelInfo, "------------------ message.Channel.ID : %s", message.Channel.ID)

	call, _, isCallFound, _ := g.CM.GetClientCallByAgentCall(message.Channel.ID)

	if !isCallFound {
		clog(LevelInfo, "Call is not found : %s", message.Channel.ID)
		return
	}

	call.Lock()
	if call.State == CALL_STATE_Terminated {
		clog(LevelInfo, "Call terminated before : %s", message.Channel.ID)
		call.Unlock()
		return
	}

	call.SetTerminationReason(CALL_TERMINATION_REASON_AgentHangup)
	ariClient, found := g.ACM.GetClient(call.ConnectionName)
	if found {
		g.CM.hangupChannel(call.ChannelId, call.UniqueId, ariClient, "")
	}
	call.Unlock()
}

// OnOutboundChannelEnter, temsilci kanalı stasis'e girdiğinde çağrılır.
func OnOutboundChannelEnter(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle, connectionId string) {

	agentChannelID := message.Channel.ID

	call, call_Uniqueu, isCallFound, _ := g.CM.GetClientCallByAgentCall(agentChannelID)

	if !isCallFound {
		clog(LevelWarn, "[%s] Call not found for outbound channel enter: %s", agentChannelID, agentChannelID)
		return
	}

	call.Lock()
	defer call.Unlock()

	// 2. Kilit Al ve Serbest Bırak

	// 3. Durum Kontrolü (Java: if (call.getState() == CALL_STATE.BRIDGED_WITH_AGENT))
	if call.State == CALL_STATE_BridgedWithAgent {
		clog(LevelInfo, "[%s] Got new outbound channel enter for already bridged call: %s. Terminating the new channel...", call_Uniqueu, agentChannelID)

		// TO DO: ariFacade.hangup(call, channelId); eşleniği
		// Hata kontrolü eklenmelidir.
		_ = h.Hangup()
		return
	}

	// 5. Başarılı Atama ve Durum Güncelleme
	clog(LevelInfo, "[%s] Bridging client with agent channel %s", call_Uniqueu, agentChannelID)

	// Java: attempt.setDialStatus(DIAL_STATUS.ANSWER);

	// Java: call.setState(CALL_STATE.BRIDGED_WITH_AGENT);
	call.State = CALL_STATE_BridgedWithAgent

	// Haritadaki struct'ı güncelle (Go map'leri struct'ları kopyalar)

	// 6. Harici Sistemleri Bilgilendirme (TO DO'lar)

	// queueLogger.onConnectAttemptSuccess(attempt);
	// TO DO: globalQueueLogger.OnConnectAttemptSuccess(attempt)
	call.LogOnConnectAttemptSuccess(false)

	// aidFacade.publishDistributionAttemptResult(...)
	// TO DO: globalAidFacade.PublishDistributionAttemptResult(call.ParentId, AID_DISTRIBUTION_STATE_Accepted, attempt.Username, call.QueueName)

	// callEventPublisher.publishCallQueueLeaveEvent(call);
	// TO DO: globalCallEventPublisher.PublishCallQueueLeaveEvent(call)
	go sendInteractionStateMessage(call, AID_DISTRIBUTION_STATE_Accepted, call.ConnectedUserName)

	// 7. Mevcut Zamanlanmış Eylemleri İptal Etme (cancelCurrentActions())
	g.CM.cancelCallActions(call, true)

	clog(LevelDebug, "[%s] All scheduled announcements cancelled.", call_Uniqueu)

	// 8. Köprüleme İşlemi (bridgeCall(channelId))
	// CallManager metodu olarak çağrılır.
	go func() {
		if err := g.CM.bridgeCall(call, agentChannelID); err != nil {
			clog(LevelError, "[%s] Failed to bridge call: %v", call_Uniqueu, err)

			return
		}
		clog(LevelInfo, "[%s] Bridge is successed.", call_Uniqueu)
	}()
}

/*
// dropDistributionLatecomers, OnOutboundChannelEnter içinden Lock altında çağrılır.
func (cm *CallManager) dropDistributionLatecomers(call *Call, answeredChannelID string) {
	// Burada kilit zaten OnOutboundChannelEnter tarafından tutuluyor olmalı.

	// Temsilcinin ARI istemcisine erişmek için ConnectionId kullanılır.
	ariClient, found := g.ACM.GetClient(call.ConnectionName)
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
	ariClient, found := g.ACM.GetClient(call_ConnectionName)
	if !found {
		clog(LevelError, "[%s] ARI Client not found for connection ID: %s", clientcall_ChannelID, call_ConnectionName)
		return reSetupCallForQueue(call, fmt.Errorf("ari client not found for connection id: %s", call_ConnectionName), true)
	}

	clog(LevelInfo, "Ari Client %+v", ariClient)
	clog(LevelInfo, "Ari Client Bridge  %+v", ariClient.Bridge())
	clog(LevelInfo, "Bridge ID :%s", bridgeID)

	bridgeKey := ari.NewKey(ari.BridgeKey, bridgeID)

	// 3. Köprü Oluşturma (ari.Client.Bridge().Create)
	bridge, err := ariClient.Bridge().Create(bridgeKey, "mixing", bridgeKey.ID)
	if err != nil {
		clog(LevelError, "[%s] Failed to create bridge %s: %v", clientcall_ChannelID, bridgeID, err)
		return reSetupCallForQueue(call, err, true)
	}

	// 4. Kanalları Köprüye Ekleme (ari.Client.Channel().Get(id).AddToBridge)

	// a. İstemci Kanalını Köprüye Ekle
	clientChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, clientcall_ChannelID))

	if err := bridge.AddChannel(clientChannelHandle.Key().ID); err != nil {
		clog(LevelInfo, "failed to add channel to bridge , error : %+v", err)
		return reSetupCallForQueue(call, err, true)
	}

	// b. Temsilci Kanalını Köprüye Ekle
	agentChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, agentChannelID))

	if err := bridge.AddChannel(agentChannelHandle.Key().ID); err != nil {
		clog(LevelInfo, "failed to add channel to bridge agent channel , error : %+v", err)
		return reSetupCallForQueue(call, err, true)
	}

	g.CM.AddOutBoundCall(agentChannelID, agentChannelHandle)

	// 5. Kanal Değişkenini Ayarlama (ari.Client.Channel().Get(id).SetVariable)
	variableName := string(DIALPLAN_VARIABLE_QeQueueMember)

	// Değişkeni istemci kanalına ayarla
	if err := clientChannelHandle.SetVariable(variableName, connectedAgentIdStr); err != nil {
		clog(LevelError, "[%s] Failed to set variable %s to %s: %v", clientcall_ChannelID, variableName, connectedAgentIdStr, err)
		return reSetupCallForQueue(call, err, true)
	}

	clog(LevelInfo, "[%s] Call successfully bridged: %s", clientcall_ChannelID, bridgeID)

	wg := new(sync.WaitGroup)

	wg.Add(1)

	go manageBridge(g.Ctx, bridge, wg)

	wg.Wait()

	return nil // Başarılı
}

func reSetupCallForQueue(call *Call, err error, reQueue bool) error {

	call.Lock()
	call_UniqueId := call.UniqueId
	call_ConnectionName := call.ConnectionName
	call_Bridge := call.Bridge
	call_BridgedChannel := call.BridgedChannel
	call_ConnectedAgentChannelID := call.ConnectedAgentChannelID
	call_ConnectedUserName := call.ConnectedUserName

	call.Bridge = ""
	call.IsCallInDistribution = false
	call.BridgedChannel = ""
	call.ConnectedAgentId = 0
	call.ConnectedAgentChannelID = ""
	call.ConnectedUserName = ""
	call.State = CALL_STATE_InQueue

	call.Unlock()

	ariClient, found := g.ACM.GetClient(call_ConnectionName)
	if found {

		g.CM.terminateBridge(call_Bridge, call_UniqueId, ariClient)

		g.CM.hangupChannel(call_BridgedChannel, call_UniqueId, ariClient, "")

		g.CM.hangupChannel(call_ConnectedAgentChannelID, call_UniqueId, ariClient, "")

	} else {
		clog(LevelError, "Ari Client not found in resetup call %s", call_UniqueId)
	}

	g.CM.RemoveOutBoundCall(call_BridgedChannel)
	g.CM.RemoveAgentCallMap(call_ConnectedAgentChannelID)

	if reQueue {

		//To DO: setup annonunce yeniden.....
		g.CM.startMoh(call, false)

		go sendInteractionStateMessage(call, AID_DISTRIBUTION_STATE_Rejected, call_ConnectedUserName)
	}

	return err

}

func (cm *CallManager) cancelCallActions(call *Call, callLocked bool) {

	g.SM.CancelByCallID(call.UniqueId)

	if !callLocked {
		call.RLock()
	}

	cm.stopMoh(call, true)

	if !callLocked {
		call.RUnlock()
	}

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

			//go func() {
			//	if err := play.Play(ctx, h, play.URI("sound:confbridge-join")).Err(); err != nil {
			//		clog(LevelError, "failed to play join sound %s", err)
			//	}
			//}()
		case e, ok := <-leaveSub.Events():
			if !ok {
				clog(LevelError, "channel left subscription closed")
				return
			}

			v := e.(*ari.ChannelLeftBridge)

			clog(LevelDebug, "channel left bridge , %s", v.Channel.Name)

			//go func() {
			//	if err := play.Play(ctx, h, play.URI("sound:confbridge-leave")).Err(); err != nil {
			//		clog(LevelError, "failed to play leave sound %+v", err)
			//	}
			//}()
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

//handleStasisEndMessage

func handleStasisEndMessage(message *ari.StasisEnd, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {

	clog(LevelTrace, "[%s] StasisEnd (connectionId : %s, application: %s, channel: %v)", ARI_MESSAGE_LOG_PREFIX, ariAppInfo.ConnectionName, message.Application, message.Channel)

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

	clog(LevelTrace, "[%s] StasisStart (connectionId : %s, application: %s, channel: %v, args: %v)", ARI_MESSAGE_LOG_PREFIX, ariAppInfo.ConnectionName, message.Application, message.Channel, message.Args)

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

	clog(LevelTrace, "[%s] Dial (connectionId : %s, application: %s, channel: %v)", ARI_MESSAGE_LOG_PREFIX, message.Application, message.Dialstatus, message.GetChannelIDs())

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

	clog(LevelTrace, "Call onOutboundChannelNoAnswer, peerId : %s , caller Id : %s , dialStastus : %s", peerId, callerId, dialStatus)

	call, _, isCallFound, _ := g.CM.GetClientCallByAgentCall(peerId)

	if !isCallFound {
		clog(LevelTrace, "Call not found in onOutboundChannelNoAnswer, peerId : %s , caller Id : %s", peerId, callerId)
		return
	}

	call.LogOnConnectAttemptFail()

	reSetupCallForQueue(call, nil, true)

	go sendInteractionStateMessage(call, AID_DISTRIBUTION_STATE_Rejected, "")

}

func OnClientChannelLeave(message *ari.StasisEnd, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {
	clog(LevelInfo, "Inbound kanalından çıkış yapıldı..")

	clog(LevelInfo, "------------------ message.Channel.ID : %s", message.Channel.ID)

	call, found := g.CM.GetCall(message.Channel.ID)

	if !found {
		clog(LevelInfo, "Call is not found : %s", message.Channel.ID)
		return
	}

	call.Lock()

	if call.State == CALL_STATE_Terminated {
		clog(LevelInfo, "Call terminated before : %s", message.Channel.ID)
		return
	}

	call_TerminationReason := CALL_TERMINATION_REASON_ClientHangup

	if call.State == CALL_STATE_BridgedWithAgent {
		call_TerminationReason = CALL_TERMINATION_REASON_Abandon
	}

	call.Unlock()

	go g.CM.terminateCall(call, call_TerminationReason)

}

func OnClientChannelEnter(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {
	clog(LevelInfo, "Inbound kanalından giriş yapıldı..")

	call := CreateCall(message, message.Channel.ID, message.Channel.Name, message.Args[0], ariAppInfo.MediaServerId)

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
	call.LogOnInit()

	//To DO: AID bağlantısını kontrol et  , uygun değilse çağrıyı reddet

	//To DO: Redis Bağlatısını kontrol et  , uygun değilse çağrıyı reddet

	clog(LevelInfo, "[CALL_CREATED] %+v", call)
	go g.CM.processNewInboundCall(call)
}
