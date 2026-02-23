package main

import (
	"fmt"

	"github.com/CyCoreSystems/ari/v6"
)

func onOutboundChannelLeave(message *ari.StasisEnd) {

	clog(LevelInfo, "[OUTBOUND_EXITED] %s", message.Channel.ID)

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

func onClientChannelLeave(message *ari.StasisEnd) {

	clog(LevelInfo, "[INBOUND_EXITED] [%s] ", message.Channel.ID)

	call, found := g.CM.GetCall(message.Channel.ID)

	if !found {
		clog(LevelInfo, "Call is not found : %s", message.Channel.ID)
		return
	}

	call.Lock()
	defer call.Unlock()

	call_TerminationReason := CALL_TERMINATION_REASON_ClientHangup

	if call.State == CALL_STATE_BridgedWithAgent {
		call_TerminationReason = CALL_TERMINATION_REASON_Abandon
	}

	go g.CM.terminateCall(call, call_TerminationReason)

}

// onOutboundChannelEnter, temsilci kanalı stasis'e girdiğinde çağrılır.
func onOutboundChannelEnter(message *ari.StasisStart, h *ari.ChannelHandle) {

	agentChannelID := message.Channel.ID

	call, call_Uniqueu, isCallFound, _ := g.CM.GetClientCallByAgentCall(agentChannelID)

	if !isCallFound {
		clog(LevelWarn, "[%s] Call not found for outbound channel enter: %s", agentChannelID, agentChannelID)
		return
	}

	call.Lock()
	defer call.Unlock()

	// 3. Durum Kontrolü (Java: if (call.getState() == CALL_STATE.BRIDGED_WITH_AGENT))
	if call.State == CALL_STATE_BridgedWithAgent {
		clog(LevelInfo, "[CALL_ALREADY_BRIDGED] [%s] Got new outbound channel enter for already bridged call: %s. Terminating the new channel...", call_Uniqueu, agentChannelID)
		err := h.Hangup()
		if err != nil {
			clog(LevelError, "[CALL_ALREADY_BRIDGED_HANGUP_ERROR] %s", call.UniqueId)
			return
		}
	}

	// 5. Başarılı Atama ve Durum Güncelleme
	clog(LevelInfo, "[%s] Bridging client with agent channel %s", call_Uniqueu, agentChannelID)

	// Java: call.setState(CALL_STATE.BRIDGED_WITH_AGENT);
	call.State = CALL_STATE_BridgedWithAgent

	// Haritadaki struct'ı güncelle (Go map'leri struct'ları kopyalar)

	// 6. Harici Sistemleri Bilgilendirme (TO DO'lar)

	// queueLogger.onConnectAttemptSuccess(attempt);
	call.LogOnConnectAttemptSuccess(false)

	// aidFacade.publishDistributionAttemptResult(...)
	// TO DO: globalAidFacade.PublishDistributionAttemptResult(call.ParentId, AID_DISTRIBUTION_STATE_Accepted, attempt.Username, call.QueueName)

	// callEventPublisher.publishCallQueueLeaveEvent(call);
	// TO DO: globalCallEventPublisher.PublishCallQueueLeaveEvent(call)

	// 7. Mevcut Zamanlanmış Eylemleri İptal Etme (cancelCurrentActions())
	g.CM.cancelCallActions(call, true)

	// 8. Köprüleme İşlemi (bridgeCall(channelId))
	// CallManager metodu olarak çağrılır.
	go func() {
		if err := g.CM.bridgeCall(call, agentChannelID); err != nil {
			clog(LevelError, "[%s] Failed to bridge call: %v", call_Uniqueu, err)
			return
		}
		clog(LevelInfo, "[%s] Bridge is successed.", call_Uniqueu)
		go sendInteractionStateMessage(call, AID_DISTRIBUTION_STATE_Accepted, call.ConnectedUserName)
	}()
}

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
		call.Lock()
	}

	call.WaitingActions = []CALL_SCHEDULED_ACTION{}

	cm.stopMoh(call, true)

	if call.ActiveActionCancelFunc != nil {
		call.ActiveActionCancelFunc()
	}

	if !callLocked {
		call.Unlock()
	}

}

