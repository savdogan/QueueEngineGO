package main

import (
	"fmt"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/CyCoreSystems/ari/v6"
)

// NewCallManager, CallManager'ın güvenli bir örneğini oluşturur ve başlatır.
func NewCallManager() *CallManager {
	return &CallManager{
		calls:           make(map[string]*Call),
		outChannels:     make(map[string]*ari.ChannelHandle),
		agentCalltoCall: make(map[string]string),
	}
}

// --- İŞLEM METOTLARI ---

// processNewInboundCall, yeni bir Call nesnesini yöneticinin listesine ekler.
func (cm *CallManager) processNewInboundCall(call *Call) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.Lock()
	cm.calls[call.UniqueId] = call
	cm.Unlock()

	logCallInfo(call, fmt.Sprintf("[ADDED]: %s , Call Manager Calls Count : %d", call.UniqueId, len(cm.calls)))

	go g.CM.processCall(call)

}

func (cm *CallManager) processCall(call *Call) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz

	//To DO: burada konference nasıl seçeceğiz
	cm.queueProcessStart(call)

}

func (cm *CallManager) queueProcessStart(call *Call) {

	call.Lock()
	call_UniqueuId := call.UniqueId
	call.CurrentProcessName = PROCESS_NAME_queue
	call_QueueName := call.QueueName
	call_InstanceID := call.InstanceID
	call.Unlock()

	clog(LevelDebug, "Call is in Queue Process Start : %s ", call_UniqueuId)

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueuId, call_QueueName)
		return
	}

	currentQueue.RLock()
	queue_WaitTimeout := currentQueue.WaitTimeout
	queue_ClientAnnounceMinEstimationTime := currentQueue.ClientAnnounceMinEstimationTime
	queue_ClientAnnounceSoundFile := currentQueue.ClientAnnounceSoundFile
	currentQueue.RUnlock()

	//scheduleQueueTimeout
	if queue_WaitTimeout > 0 {

		delay := time.Duration(queue_WaitTimeout) * time.Second

		g.SM.ScheduleTask(call_UniqueuId, delay, func() {
			processQueueAction(call_UniqueuId, CALL_SCHEDULED_ACTION_QueueTimeout)
		})
	} else {
		clog(LevelDebug, "No Queue Timeout set for Call: %s , QueueName : %s", call_UniqueuId, call_QueueName)
	}

	//scheduleClientAnnounce
	//To Do: Get From AID call estimation time  and aşağıdaki koşula ekle
	if queue_ClientAnnounceSoundFile != "" && queue_ClientAnnounceMinEstimationTime > 0 {

		g.SM.ScheduleTask(call_UniqueuId, 1, func() {
			processQueueAction(call_UniqueuId, CALL_SCHEDULED_ACTION_ClientAnnounce)
		})
	}

	//schedulePeriodicAnnounce
	go cm.setupPeriodicAnnounce(call_UniqueuId)
	//scheduleActionAnnounce
	go cm.setupActionAnnounce(call_UniqueuId)
	//schedulePositionAnnounce
	go cm.setupPositionAnnounce(call_UniqueuId)

	call.Lock()
	call.State = CALL_STATE_InQueue
	call.Unlock()

	err = PublishNewInteractionMessage(NewCallInteraction{
		Groups:              []int{},
		InstanceID:          call_InstanceID,
		InteractionID:       call_UniqueuId,
		InteractionPriority: 5,
		InteractionQueue:    call_QueueName,
		PreferredAgent:      "",
		RequiredSkills:      []int{},
		Timestamp:           time.Now().UnixMilli(),
	})

	if err != nil {
		clog(LevelError, "Publish New Interaction Message is failed for Call: %s , QueueName : %s , Error : %v", call_UniqueuId, call_QueueName, err)
		return
	}

	clog(LevelDebug, "Call is in Queue Process Setup Completed : %s ", call_UniqueuId)

}

