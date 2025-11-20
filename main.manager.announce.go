package main

import "time"

// OK
func (cm *CallManager) setupPositionAnnounce(call_UniqueId string) {

	clog(LevelDebug, "Setup starting Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.RLock()
	call_QueueName := call.QueueName
	call_PositionAnnouncePlayCount := call.PositionAnnouncePlayCount
	call.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_PositionAnnounceIsActive := currentQueue.ReportPosition
	queue_PositionAnnouncePlayCount := 1000
	queue_PositionAnnounceInitialDelay := currentQueue.PositionAnnounceInitialDelay
	queue_PositionReportHoldTimeIsActive := currentQueue.ReportHoldTime
	currentQueue.mu.RUnlock()

	if !queue_PositionAnnounceIsActive && !queue_PositionReportHoldTimeIsActive {
		clog(LevelDebug, "Queue Position Announce setup skipped , Because : queue_PositionAnnounceIsActive and queue_PositionReportHoldTimeIsActive  are false: call id :  %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_PositionAnnouncePlayCount >= int(queue_PositionAnnouncePlayCount) {
		clog(LevelDebug, "Queue Position Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	initialDelay := time.Duration(queue_PositionAnnounceInitialDelay) * time.Second

	globalScheduler.ScheduleTask(call_UniqueId, initialDelay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_PositionAnnounce)
	})

	clog(LevelDebug, "Setup scheduled Position Announce after %d seconds for Call: %s", queue_PositionAnnounceInitialDelay, call_UniqueId)

}

// OK
func (cm *CallManager) setupActionAnnounce(call_UniqueId string) {

	clog(LevelDebug, "Setup starting Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.RLock()
	call_QueueName := call.QueueName
	call_ActionAnnouncePlayCount := call.ActionAnnouncePlayCount
	call_ActionAnnounceProhibition := call.ActionAnnounceProhibition
	call.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_ActionAnnounceSoundFile := currentQueue.ActionAnnounceSoundFile
	queue_ActionAnnounceMaxPlayCount := currentQueue.ActionAnnounceMaxPlayCount
	queue_ActionAnnounceInitialDelay := currentQueue.ActionAnnounceInitialDelay
	currentQueue.mu.RUnlock()

	if queue_ActionAnnounceMaxPlayCount == 0 || call_ActionAnnounceProhibition {
		clog(LevelDebug, "Queue Action Announce is not active or Action Announce Prohibition is true , so Action Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if queue_ActionAnnounceSoundFile == "" {
		clog(LevelDebug, "Queue Action Announce Sound File is empty , so Action Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_ActionAnnouncePlayCount >= int(queue_ActionAnnounceMaxPlayCount) {
		clog(LevelDebug, "Queue Action Announce max play count reached , so Action Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	initialDelay := time.Duration(queue_ActionAnnounceInitialDelay) * time.Second

	globalScheduler.ScheduleTask(call_UniqueId, initialDelay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_ActionAnnounce)
	})

	clog(LevelDebug, "Setup scheduled Action Announce after %d seconds for Call: %s", queue_ActionAnnounceInitialDelay, call_UniqueId)

}

func (cm *CallManager) runActionPeriodicAnnounce(call_UniqueId string) {

	clog(LevelDebug, "Processing Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_PeriodicAnnounce) {
		return
	}

	call.RLock()
	call_QueueName := call.QueueName
	call_PeriodicPlayAnnounceCount := call.PeriodicPlayAnnounceCount
	call.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_PeriodicAnnounce := currentQueue.PeriodicAnnounce
	queue_PeriodicAnnounceMaxPlayCount := currentQueue.PeriodicAnnounceMaxPlayCount
	currentQueue.mu.RUnlock()

	if queue_PeriodicAnnounce == "" {
		clog(LevelDebug, "Queue Periodic Announce is empty , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_PeriodicPlayAnnounceCount >= int(queue_PeriodicAnnounceMaxPlayCount) {
		clog(LevelDebug, "Queue Periodic Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	call.Lock()
	call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_PeriodicAnnounce
	call.Unlock()

	//To DO: Buraya şimdi aksiyon yazılacak

}

// OK
func (cm *CallManager) setupPeriodicAnnounce(call_UniqueId string) {

	clog(LevelDebug, "Setup starting Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.RLock()
	call_QueueName := call.QueueName
	call_PeriodicAnnouncePlayCount := call.PeriodicPlayAnnounceCount
	call.RUnlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return
	}

	currentQueue.mu.RLock()
	queue_PeriodicAnnounce := currentQueue.PeriodicAnnounce
	queue_PeriodicAnnouncePlayCount := currentQueue.PeriodicAnnounceMaxPlayCount
	queue_PeriodicAnnounceInitialDelay := currentQueue.PeriodicAnnounceInitialDelay
	currentQueue.mu.RUnlock()

	if queue_PeriodicAnnounce == "" {
		clog(LevelDebug, "Queue Periodic Announce is empty , so Periodic Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_PeriodicAnnouncePlayCount >= int(queue_PeriodicAnnouncePlayCount) {
		clog(LevelDebug, "Queue Periodic Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	initialDelay := time.Duration(queue_PeriodicAnnounceInitialDelay) * time.Second

	globalScheduler.ScheduleTask(call_UniqueId, initialDelay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_PeriodicAnnounce)
	})

	clog(LevelDebug, "Setup scheduled Periodic Announce after %d seconds for Call: %s", queue_PeriodicAnnounceInitialDelay, call_UniqueId)

}

func isCallAvailForNextAction(call *Call, action CALL_SCHEDULED_ACTION) bool {
	//To DO : Checck Active Announce Processes for Call
	call.Lock()
	defer call.Unlock()
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
		clog(LevelWarn, "Unknown scheduled action: %s for Call: %s", action, call_UniqueId)
	}
}

func (cm *CallManager) runActionQueueTimeOut(call_UniqueId string) {

	clog(LevelDebug, "Processing Queue Timeout for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.RLock()
	call_State := call.State
	call.RUnlock()

	if call_State != CALL_STATE_InQueue {
		clog(LevelDebug, "[CALL_PROCESS_INFO] call is not waiting , so queue wait time action ignored, call id : %s", call_UniqueId)
		return
	}

	//To DO : Terminate Call Aksiyonları

}

func (cm *CallManager) runActionClientAnnounce(call_UniqueId string) {

	clog(LevelDebug, "Processing Client Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_ClientAnnounce) {
		return
	}

	call.Lock()
	call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_ClientAnnounce
	call.Unlock()
	//To DO: Buraya şimdi aksiyon yazılacak

}

/*

func (cm *CallManager) startMoh(call *Call) error {

	globalQueueManager.RLock()
	defer globalQueueManager.RUnlock()

	globalClientManager.Lock()
	defer globalClientManager.Unlock()

	currentQueue, err := globalQueueManager.GetQueueByName(call.QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return err
	}

	musicClass := currentQueue.MusicClass
	if musicClass == "" {
		musicClass = "Default"
		clog(LevelDebug, "currentQueue.MusicClass is empty. Using Default MOH Class for Call Moh Start , Call Id : %s", call.UniqueId)
	}

	ariClient, found := globalClientManager.GetClient(call.ConnectionId)

	if !found {
		clog(LevelError, "ARI Client not found for Call Moh Start , Call Application : %s , Call Id : %s", call.Application, call.UniqueId)
		return fmt.Errorf("ARI Client not found for Call Moh Start , Call Application : %s , Call Id : %s", call.Application, call.UniqueId)
	}

	ch := ariClient.Channel().Get(call.ChannelKey)

	err = ch.MOH(musicClass)
	if err != nil {
		clog(LevelError, "Error starting MOH for Call Id : %s , Error : %+v", call.UniqueId, err)
		return err
	}

	return nil
}
*/
