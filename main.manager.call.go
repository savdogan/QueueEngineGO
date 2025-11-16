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
	call.mu.Unlock()

	CustomLog(LevelDebug, "Call is in Queue Process Start : %s ", call_UniqueuId)

	globalQueueManager.mu.RLock()
	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)
	globalQueueManager.mu.RUnlock()

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

	globalQueueManager.mu.RLock()
	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)
	globalQueueManager.mu.RUnlock()

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

	globalQueueManager.mu.RLock()
	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)
	globalQueueManager.mu.RUnlock()

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

	globalQueueManager.mu.RLock()
	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)
	globalQueueManager.mu.RUnlock()

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