// terminateCall, verilen çağrıyı güvenli bir şekilde sonlandırır ve kaynakları temizler.
func (cm *CallManager) terminateCall(call *Call, terminationReason CALL_TERMINATION_REASON) {

	call.Lock() //------------------------------------------

	call_UniqueId := call.UniqueId
	call_ConnectionName := call.ConnectionName
	call_Bridge := call.Bridge
	call_BridgedChannel := call.BridgedChannel
	call_ConnectedAgentChannelID := call.ConnectedAgentChannelID
	call_ConnectedUserName := call.ConnectedUserName
	call_State := call.State

	call.Bridge = ""
	call.IsCallInDistribution = false
	call.BridgedChannel = ""
	call.ConnectedAgentId = 0
	call.ConnectedAgentChannelID = ""
	call.ConnectedUserName = ""
	call.State = CALL_STATE_InQueue

	if call_State == CALL_STATE_Terminated {
		clog(LevelDebug, "[%s] Call already terminated. No action taken.", call_UniqueId)
		call.Unlock() //------------------------------------------
		return        // Zaten sonlanmışsa, hiçbir şey yapma
	}
	call.State = CALL_STATE_Terminated
	if call.TerminationReason == "" {
		call.SetTerminationReason(terminationReason)
	}

	call.Unlock() //------------------------------------------

	ariClient, found := g.ACM.GetClient(call_ConnectionName)
	if found {

		g.CM.terminateBridge(call_Bridge, call_UniqueId, ariClient)

		g.CM.hangupChannel(call_BridgedChannel, call_UniqueId, ariClient, "")

		g.CM.hangupChannel(call_ConnectedAgentChannelID, call_UniqueId, ariClient, "")

	} else {
		clog(LevelError, "Ari Client not found in terminate call %s", call_UniqueId)
	}

	g.CM.RemoveOutBoundCall(call_BridgedChannel)
	g.CM.RemoveAgentCallMap(call_ConnectedAgentChannelID)

	logAnyInfo(call, "Terminated Call")

	call.LogOnTerminate()

	cm.RemoveCall(call_UniqueId)

	go sendInteractionStateMessage(call, AID_DISTRIBUTION_STATE_Terminated, call_ConnectedUserName)

}

func sendInteractionStateMessage(call *Call, state AID_DISTRIBUTION_STATE, agentUserName string) {

	agentUserNameL := agentUserName

	call.RLock()
	call_UniqueId := call.UniqueId
	call_InstanceId := call.InstanceID
	call_QueueName := call.QueueName
	if agentUserName == "" {
		agentUserNameL = call.ConnectedUserName
	}
	call.RUnlock()

	PublishInteractionStateMessage(InteractionState{
		InteractionID:        call_UniqueId,
		InstanceID:           call_InstanceId,
		RelatedAgentUsername: agentUserNameL,
		State:                string(state),
		QueueName:            call_QueueName,
	})

}

