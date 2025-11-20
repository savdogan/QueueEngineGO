package main

import (
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/CyCoreSystems/ari/v6"
)

// NewCallManager, CallManager'ın güvenli bir örneğini oluşturur ve başlatır.
func NewCallManager() *CallManager {
	return &CallManager{
		calls:       make(map[string]*Call),
		outChannels: make(map[string]*ari.ChannelHandle),
	}
}

// --- İŞLEM METOTLARI ---

// processNewInboundCall, yeni bir Call nesnesini yöneticinin listesine ekler.
func (cm *CallManager) processNewInboundCall(call *Call) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.mu.Lock()
	cm.calls[call.UniqueId] = call
	cm.mu.Unlock()

	logCallInfo(call, fmt.Sprintf("[ADDED]: %s , Call Manager Calls Count : %d", call.UniqueId, len(cm.calls)))

	go globalCallManager.processCall(call)

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

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueuId, call_QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_WaitTimeout := currentQueue.WaitTimeout
	queue_ClientAnnounceMinEstimationTime := currentQueue.ClientAnnounceMinEstimationTime
	queue_ClientAnnounceSoundFile := currentQueue.ClientAnnounceSoundFile
	currentQueue.mu.RUnlock()

	//scheduleQueueTimeout
	if queue_WaitTimeout > 0 {

		delay := time.Duration(queue_WaitTimeout) * time.Second

		globalScheduler.ScheduleTask(call_UniqueuId, delay, func() {
			processQueueAction(call_UniqueuId, CALL_SCHEDULED_ACTION_QueueTimeout)
		})
	} else {
		clog(LevelDebug, "No Queue Timeout set for Call: %s , QueueName : %s", call_UniqueuId, call_QueueName)
	}

	//scheduleClientAnnounce
	//To Do: Get From AID call estimation time  and aşağıdaki koşula ekle
	if queue_ClientAnnounceSoundFile != "" && queue_ClientAnnounceMinEstimationTime > 0 {

		globalScheduler.ScheduleTask(call_UniqueuId, 1, func() {
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
	// 1. Kilitleme (Java'daki call.lock() eşleniği)
	call.RLock()
	call_UniqueId := call.UniqueId
	call_state := call.State
	call_InstanceID := call.InstanceID
	call_QueueName := call.QueueName
	call_Bridge := call.Bridge
	call_BridgedChannel := call.BridgedChannel
	call_connectionName := call.ConnectionName
	call_ConnectedAgentChannelID := call.ConnectedAgentChannelID
	call.RUnlock() // Kilidi try-finally bloğu gibi her durumda serbest bırak

	// 2. Durum Kontrolü

	call.Lock()
	if call_state == CALL_STATE_Terminated {
		clog(LevelDebug, "[%s] Call already terminated. No action taken.", call_UniqueId)
		call.Unlock()
		return // Zaten sonlanmışsa, hiçbir şey yapma
	}
	call.State = CALL_STATE_Terminated
	call.TerminationReason = terminationReason
	call.Unlock()

	// 3. Loglama ve Durumu Güncelleme
	// log.info("[{}] Terminated: {}", call.getUniqueId(), call);

	clog(LevelInfo, "[%s] Terminated. Reason: %s", call_UniqueId, terminationReason)

	// 4. Zamanlanmış Görevleri İptal Etme (Scheduler Manager kullanımı)
	// call.getScheduledActions().values().forEach(a -> a.cancel(true));

	// globalScheduler'ı kullanarak UniqueId'ye ait tüm görevleri iptal et.
	globalScheduler.CancelByCallID(call_UniqueId)
	clog(LevelDebug, "[%s] All scheduled tasks cancelled.", call_UniqueId)

	logAnyInfo(call, "[TERMINATED]")

	ariClient, found := globalClientManager.GetClient(call_connectionName)
	if found {

		terminateBridge(call_Bridge, call_UniqueId, ariClient)

		terminateChannel(call_BridgedChannel, call_UniqueId, ariClient)

		terminateChannel(call_ConnectedAgentChannelID, call_UniqueId, ariClient)

	} else {
		clog(LevelError, "Ari Client not found in terminate call %s", call_UniqueId)
	}

	cm.RemoveCall(call_UniqueId)

	go func() {
		PublishInteractionStateMessage(InteractionState{
			InteractionID:        call_UniqueId,
			InstanceID:           call_InstanceID,
			RelatedAgentUsername: "",
			State:                string(CALL_STATE_Terminated),
			QueueName:            call_QueueName,
		})
	}()

}

// onAidDistributionMessage, AID'den gelen dağıtım isteğini işler.
func (cm *CallManager) onAidDistributionMessage(call *Call, message *RedisCallDistributionMessage) {
	call.Lock() // Kilitleme

	// 1. Durum Kontrolü
	if call.IsCallInDistribution {
		clog(LevelWarn, "[%s] Got new distribution request while executing the previous one. Ignoring...", call.UniqueId)
		call.Unlock()
		return
	} else if call.State == CALL_STATE_BridgedWithAgent {
		clog(LevelWarn, "[%s] Got new distribution request but there is already connected agent. Dismissing distribution...", call.UniqueId)
		call.Unlock()

		// Kilit dışında Facade çağrısı
		//ds.AidFacade.PublishDistributionDismissedMessage(call.ParentId, call.QueueName)
		return
	}

	// 2. Hazırlık ve Sayaç Artırma (Atomic işlemler, sadeleştirilmiş struct'ta manuel yapılır)
	users := message.Users
	dialTimeout := 5000 //call.QueueTimeout

	// AtomicLong.incrementAndGet() eşleniği
	call.DistributionAttemptNumber++
	attemptNumber := call.DistributionAttemptNumber

	clog(LevelInfo, "[%s] Trying to distribute call to %d agents with dial timeout %d sec (attempt %d)",
		call.UniqueId, len(users), dialTimeout, attemptNumber)

	// 3. Logger Başlatma
	//ds.QueueLogger.OnDistributionIterationStart()

	// Kilit serbest bırakılmadan tüm bilgiler Call struct'ına işlenmeliydi.
	// Ancak Originate çağrıları I/O bloklamalı olabileceği için, her denemeyi
	// Go rutininde başlatıp kilidi serbest bırakmamız gerekir.
	call.Unlock() // Kilit serbest bırakıldı

	// 4. Dağıtım Denemelerini Başlatma (Her kullanıcı için Go rutini)

	// Her deneme ayrı bir Go rutininde çalışır.
	go cm.startDistributionAttempt(call, message.Users[0], attemptNumber, dialTimeout)

	// 5. Redis Güncelleme
	// Not: Tüm startDistributionAttempt işlemleri tamamlanmadan bu çağrılabilir.
	// Eğer tüm attempts'ların haritaya eklenmesi gerekiyorsa, yukarıdaki for döngüsü bloklamalı olmalıdır.
	//ds.RedisCallAdapter.UpdateDistributionAttempt(call)
}

func (cm *CallManager) startDistributionAttempt(call *Call, user UserDist, attemptNumber int64, dialTimeout int) {
	// 1. Kanal ID Oluşturma
	channelID := fmt.Sprintf("%s-agent-%s-%d", call.UniqueId, user.Username, attemptNumber)

	// 3. Call Struct'ına Ekleme (Kilit gerekli)
	call.Lock()
	call.IsCallInDistribution = true
	call.ConnectedAgentId = user.ID
	call.Unlock()

	// 4. Cache/Redis Güncelleme
	//ds.CallCache.AddOutboundChannel(channelID, call)

	clog(LevelDebug, "[%s] Starting distribution attempt to agent %s (%d) on channel %s",
		call.UniqueId, user.Username, user.ID, channelID)

	// 5. ARI İşlemi (Originate)
	err := OriginateAgentChannel(call, user, dialTimeout, channelID)

	if err != nil {
		clearCallProperties(call, nil)
	}

	// NOT: Java'daki .timeout() ve .subscribe() mantığı, Go'da ARI olayları dinleyicisinde (listenApp)
	// ve Scheduler'ınızda (zamanlayıcı) yönetilmelidir.
}

func terminateChannel(channelId string, callUniqueId string, ariClient ari.Client) {
	if channelId != "" {
		agentChannelHandle := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, channelId))
		if agentChannelHandle.ID() != "" {
			clog(LevelTrace, "Agent channel(%s) is deleting for the call (%s)", channelId, callUniqueId)
			err := agentChannelHandle.Hangup()
			if err != nil {
				clog(LevelError, "Error while agentChannelHandle Hangup(ConnectedAgentChannelID) the agent channel : %s , error : %+v ", channelId, err)
			}
		} else {
			clog(LevelTrace, "function agentChannelHandle.ID() did not return any channel handle for the channel id (%s) of the call (%s)", channelId, callUniqueId)
		}
	} else {
		clog(LevelTrace, "Any channel is found for hangup , call id : %s", callUniqueId)
	}
}

func terminateBridge(bridgeChannelId string, callUniqueId string, ariClient ari.Client) {
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
