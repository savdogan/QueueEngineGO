package main

import "time"

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
	defer cm.mu.Unlock()

	cm.calls[call.UniqueId] = call

	logCallInfo(call)

	CustomLog(LevelInfo, "Call added: %s , Call Manager Calls Count : %d", call.UniqueId, len(cm.calls))

}

func (cm *CallManager) ProcessCall(call *Call) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.mu.Lock()
	defer cm.mu.Unlock()

	//To DO: burada konference nasıl seçeceğiz
	cm.QueueProcessStart(call)

}

func (cm *CallManager) QueueProcessStart(call *Call) {
	CustomLog(LevelDebug, "Call is in Queue Process Start : %s ", call.UniqueId)
	call.CurrentProcessName = PROCESS_NAME_queue

	//To DO: Queue Zamanlanmış Activiteleri Başlat

	/*
			  void scheduleCallActions() {
		    scheduleQueueTimeout(call);
		    scheduleAnnounce(new ClientAnnounce(call, callApplicationFactory));
		    scheduleAnnounce(new PeriodicAnnounce(call, callApplicationFactory));
		    scheduleAnnounce(new ActionAnnounce(call, callApplicationFactory));
		    scheduleAnnounce(new PositionAnnounce(call, callApplicationFactory));
		  }
	*/

	currentQueue, err := globalQueueManager.GetQueueByName(call.QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return
	}

	globalQueueManager.mu.RLock()
	defer globalQueueManager.mu.RUnlock()

	//scheduleQueueTimeout
	if currentQueue.WaitTimeout > 0 {

		delay := time.Duration(currentQueue.WaitTimeout) * time.Second

		globalScheduler.ScheduleTask(call.UniqueId, delay, func() {
			processQueueAction(call.UniqueId, CALL_SCHEDULED_ACTION_QueueTimeout)
		})
	} else {
		CustomLog(LevelDebug, "No Queue Timeout set for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
	}

	//scheduleClientAnnounce
	//To Do: Get From AID call estimation time  and aşağıdaki koşula ekle
	if currentQueue.ClientAnnounceSoundFile != "" && currentQueue.ClientAnnounceMinEstimationTime > 0 {
		globalScheduler.ScheduleTask(call.UniqueId, 1, func() {
			processQueueAction(call.UniqueId, CALL_SCHEDULED_ACTION_ClientAnnounce)
		})
	}

	//schedulePeriodicAnnounce
	cm.setupPeriodicAnnounce(call.UniqueId)

	//scheduleActionAnnounce

	//schedulePositionAnnounce

}

func isCallAvailForNextAction(call *Call, action CALL_SCHEDULED_ACTION) bool {
	//To DO : Checck Active Announce Processes for Call
	if call.CurrentCallScheduleAction == CALL_SCHEDULED_ACTION_Empty {
		return true
	}
	call.WaitingActions = append(call.WaitingActions, action)
	CustomLog(LevelDebug, "Action couldn't process, because an another action is running , running action : %s , current action : %s", call.CurrentCallScheduleAction, action)
	return false
}

func processQueueAction(callUniqueId string, action CALL_SCHEDULED_ACTION) {

	switch action {
	case CALL_SCHEDULED_ACTION_QueueTimeout:
		globalCallManager.runActionQueueTimeOut(callUniqueId)
	case CALL_SCHEDULED_ACTION_ClientAnnounce:
		globalCallManager.runActionClientAnnounce(callUniqueId)
	case CALL_SCHEDULED_ACTION_PeriodicAnnounce:
		globalCallManager.runActionPeriodicAnnounce(callUniqueId)
	default:
		CustomLog(LevelWarn, "Unknown scheduled action: %s for Call: %s", action, callUniqueId)
	}
}

func (cm *CallManager) runActionQueueTimeOut(callUniqueId string) {

	CustomLog(LevelDebug, "Processing Queue Timeout for Call: %s", callUniqueId)

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	call, found := cm.calls[callUniqueId]

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", callUniqueId)
		return
	}

	if call.State != CALL_STATE_InQueue {
		CustomLog(LevelDebug, "[CALL_PROCESS_INFO] call is not waiting , so queue wait time action ignored, call id : %s", callUniqueId)
		return
	}

	//To DO : Terminate Call Aksiyonları

}

func (cm *CallManager) runActionClientAnnounce(callUniqueId string) {

	CustomLog(LevelDebug, "Processing Client Announce for Call: %s", callUniqueId)

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	call, found := cm.calls[callUniqueId]

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", callUniqueId)
		return
	}

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_ClientAnnounce) {
		return
	}

	call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_ClientAnnounce

	//To DO: Buraya şimdi aksiyon yazılacak

}

func (cm *CallManager) runActionPeriodicAnnounce(callUniqueId string) {

	CustomLog(LevelDebug, "Processing Periodic Announce for Call: %s", callUniqueId)

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	call, found := cm.calls[callUniqueId]

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", callUniqueId)
		return
	}

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_PeriodicAnnounce) {
		return
	}

	currentQueue, err := globalQueueManager.GetQueueByName(call.QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return
	}

	globalQueueManager.mu.RLock()
	defer globalQueueManager.mu.RUnlock()

	if currentQueue.PeriodicAnnounce == "" {
		CustomLog(LevelDebug, "Queue Periodic Announce is empty , so Periodic Announce process skipped , call id : %s, queue name : %s", call.UniqueId, call.QueueName)
		return
	}

	if call.PeriodicPlayAnnounceCount >= int(currentQueue.PeriodicAnnounceMaxPlayCount) {
		CustomLog(LevelDebug, "Queue Periodic Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call.UniqueId, call.QueueName)
		return
	}

	call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_PeriodicAnnounce

	//To DO: Buraya şimdi aksiyon yazılacak

}

func (cm *CallManager) setupPeriodicAnnounce(callUniqueId string) {

	CustomLog(LevelDebug, "Setup starting Periodic Announce for Call: %s", callUniqueId)

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	call, found := cm.calls[callUniqueId]

	if !found {
		CustomLog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", callUniqueId)
		return
	}

	currentQueue, err := globalQueueManager.GetQueueByName(call.QueueName)

	if err != nil {
		CustomLog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return
	}

	globalQueueManager.mu.RLock()
	defer globalQueueManager.mu.RUnlock()

	if currentQueue.PeriodicAnnounce == "" {
		CustomLog(LevelDebug, "Queue Periodic Announce is empty , so Periodic Announce setup skipped , call id : %s, queue name : %s", call.UniqueId, call.QueueName)
		return
	}

	if call.PeriodicPlayAnnounceCount >= int(currentQueue.PeriodicAnnounceMaxPlayCount) {
		CustomLog(LevelDebug, "Queue Periodic Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call.UniqueId, call.QueueName)
		return
	}

	initialDelay := time.Duration(currentQueue.PeriodicAnnounceInitialDelay) * time.Second

	globalScheduler.ScheduleTask(call.UniqueId, initialDelay, func() {
		processQueueAction(call.UniqueId, CALL_SCHEDULED_ACTION_PeriodicAnnounce)
	})

	CustomLog(LevelDebug, "Setup scheduled Periodic Announce after %d seconds for Call: %s", currentQueue.PeriodicAnnounceInitialDelay, call.UniqueId)

}

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