// onAidDistributionMessage, AID'den gelen dağıtım isteğini işler.
func (cm *CallManager) onAidDistributionMessage(call *Call, message *RedisCallDistributionMessage) {

	dialTimeout := 5000
	queue_Name := message.QueueName
	call_UniqueId := message.InteractionID
	agent := message.Users[0]

	currentQueue, err := g.QCM.GetQueueByName(queue_Name)
	if err != nil {
		errMsg := fmt.Sprintf("Queue not found: %s", queue_Name)
		clog(LevelError, "[%s] %s %+v", call_UniqueId, errMsg, err)
		return
	}

	currentQueue.RLock()
	queueID := message.QueueName
	musicClassOnHold := currentQueue.MusicClassOnHold
	agentAnnounceFile := currentQueue.Announce
	currentQueue.RUnlock()

	call.Lock() // Kilitleme

	// 1. Durum Kontrolü
	if call.IsCallInDistribution {
		clog(LevelWarn, "[%s] Got new distribution request while executing the previous one. Ignoring...", call.UniqueId)
		call.Unlock()
		return
	} else if call.State == CALL_STATE_BridgedWithAgent {
		clog(LevelWarn, "[%s] Got new distribution request but there is already connected agent. Dismissing distribution...", call.UniqueId)
		call.Unlock()
		go sendInteractionStateMessage(call, AID_DISTRIBUTION_STATE_Dismissed, agent.Username)
		return
	} else if call.DistributionAttemptNumber > 2 {
		clog(LevelWarn, "[%s] call.DistributionAttemptNumber is reached.", call.UniqueId)
		ariClient, found := g.ACM.GetClient(call.ConnectionName)
		call.SetTerminationReason(CALL_TERMINATION_REASON_MaxAttemptReached)
		call.Unlock()
		if found {
			g.CM.hangupChannel(call.ChannelId, call.UniqueId, ariClient, "")
		}
		return
	}

	//Call
	call.DistributionAttemptNumber++
	call.IsCallInDistribution = true
	call.ConnectedAgentId = agent.ID
	call.ConnectedUserName = agent.Username
	call.AttempStartTime = time.Now()

	// 3. Logger Başlatma
	//ds.QueueLogger.OnDistributionIterationStart()

	// Agent Kanal ID Oluşturma
	agentChannelID := fmt.Sprintf("%s-agent-%s-%d", call.UniqueId, agent.Username, call.DistributionAttemptNumber)
	call.ConnectedAgentChannelID = agentChannelID
	// variables tanımlamaları

	variables := make(map[string]string)

	// DIALPLAN_VARIABLE.dialUri
	variables[string(DIALPLAN_VARIABLE_DialUri)] = buildAgentUri(agent)

	// DIALPLAN_VARIABLE.dialTimeout
	variables[string(DIALPLAN_VARIABLE_DialTimeout)] = fmt.Sprintf("%d", dialTimeout)

	// CDR değişkeni (call_type)
	variables[getCdrVariableName(DIALPLAN_VARIABLE_CallType)] = AGENT_LEG_CALL_TYPE

	// 3. SIP Başlıklarını Ekle

	// Dağıtım Girişimi Sayısı (DISTRIBUTION_ATTEMPT_NUMBER)
	addSipHeaderVariable(variables, SIP_HEADERS["DISTRIBUTION_ATTEMPT_NUMBER"], fmt.Sprintf("%d", call.DistributionAttemptNumber))

	// Kuyruk Adı (QUEUE_NAME)
	addSipHeaderVariable(variables, SIP_HEADERS["QUEUE_NAME"], queue_Name)

	// Kuyruk ID (QUEUE_ID)
	addSipHeaderVariable(variables, SIP_HEADERS["QUEUE_ID"], queueID)

	// MOH Sınıfı (HOLD_MOH_CLASS)
	if strings.TrimSpace(musicClassOnHold) != "" {
		addSipHeaderVariable(variables, SIP_HEADERS["HOLD_MOH_CLASS"], musicClassOnHold)
	}

	// Agent Announce
	if strings.TrimSpace(agentAnnounceFile) != "" {
		variables[string(DIALPLAN_VARIABLE_AgentAnnounce)] = agentAnnounceFile
	}

	// Dial Context Belirleme
	// (StringUtils.isNotBlank(call.getDialContext())) ? call.getDialContext() : DIALPLAN_CONTEXT.qe_dial_agent.toString();

	dialContext := string(DIALPLAN_CONTEXT_QeDialAgent) // Varsayılan değer
	if strings.TrimSpace(call.DialContext) != "" {
		dialContext = call.DialContext
	}

	// Başlatma Hedefini Oluşturma
	endpoint := fmt.Sprintf("Local/s@%s", dialContext)

	// 1. ARI İstemcisini Bulma
	ariClient, found := g.ACM.GetClient(call.ConnectionName)
	if !found {
		errMsg := fmt.Sprintf("ARI Client not found for Connection ID: %s", call.ConnectionName)
		clog(LevelInfo, "Could not originate agent channel, connection name: %s, error : %s", call.ConnectionName, fmt.Errorf("%s", errMsg))
		call.Unlock()
		reSetupCallForQueue(call, fmt.Errorf("%s", errMsg), true)
		return
	}

	orgRequest := ari.OriginateRequest{
		Endpoint:   endpoint,
		App:        call.OutBoundApplicationName,
		AppArgs:    call_UniqueId,
		ChannelID:  agentChannelID,
		Originator: call_UniqueId,
		Timeout:    -1, //To DO: Buraya ne gelecek bulalaım .
		Variables:  variables,
	}

	call.Unlock()

	g.CM.AddAgentCallMap(agentChannelID, call_UniqueId)

	// ARI Kütüphanesi Originate() metodunu kullanır.
	ch, err := ariClient.Channel().Originate(nil, orgRequest)
	if err != nil {
		clog(LevelError, "Could not originate outbound channel call_UniqueId : %s, error: %+v", call_UniqueId, err)
		reSetupCallForQueue(call, err, true)
		return
	}

	clog(LevelInfo, "[%s] Originated channel: %s, channelID : %s", call_UniqueId, agentChannelID, ch.ID())
}

