package main

import (
	"fmt"
	_ "net/http/pprof"
	"time"
)

// NewCallManager, CallManager'ın güvenli bir örneğini oluşturur ve başlatır.
func NewCallManager() *CallManager {
	return &CallManager{
		calls: make(map[string]*Call),
	}
}

// --- İŞLEM METOTLARI ---

// AddCall, yeni bir Call nesnesini yöneticinin listesine ekler.
func (cm *CallManager) AddCall(call *Call) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.mu.Lock()
	cm.calls[call.UniqueId] = call
	cm.mu.Unlock()

	logCallInfo(call, fmt.Sprintf("[ADDED]: %s , Call Manager Calls Count : %d", call.UniqueId, len(cm.calls)))

	go globalCallManager.ProcessCall(call)

}

func (cm *CallManager) ProcessCall(call *Call) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz

	//To DO: burada konference nasıl seçeceğiz
	cm.QueueProcessStart(call)

}

func (cm *CallManager) QueueProcessStart(call *Call) {

	call.mu.Lock()
	call_UniqueuId := call.UniqueId
	call.CurrentProcessName = PROCESS_NAME_queue
	call_QueueName := call.QueueName
	call_InstanceID := call.InstanceID
	call.mu.Unlock()

	CustomLog(LevelDebug, "Call is in Queue Process Start : %s ", call_UniqueuId)

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueuId, call_QueueName)
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
		CustomLog(LevelDebug, "No Queue Timeout set for Call: %s , QueueName : %s", call_UniqueuId, call_QueueName)
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

	call.mu.Lock()
	call.State = CALL_STATE_InQueue
	call.mu.Unlock()

	err = PublishNewInterActionMessage(NewCallInteraction{
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
		CustomLog(LevelError, "Publish New Interaction Message is failed for Call: %s , QueueName : %s , Error : %v", call_UniqueuId, call_QueueName, err)
		return
	}

	CustomLog(LevelDebug, "Call is in Queue Process Setup Completed : %s ", call_UniqueuId)

}

func isCallAvailForNextAction(call *Call, action CALL_SCHEDULED_ACTION) bool {
	//To DO : Checck Active Announce Processes for Call
	call.mu.Lock()
	defer call.mu.Unlock()
	if call.CurrentCallScheduleAction == CALL_SCHEDULED_ACTION_Empty {
		return true
	}
	call.WaitingActions = append(call.WaitingActions, action)
	return false
}

func processQueueAction(call_UniqueId string, action CALL_SCHEDULED_ACTION) {

	switch action {
	case CALL_SCHEDULED_ACTION_QueueTimeout:
		globalCallManager.runActionQueueTimeOut(call_UniqueId)
	case CALL_SCHEDULED_ACTION_ClientAnnounce:
		globalCallManager.runActionClientAnnounce(call_UniqueId)
	case CALL_SCHEDULED_ACTION_PeriodicAnnounce:
		globalCallManager.runActionPeriodicAnnounce(call_UniqueId)
	default:
		CustomLog(LevelWarn, "Unknown scheduled action: %s for Call: %s", action, call_UniqueId)
	}
}