func handleStasisEndMessage(message *ari.StasisEnd, ariAppInfo AriAppInfo) {

	clog(LevelTrace, "[%s] StasisEnd (connectionId : %s, application: %s, channel: %v)", ARI_MESSAGE_LOG_PREFIX, ariAppInfo.ConnectionName, message.Application, message.Channel)

	if message.Channel.ID == "" {
		clog(LevelError, "[StasisEnd] message has no channel %+v", message)
		return // Kanal yoksa işlemi durdur
	}

	// Client Uygulamasına Giriş
	if isInboundApplication(message.Application) {
		onClientChannelLeave(message)
	} else if isOutboundApplication(message.Application) {
		onOutboundChannelLeave(message)
	} else {
		clog(LevelError, "[UNKNOWN_APPLICATION] Got StasisEnd for unknown ARI application: %s", message.Application)
	}
}

func handleStasisStartMessage(message *ari.StasisStart, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {

	clog(LevelTrace, "[%s] StasisStart (connectionId : %s, application: %s, channel: %v, args: %v)", ARI_MESSAGE_LOG_PREFIX, ariAppInfo.ConnectionName, message.Application, message.Channel, message.Args)

	if message.Channel.ID == "" {
		clog(LevelError, "[StasisStart] message has no channel %+v", message)
		return // Kanal yoksa işlemi durdur
	}

	// Args kontrolü. Go'da boş string dizisi kontrolü: len(message.Args) == 0
	if len(message.Args) == 0 {
		clog(LevelError, "[NO_ARGUMENT] %s message has no arguments: %v", message.Channel.ID, message)
		return // Argüman yoksa işlemi durdur
	}

	// Client Uygulamasına Giriş
	if isInboundApplication(message.Application) {
		onClientChannelEnter(message, h, ariAppInfo)
	} else if isOutboundApplication(message.Application) {
		onOutboundChannelEnter(message, h)
	} else {
		clog(LevelError, "[UNKNOWN_APPLICATION] Got StasisEnd for unknown ARI application: %s", message.Application)
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

	call, _, found, _ := g.CM.GetClientCallByAgentCall(peerId)

	if !found {
		clog(LevelTrace, "[CALL_NOT_FOUND] %s in onOutboundChannelNoAnswer, peerId : %s ", callerId, peerId)
		return
	}

	call.LogOnConnectAttemptFail()

	reSetupCallForQueue(call, nil, true)

	go sendInteractionStateMessage(call, AID_DISTRIBUTION_STATE_Rejected, "")

}

func onClientChannelEnter(message *ari.StasisStart, h *ari.ChannelHandle, ariAppInfo AriAppInfo) {

	clog(LevelInfo, "[INBOUND_ENTERED] %s", message.Channel.ID)

	call := CreateCall(message, message.Channel.ID, message.Channel.Name, message.Args[0], ariAppInfo.MediaServerId)

	if call == nil {
		h.Hangup()
		clog(LevelError, "[CALL_CREATE_ERROR] %s call is empty , channel hangup", message.Channel.ID)
		return
	}

	key := h.Key()

	if key == nil {
		clog(LevelError, "[CALL_CHANNEL_ERROR] %s Channel Key is nil", message.Channel.ID)
		return
	}

	call.ChannelKey = key
	call.Application = message.Application
	call.ConnectionName = ariAppInfo.ConnectionName
	call.OutBoundApplicationName = ariAppInfo.OutboundAppName
	call.InstanceID = ariAppInfo.InstanceID
	call.LogOnInit()

	clog(LevelDebug, "[CALL_CREATED] %+v", call)

	go g.CM.processNewInboundCall(call)
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