func (cm *CallManager) hangupChannel(channelId string, callUniqueId string, ariClient ari.Client, reason string) {
	if channelId == "" {
		return
	}

	hangupReason := "normal"

	if reason != "" {
		hangupReason = reason
	}

	// Direkt kapatmayı dene
	err := ariClient.Channel().Hangup(ari.NewKey(ari.ChannelKey, channelId), hangupReason)

	if err != nil {
		// Eğer hata "Not Found" ise zaten amacımıza ulaşmışızdır (kanal kapalı).
		// Hata "Not Found" DEĞİLSE logla.
		if !strings.Contains(err.Error(), "Not Found") {
			clog(LevelError, "Failed to hangup channel %s:, main call : %s,  %v", channelId, callUniqueId, err)
		}
	}
}

func (cm *CallManager) terminateBridge(bridgeChannelId string, callUniqueId string, ariClient ari.Client) {
	if bridgeChannelId != "" {
		bridgeKey := ari.NewKey(ari.BridgeKey, bridgeChannelId)
		bridgeHandle := ariClient.Bridge().Get(bridgeKey)
		if bridgeHandle.ID() != "" {
			clog(LevelTrace, "Bridge(%s) is deleting, Call Id : %s", bridgeChannelId, callUniqueId)
			err := bridgeHandle.Delete()
			if err != nil {
				clog(LevelError, "Error while deleting the bridge : %s , error : %+v ", bridgeChannelId, err)
			}
		} else {
			clog(LevelTrace, "function bridgeHandle.ID() did not return any bridge handle for the bridge channel id (%s) of the call (%s)", bridgeChannelId, callUniqueId)
		}
	} else {
		clog(LevelTrace, "Any bridge is found for deleting , call id : %s", callUniqueId)
	}
}

func (qLog *WbpQueueLog) calculateQueueWaitDuration(now time.Time) float64 {
	// --- EKLENECEK KISIM ---
	if qLog == nil {
		return 0.0
	}
	// -----------------------

	endTime := now
	if qLog.ConnectTime != nil {
		endTime = *qLog.ConnectTime
	}

	duration := endTime.Sub(qLog.ArrivalTime).Seconds()
	if duration < 0 {
		return 0
	}
	return duration
}

func (qLog *WbpQueueLog) calculateInCallDuration(now time.Time) float64 {

	if qLog.ConnectTime == nil {
		return 0.0
	}
	// Konuşma süresi: Ayrılma Zamanı (Now) - Bağlanma Zamanı
	duration := now.Sub(*qLog.ConnectTime).Seconds()

	if duration < 0 {
		return 0
	}
	return duration
}

func (call *Call) LogOnConnectAttemptSuccess(lockCall bool) {

	if lockCall {
		call.Lock()
		defer call.Unlock()
	}

	if call.QeueuLog == nil {
		clog(LevelInfo, "Call Queue Log is empty: %s", call.UniqueId)
		return
	}

	// Kolay erişim için alias
	qLog := call.QeueuLog

	// 2. Zaman Damgaları
	// Java: System.nanoTime() ve new Date()
	now := time.Now()

	// Attempt nesnesini güncelleme
	call.AttempEndTime = now
	// attempt.QueueLog = qLog // Eğer struct içinde varsa set edilir.
	if !call.AttempStartTime.IsZero() {
		durationSec := now.Sub(call.AttempStartTime).Seconds()
		qLog.ConnectAttemptsDuration += durationSec
	}

	// 4. Bağlanma Zamanı (Nullable olduğu için pointer atıyoruz)
	qLog.ConnectTime = &now

	pos := call.Position
	qLog.FinalPosition = &pos

	agentName := call.ConnectedUserName
	agentID := call.ConnectedAgentId

	qLog.ConnectedAgent = &agentName
	qLog.ConnectedAgentID = &agentID

	qLog.Handled = true
}