func (cm *CallManager) runActionQueueTimeOut(call_UniqueId string) {

	CustomLog(LevelDebug, "Processing Queue Timeout for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.mu.RLock()
	call_State := call.State
	call.mu.RUnlock()

	if call_State != CALL_STATE_InQueue {
		CustomLog(LevelDebug, "[CALL_PROCESS_INFO] call is not waiting , so queue wait time action ignored, call id : %s", call_UniqueId)
		return
	}

	//To DO : Terminate Call Aksiyonları

}

func (cm *CallManager) runActionClientAnnounce(call_UniqueId string) {

	CustomLog(LevelDebug, "Processing Client Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_ClientAnnounce) {
		return
	}

	call.mu.Lock()
	call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_ClientAnnounce
	call.mu.Unlock()
	//To DO: Buraya şimdi aksiyon yazılacak

}

func (cm *CallManager) runActionPeriodicAnnounce(call_UniqueId string) {

	CustomLog(LevelDebug, "Processing Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_PeriodicAnnounce) {
		return
	}

	call.mu.RLock()
	call_QueueName := call.QueueName
	call_PeriodicPlayAnnounceCount := call.PeriodicPlayAnnounceCount
	call.mu.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_PeriodicAnnounce := currentQueue.PeriodicAnnounce
	queue_PeriodicAnnounceMaxPlayCount := currentQueue.PeriodicAnnounceMaxPlayCount
	currentQueue.mu.RUnlock()

	if queue_PeriodicAnnounce == "" {
		CustomLog(LevelDebug, "Queue Periodic Announce is empty , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_PeriodicPlayAnnounceCount >= int(queue_PeriodicAnnounceMaxPlayCount) {
		CustomLog(LevelDebug, "Queue Periodic Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	call.mu.Lock()
	call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_PeriodicAnnounce
	call.mu.Unlock()

	//To DO: Buraya şimdi aksiyon yazılacak

}

// OK
func (cm *CallManager) setupPeriodicAnnounce(call_UniqueId string) {

	CustomLog(LevelDebug, "Setup starting Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.mu.RLock()
	call_QueueName := call.QueueName
	call_PeriodicAnnouncePlayCount := call.PeriodicPlayAnnounceCount
	call.mu.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_PeriodicAnnounce := currentQueue.PeriodicAnnounce
	queue_PeriodicAnnouncePlayCount := currentQueue.PeriodicAnnounceMaxPlayCount
	queue_PeriodicAnnounceInitialDelay := currentQueue.PeriodicAnnounceInitialDelay
	currentQueue.mu.RUnlock()

	if queue_PeriodicAnnounce == "" {
		CustomLog(LevelDebug, "Queue Periodic Announce is empty , so Periodic Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_PeriodicAnnouncePlayCount >= int(queue_PeriodicAnnouncePlayCount) {
		CustomLog(LevelDebug, "Queue Periodic Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	initialDelay := time.Duration(queue_PeriodicAnnounceInitialDelay) * time.Second

	globalScheduler.ScheduleTask(call_UniqueId, initialDelay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_PeriodicAnnounce)
	})

	CustomLog(LevelDebug, "Setup scheduled Periodic Announce after %d seconds for Call: %s", queue_PeriodicAnnounceInitialDelay, call_UniqueId)

}

// OK
func (cm *CallManager) setupPositionAnnounce(call_UniqueId string) {

	CustomLog(LevelDebug, "Setup starting Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.mu.RLock()
	call_QueueName := call.QueueName
	call_PositionAnnouncePlayCount := call.PositionAnnouncePlayCount
	call.mu.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_PositionAnnounceIsActive := currentQueue.ReportPosition
	queue_PositionAnnouncePlayCount := 1000
	queue_PositionAnnounceInitialDelay := currentQueue.PositionAnnounceInitialDelay
	queue_PositionReportHoldTimeIsActive := currentQueue.ReportHoldTime
	currentQueue.mu.RUnlock()

	if !queue_PositionAnnounceIsActive && !queue_PositionReportHoldTimeIsActive {
		CustomLog(LevelDebug, "Queue Position Announce setup skipped , Because : queue_PositionAnnounceIsActive and queue_PositionReportHoldTimeIsActive  are false: call id :  %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_PositionAnnouncePlayCount >= int(queue_PositionAnnouncePlayCount) {
		CustomLog(LevelDebug, "Queue Position Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	initialDelay := time.Duration(queue_PositionAnnounceInitialDelay) * time.Second

	globalScheduler.ScheduleTask(call_UniqueId, initialDelay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_PositionAnnounce)
	})

	CustomLog(LevelDebug, "Setup scheduled Position Announce after %d seconds for Call: %s", queue_PositionAnnounceInitialDelay, call_UniqueId)

}

// OK
func (cm *CallManager) setupActionAnnounce(call_UniqueId string) {

	CustomLog(LevelDebug, "Setup starting Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.mu.RLock()
	call_QueueName := call.QueueName
	call_ActionAnnouncePlayCount := call.ActionAnnouncePlayCount
	call_ActionAnnounceProhibition := call.ActionAnnounceProhibition
	call.mu.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_ActionAnnounceSoundFile := currentQueue.ActionAnnounceSoundFile
	queue_ActionAnnounceMaxPlayCount := currentQueue.ActionAnnounceMaxPlayCount
	queue_ActionAnnounceInitialDelay := currentQueue.ActionAnnounceInitialDelay
	currentQueue.mu.RUnlock()

	if queue_ActionAnnounceMaxPlayCount == 0 || call_ActionAnnounceProhibition {
		CustomLog(LevelDebug, "Queue Action Announce is not active or Action Announce Prohibition is true , so Action Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if queue_ActionAnnounceSoundFile == "" {
		CustomLog(LevelDebug, "Queue Action Announce Sound File is empty , so Action Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_ActionAnnouncePlayCount >= int(queue_ActionAnnounceMaxPlayCount) {
		CustomLog(LevelDebug, "Queue Action Announce max play count reached , so Action Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	initialDelay := time.Duration(queue_ActionAnnounceInitialDelay) * time.Second

	globalScheduler.ScheduleTask(call_UniqueId, initialDelay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_ActionAnnounce)
	})

	CustomLog(LevelDebug, "Setup scheduled Action Announce after %d seconds for Call: %s", queue_ActionAnnounceInitialDelay, call_UniqueId)

}

// TerminateCall, verilen çağrıyı güvenli bir şekilde sonlandırır ve kaynakları temizler.
func (cm *CallManager) TerminateCall(call *Call, terminationReason CALL_TERMINATION_REASON) {
	// 1. Kilitleme (Java'daki call.lock() eşleniği)
	call.mu.Lock()
	defer call.mu.Unlock() // Kilidi try-finally bloğu gibi her durumda serbest bırak

	// 2. Durum Kontrolü
	if call.State == CALL_STATE_Terminated {
		CustomLog(LevelDebug, "[%s] Call already terminated. No action taken.", call.UniqueId)
		return // Zaten sonlanmışsa, hiçbir şey yapma
	}

	// 3. Loglama ve Durumu Güncelleme
	// log.info("[{}] Terminated: {}", call.getUniqueId(), call);
	CustomLog(LevelInfo, "[%s] Terminated: %s. Reason: %s", call.UniqueId, call.String(), terminationReason)

	call.State = CALL_STATE_Terminated
	call.TerminationReason = terminationReason

	// 4. Zamanlanmış Görevleri İptal Etme (Scheduler Manager kullanımı)
	// call.getScheduledActions().values().forEach(a -> a.cancel(true));

	// globalScheduler'ı kullanarak UniqueId'ye ait tüm görevleri iptal et.
	globalScheduler.CancelByCallID(call.UniqueId)
	CustomLog(LevelDebug, "[%s] All scheduled tasks cancelled.", call.UniqueId)

	// 5. Köprüyü İptal Etme (Bridge'i Temizleme)
	// if (call.getBridge() != null) { ariFacade.destroyBridge(call, call.getBridge()).subscribe(); }

	if call.Bridge != "" { // String olduğu için boş string kontrolü yapılır.
		// To DO: ARI üzerinden köprüyü yok etme mantığı buraya gelir.
		// globalClientManager üzerinden ARI istemcisine erişilmelidir.
		// Örneğin: destroyBridge(call, call.Bridge)
		CustomLog(LevelInfo, "[%s] Destroying bridge: %s", call.UniqueId, call.Bridge)
	}

	// 6. Dağıtım Denemesi Kanallarını Kapatma (Hangup)
	// call.getCurrentDistributionAttempts().keySet().forEach(c -> ariFacade.hangup(call, c).subscribe());

	// map'i kilitlemeye gerek yok, çünkü lock zaten aktif.
	// Ancak map'teki kanallar için hangup çağırmalıyız.
	for channelID := range call.CurrentDistributionAttempts {
		// To DO: ARI üzerinden kanalı kapatma mantığı buraya gelir.
		// Örneğin: hangupChannel(call.ConnectionId, channelID)
		CustomLog(LevelDebug, "[%s] Hanging up distribution channel: %s", call.UniqueId, channelID)
	}

	// --- Kilit Bırakıldı (defer call.Unlock() sayesinde) ---

	// 7. Çağrıyı Yöneticiden Kaldırma (CallManager metodu)
	cm.RemoveCall(call.UniqueId)
}

// OnAidDistributionMessage, AID'den gelen dağıtım isteğini işler.
func (cm *CallManager) OnAidDistributionMessage(call *Call, message *RedisCallDistributionMessage) {
	call.mu.Lock() // Kilitleme

	// 1. Durum Kontrolü
	if len(call.CurrentDistributionAttempts) > 0 {
		CustomLog(LevelWarn, "[%s] Got new distribution request while executing the previous one. Ignoring...", call.UniqueId)
		call.mu.Unlock()
		return
	} else if call.State == CALL_STATE_BridgedWithAgent {
		CustomLog(LevelWarn, "[%s] Got new distribution request but there is already connected agent. Dismissing distribution...", call.UniqueId)
		call.mu.Unlock()

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

	CustomLog(LevelInfo, "[%s] Trying to distribute call to %d agents with dial timeout %d sec (attempt %d)",
		call.UniqueId, len(users), dialTimeout, attemptNumber)

	// 3. Logger Başlatma
	//ds.QueueLogger.OnDistributionIterationStart()

	// Kilit serbest bırakılmadan tüm bilgiler Call struct'ına işlenmeliydi.
	// Ancak Originate çağrıları I/O bloklamalı olabileceği için, her denemeyi
	// Go rutininde başlatıp kilidi serbest bırakmamız gerekir.
	call.mu.Unlock() // Kilit serbest bırakıldı

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

	// 2. Attempt Nesnesini Oluşturma
	attempt := CallDistributionAttempt{
		UserID:        user.ID,
		Username:      user.Username,
		DialTimeout:   dialTimeout,
		AttemptNumber: attemptNumber,
		StartTime:     time.Now(),
	}

	// 3. Call Struct'ına Ekleme (Kilit gerekli)
	call.mu.Lock()
	if call.CurrentDistributionAttempts == nil {
		// Haritanın değer tipini 'attempt' değişkeninin tipine göre ayarlayın.
		// (Örn: map[string]int veya map[string]*AttemptStruct)
		call.CurrentDistributionAttempts = make(map[string]CallDistributionAttempt)
	}
	call.CurrentDistributionAttempts[channelID] = attempt
	call.mu.Unlock()

	// 4. Cache/Redis Güncelleme
	//ds.CallCache.AddOutboundChannel(channelID, call)

	CustomLog(LevelDebug, "[%s] Starting distribution attempt to agent %s (%d) on channel %s",
		call.UniqueId, user.Username, user.ID, channelID)

	// 5. ARI İşlemi (Originate)
	err := OriginateAgentChannel(call, user, dialTimeout, channelID)

	if err != nil {
		// Java'daki doOnError eşleniği: Originate başarısız olursa
		// Varsayılan olarak DIAL_STATUS_Chanunavail kullanıyoruz, ancak Java'da UNKNOWN kullanılmış.
		cm.handleFailedDistributionAttempt(call, channelID, DIAL_STATUS_Unknown)
	}

	// NOT: Java'daki .timeout() ve .subscribe() mantığı, Go'da ARI olayları dinleyicisinde (listenApp)
	// ve Scheduler'ınızda (zamanlayıcı) yönetilmelidir.
}

// handleFailedDistributionAttempt, başarısız bir denemeyi işler (Java'daki lambda fonksiyonu eşleniği).
func (cm *CallManager) handleFailedDistributionAttempt(call *Call, channelID string, status DIAL_STATUS) {
	CustomLog(LevelError, "[%s] Originate failed for channel %s. Status: %d", call.UniqueId, channelID, status)

	call.mu.Lock()
	defer call.mu.Unlock()

	// Denemeyi al, güncelle ve map'ten kaldır
	if attempt, ok := call.CurrentDistributionAttempts[channelID]; ok {
		attempt.EndTime = time.Now()
		attempt.DialStatus = status
		// Not: Map'e geri atama yapısı pointer olmadığı için, attempt'in kendisi güncellenmez.
		// Bu yüzden tüm struct'ı güncellemek gerekir:
		call.CurrentDistributionAttempts[channelID] = attempt
	}

	delete(call.CurrentDistributionAttempts, channelID)

	// Hata ayıklama veya yeniden deneme mantığı buradan devam eder.
}

/*

func (cm *CallManager) startMoh(call *Call) error {

	globalQueueManager.mu.RLock()
	defer globalQueueManager.mu.RUnlock()

	globalClientManager.mu.Lock()
	defer globalClientManager.mu.Unlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call.QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return err
	}

	musicClass := currentQueue.MusicClass
	if musicClass == "" {
		musicClass = "Default"
		CustomLog(LevelDebug, "currentQueue.MusicClass is empty. Using Default MOH Class for Call Moh Start , Call Id : %s", call.UniqueId)
	}

	ariClient, found := globalClientManager.GetClient(call.ConnectionId)

	if !found {
		CustomLog(LevelError, "ARI Client not found for Call Moh Start , Call Application : %s , Call Id : %s", call.Application, call.UniqueId)
		return fmt.Errorf("ARI Client not found for Call Moh Start , Call Application : %s , Call Id : %s", call.Application, call.UniqueId)
	}

	ch := ariClient.Channel().Get(call.ChannelKey)

	err = ch.MOH(musicClass)
	if err != nil {
		CustomLog(LevelError, "Error starting MOH for Call Id : %s , Error : %+v", call.UniqueId, err)
		return err
	}

	return nil
}
*/

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) GetCall(uniqueId string) (*Call, bool) {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	call, ok := cm.calls[uniqueId]
	return call, ok
}

// RemoveCall, bir Call nesnesini UniqueId kullanarak listeden siler.
func (cm *CallManager) RemoveCall(uniqueId string) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.calls, uniqueId)
	// CustomLog(constants.LevelInfo, "Call removed: %s", uniqueId)
}

// GetAllCalls, anlık olarak aktif tüm çağrıların bir listesini döndürür.
// Büyük çağrı sayıları için dikkatli kullanılmalıdır.
func (cm *CallManager) GetAllCalls() []*Call {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	list := make([]*Call, 0, len(cm.calls))
	for _, call := range cm.calls {
		list = append(list, call)
	}
	return list
}

// GetCount, aktif çağrı sayısını döndürür.
func (cm *CallManager) GetCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.calls)
}