func (call *Call) LogOnConnectAttemptFail() {
	// 1. Thread Safety (Java: synchronized)
	call.Lock()
	defer call.Unlock()

	if call.QeueuLog == nil {
		clog(LevelInfo, "Call Queue Log is empty: %s", call.UniqueId)
		return
	}

	qLog := call.QeueuLog

	now := time.Now()

	call.AttempEndTime = now

	if !call.AttempStartTime.IsZero() {
		durationSec := now.Sub(call.AttempStartTime).Seconds()
		qLog.ConnectAttemptsDuration += durationSec
	}
}

func (call *Call) LogOnInit() {

	call.Lock()
	defer call.Unlock()

	if call.QeueuLog == nil {
		call.QeueuLog = &WbpQueueLog{}
	}

	qLog := call.QeueuLog

	qLog.MediaUniqueID = call.GetMediaUniqueId()

	currentQueue, err := g.QCM.GetQueueByName(call.QueueName)

	if err == nil {
		currentQueue.RLock()
		qLog.QueueName = call.QueueName
		qLog.QueueID = currentQueue.ID
		currentQueue.RUnlock()
	}

	qLog.UniqueID = call.UniqueId
	qLog.ParentID = call.ParentId

	priority := call.Priority
	qLog.Priority = &priority

	if len(call.Skills) > 0 {
		var skillStrs []string
		for _, skillID := range call.Skills {
			skillStrs = append(skillStrs, fmt.Sprintf("%d", skillID))
		}
		joinedSkills := strings.Join(skillStrs, ",")
		qLog.Skills = &joinedSkills
	}

	if call.PreferredAgent != "" {
		prefAgent := call.PreferredAgent
		qLog.PreferredAgent = &prefAgent
	}

	qLog.ArrivalTime = time.Now()

	qLog.MediaServerID = call.ServerId
	qLog.QueueServiceID = call.InstanceID

}

// OnTerminate, çağrı sonlandığında logları hesaplar ve kaydeder.
func (call *Call) LogOnTerminate() {
	call.Lock()
	defer call.Unlock()

	if call.QeueuLog == nil {
		clog(LevelInfo, "Call Queue Log is empty: %s", call.UniqueId)
		return
	}

	qLog := call.QeueuLog

	now := time.Now()
	qLog.LeaveTime = &now

	if qLog.FinalPosition == nil {
		if !call.FinalPositionObtained {
			// Simülasyon: aidFacade.updateFinalPosition(call)
			// Eğer bu işlem Go tarafında bir metod ise buraya eklenmeli.
			// Örn: s.updateFinalPosition()
		}

		// Call struct'ındaki Position int olduğu için pointer'a çevirip atıyoruz
		pos := call.Position
		qLog.FinalPosition = &pos
	}

	qLog.QueueResult = int64(call.QueueResult)

	queueWaitDuration := qLog.calculateQueueWaitDuration(now)
	inCallDuration := qLog.calculateInCallDuration(now)

	callQueue, queueFound := g.QCM.GetQueueByName(call.QueueName)

	if queueFound != nil {
		callQueue.RLock()

		targetSLThreshold := float64(callQueue.TargetServiceLevelThreshold)
		shortAbandonedThreshold := float64(callQueue.ShortAbandonedThreshold)
		// Service Level: Handled = true VE bekleme süresi hedefin altındaysa
		qLog.ServiceLevel = qLog.Handled && (queueWaitDuration <= targetSLThreshold)
		// Short Abandoned: Handled = false VE bekleme süresi eşiğin altındaysa
		qLog.ShortAbandoned = !qLog.Handled && (queueWaitDuration <= shortAbandonedThreshold)

		call.RUnlock()
	}

	qLog.QueueWaitDuration = queueWaitDuration
	qLog.InCallDuration = inCallDuration
	qLog.ConnectAttempts = call.DistributionAttemptNumber

	clog(LevelInfo, "Saving QueueLog: %s", qLog.UniqueID)

	if err := g.QCM.InsertQueueLog(qLog); err != nil {
		clog(LevelError, "Could not save QueueLog: %v", err)
	}
}
