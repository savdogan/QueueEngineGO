package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/CyCoreSystems/ari/v6"
	"github.com/CyCoreSystems/ari/v6/ext/play"
)

func processQueueAction(call_UniqueId string, action CALL_SCHEDULED_ACTION) {

	switch action {
	case CALL_SCHEDULED_ACTION_QueueTimeout:
		g.CM.runActionQueueTimeOut(call_UniqueId)
	case CALL_SCHEDULED_ACTION_ClientAnnounce:
		g.CM.runActionClientAnnounce(call_UniqueId)
	case CALL_SCHEDULED_ACTION_PeriodicAnnounce:
		g.CM.runActionPeriodicAnnounce(call_UniqueId)
	case CALL_SCHEDULED_ACTION_PositionAnnounce:
		g.CM.runActionPositionAnnounce(call_UniqueId)
	default:
		clog(LevelWarn, "Unknown scheduled action: %s for Call: %s", action, call_UniqueId)
	}
}

// OK
func (cm *CallManager) setupPositionAnnounce(call_UniqueId string, initial bool) error {

	clog(LevelDebug, "Setup starting Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		err := clogE(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return err
	}

	call.RLock()
	call_QueueName := call.QueueName
	call_PositionAnnouncePlayCount := call.PositionAnnouncePlayCount
	call.RUnlock()

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)

	if err != nil {
		err := clogE(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return err
	}

	currentQueue.RLock()
	queue_PositionAnnounceIsActive := currentQueue.ReportPosition
	queue_PositionAnnouncePlayCount := 1000
	queue_ScheduleTime := currentQueue.PeriodicAnnounceFrequency
	if initial {
		queue_ScheduleTime = currentQueue.PositionAnnounceInitialDelay
	}
	queue_PositionReportHoldTimeIsActive := currentQueue.ReportHoldTime
	currentQueue.RUnlock()

	if !queue_PositionAnnounceIsActive && !queue_PositionReportHoldTimeIsActive {
		clog(LevelDebug, "Queue Position Announce setup skipped , Because : queue_PositionAnnounceIsActive and queue_PositionReportHoldTimeIsActive  are false: call id :  %s, queue name : %s", call_UniqueId, call_QueueName)
		return nil
	}

	if call_PositionAnnouncePlayCount >= int(queue_PositionAnnouncePlayCount) {
		clog(LevelDebug, "Queue Position Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return nil
	}

	delay := time.Duration(queue_ScheduleTime) * time.Second

	g.SM.ScheduleTask(call_UniqueId, delay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_PositionAnnounce)
	})

	clog(LevelDebug, "Setup scheduled Position Announce after %d seconds for Call: %s", queue_ScheduleTime, call_UniqueId)

	return nil
}

// OK
func (cm *CallManager) setupActionAnnounce(call_UniqueId string, initial bool) error {

	clog(LevelDebug, "Setup starting Action Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		return clogE(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
	}

	call.RLock()
	call_QueueName := call.QueueName
	call_ActionAnnouncePlayCount := call.ActionAnnouncePlayCount
	call_ActionAnnounceProhibition := call.ActionAnnounceProhibition
	call.RUnlock()

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)

	if err != nil {
		return clogE(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
	}

	currentQueue.RLock()
	queue_ActionAnnounceSoundFile := currentQueue.ActionAnnounceSoundFile
	queue_ActionAnnounceMaxPlayCount := currentQueue.ActionAnnounceMaxPlayCount
	queue_ScheduleTime := currentQueue.ActionAnnounceInitialDelay
	if !initial {
		queue_ScheduleTime = currentQueue.ActionAnnounceFrequency
	}
	currentQueue.RUnlock()

	if queue_ActionAnnounceMaxPlayCount == 0 || call_ActionAnnounceProhibition {
		clog(LevelDebug, "Queue Action Announce is not active or Action Announce Prohibition is true , so Action Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return nil
	}

	if queue_ActionAnnounceSoundFile == "" {
		clog(LevelDebug, "Queue Action Announce Sound File is empty , so Action Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return nil
	}

	if call_ActionAnnouncePlayCount >= int(queue_ActionAnnounceMaxPlayCount) {
		clog(LevelDebug, "Queue Action Announce max play count reached , so Action Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return nil
	}

	delay := time.Duration(queue_ScheduleTime) * time.Second

	g.SM.ScheduleTask(call_UniqueId, delay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_ActionAnnounce)
	})

	clog(LevelDebug, "Setup scheduled Action Announce after %d seconds for Call: %s", queue_ScheduleTime, call_UniqueId)

	return nil

}

func (cm *CallManager) setupPeriodicAnnounce(call_UniqueId string, initial bool) error {

	clog(LevelDebug, "Setup starting Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		return clogE(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
	}

	call.RLock()
	call_QueueName := call.QueueName
	call_PeriodicAnnouncePlayCount := call.PeriodicPlayAnnounceCount
	call.RUnlock()

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)

	if err != nil {
		return clogE(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
	}

	currentQueue.RLock()
	queue_PeriodicAnnounce := currentQueue.PeriodicAnnounce
	queue_PeriodicAnnouncePlayCount := currentQueue.PeriodicAnnounceMaxPlayCount
	queue_ScheduleTime := currentQueue.PeriodicAnnounceInitialDelay
	if !initial {
		queue_ScheduleTime = currentQueue.PeriodicAnnounceFrequency
	}
	currentQueue.RUnlock()

	if queue_PeriodicAnnounce == "" {
		clog(LevelDebug, "Queue Periodic Announce is empty , so Periodic Announce setup skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return nil
	}

	if call_PeriodicAnnouncePlayCount >= int(queue_PeriodicAnnouncePlayCount) {
		clog(LevelDebug, "Queue Periodic Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return nil
	}

	delay := time.Duration(queue_ScheduleTime) * time.Second

	g.SM.ScheduleTask(call_UniqueId, delay, func() {
		processQueueAction(call_UniqueId, CALL_SCHEDULED_ACTION_PeriodicAnnounce)
	})

	clog(LevelDebug, "Setup scheduled Periodic Announce after %d seconds for Call: %s", queue_ScheduleTime, call_UniqueId)

	return nil

}

func (cm *CallManager) runActionPeriodicAnnounce(call_UniqueId string) {

	clog(LevelDebug, "Processing Periodic Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.RLock()
	call_QueueName := call.QueueName
	call_PeriodicPlayAnnounceCount := call.PeriodicPlayAnnounceCount
	call.RUnlock()

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return
	}

	currentQueue.RLock()
	queue_PeriodicAnnounce := currentQueue.PeriodicAnnounce
	queue_PeriodicAnnounceMaxPlayCount := currentQueue.PeriodicAnnounceMaxPlayCount
	currentQueue.RUnlock()

	if queue_PeriodicAnnounce == "" {
		clog(LevelDebug, "Queue Periodic Announce is empty , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if call_PeriodicPlayAnnounceCount >= int(queue_PeriodicAnnounceMaxPlayCount) {
		clog(LevelDebug, "Queue Periodic Announce max play count reached , so Periodic Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_PeriodicAnnounce) {
		return
	}

	announceList := []string{
		queue_PeriodicAnnounce,
	}

	go g.CM.startAnnounce(announceList, call, false, CALL_SCHEDULED_ACTION_PeriodicAnnounce)

}

func (cm *CallManager) runActionPositionAnnounce(call_UniqueId string) {

	clog(LevelDebug, "Processing Position Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.RLock()
	call_QueueName := call.QueueName
	call.RUnlock()

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return
	}

	as := &AnnouncementService{}

	currentQueue.RLock()
	as.queue_AnnouncePosition = currentQueue.AnnouncePosition
	if as.queue_AnnouncePosition != "yes" {
		clog(LevelDebug, "Queue Position Announce is not 'yes' , so Position Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		currentQueue.RUnlock()
		return
	}
	as.queue_FirstPositionAnnounce = currentQueue.QueueYouAreNext
	as.queue_HoldTime = currentQueue.ReportHoldTime
	as.queue_HoldTimeAnnounceFile = currentQueue.QueueHoldTime
	as.queue_HoldTimeMode = HoldTimeAnnounceMode(currentQueue.HoldTimeAnnounceCalculationMode)
	as.queue_LessThanFile = currentQueue.QueueLessThan
	as.queue_MoreThanFile = currentQueue.QueueMoreThan
	as.queue_MaxAnnouncedHoldTime = int(currentQueue.MaxAnnouncedHoldTime)
	as.queue_MinAnnouncedHoldTime = int(currentQueue.MinAnnouncedHoldTime)
	as.queue_PostPositionAnnounce = currentQueue.QueueCallsWaiting
	as.queue_PrePositionAnnounce = currentQueue.QueueThereAre
	currentQueue.RUnlock()

	as.call_Position = 21
	as.call_EstimationTime = 132
	as.call_SpellOutLanguage = LANGUAGE_TR

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_PositionAnnounce) {
		return
	}

	announceList := append(as.buildPositionFileSequence(985), as.buildHoldTimeFileSequence(96)...)

	clog(LevelDebug, "[Değerler]%+v", announceList)

	go g.CM.startAnnounce(announceList, call, false, CALL_SCHEDULED_ACTION_PositionAnnounce)

}

func (cm *CallManager) runActionQueueTimeOut(call_UniqueId string) {

	clog(LevelDebug, "Processing Queue Timeout for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.Lock()

	if call.State != CALL_STATE_InQueue {
		clog(LevelDebug, "[CALL_PROCESS_INFO] call is not waiting , so queue wait time action ignored, call id : %s", call_UniqueId)
		call.Unlock()
		return
	}

	clog(LevelWarn, "[%s] Queue wait time is reached.", call.UniqueId)

	ariClient, found := g.ACM.GetClient(call.ConnectionName)

	call.SetTerminationReason(CALL_TERMINATION_REASON_QueueWaitTimeReached)

	call.Unlock()

	g.CM.cancelCallActions(call, false)

	if found {
		g.CM.hangupChannel(call.ChannelId, call.UniqueId, ariClient, "")
	}
}

func (cm *CallManager) runActionClientAnnounce(call_UniqueId string) {
	//queue_ClientAnnounceSoundFile

	clog(LevelDebug, "Processing Client Announce for Call: %s", call_UniqueId)

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelInfo, "[CALL_PROCESS_ERROR] call is not found , call id : %s", call_UniqueId)
		return
	}

	call.RLock()
	call_QueueName := call.QueueName
	call.RUnlock()

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call.UniqueId, call.QueueName)
		return
	}

	currentQueue.RLock()
	queue_ClientAnnounceSoundFile := currentQueue.ClientAnnounceSoundFile
	currentQueue.RUnlock()

	if queue_ClientAnnounceSoundFile == "" {
		clog(LevelDebug, "Queue Client Announce is empty , so Client Announce process skipped , call id : %s, queue name : %s", call_UniqueId, call_QueueName)
		return
	}

	if !isCallAvailForNextAction(call, CALL_SCHEDULED_ACTION_ClientAnnounce) {
		return
	}

	announceList := []string{
		queue_ClientAnnounceSoundFile,
	}

	go g.CM.startAnnounce(announceList, call, false, CALL_SCHEDULED_ACTION_ClientAnnounce)

}

func (cm *CallManager) startMoh(call *Call, callLocked bool) error {

	if !callLocked {
		call.RLock()
	}

	call_IsExistsActiveMOH := call.IsExistsActiveMOH
	call_ConnectionName := call.ConnectionName
	call_UniqueId := call.UniqueId
	call_ChannelKey := call.ChannelKey
	call_QueueName := call.QueueName

	if !callLocked {
		call.RUnlock()
	}

	if call_IsExistsActiveMOH {
		clog(LevelDebug, "call_IsExistsActiveMOH Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return nil
	}

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)
	currentQueue.RLock()
	currentQueue_MusicClass := currentQueue.MusicClass
	currentQueue.RUnlock()

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return err
	}

	musicClass := currentQueue_MusicClass
	if musicClass == "" {
		musicClass = "Default"
		clog(LevelDebug, "currentQueuesicClass is empty. Using Default MOH Class for Call Moh Start , Call Id : %s", call_UniqueId)
	}

	ariClient, found := g.ACM.GetClient(call_ConnectionName)

	if !found {
		clog(LevelError, "ARI Client not found for Call Moh Start , Call Id : %s", call_UniqueId)
		return fmt.Errorf("ARI Client not found for Call Moh Start , Call Id : %s", call_UniqueId)
	}

	ch := ariClient.Channel().Get(call_ChannelKey)

	if ch.ID() == "" {
		clog(LevelError, "[MOH_ERROR]Error starting MOH for Call Id : %s , Channel is not found", call_UniqueId)
		return err
	}

	err = ch.MOH(musicClass)
	if err != nil {
		clog(LevelError, "[MOH_ERROR]Error starting MOH for Call Id : %s , Error : %+v", call_UniqueId, err)
		return err
	}

	if !callLocked {
		call.Lock()
	}

	call.IsExistsActiveMOH = true

	if !callLocked {
		call.Unlock()
	}

	return nil
}

func (cm *CallManager) stopMoh(call *Call, callLocked bool) error {

	if !callLocked {
		call.RLock()
	}

	call_ConnectionName := call.ConnectionName
	call_UniqueId := call.UniqueId
	call_ChannelKey := call.ChannelKey
	call_QueueName := call.QueueName
	call_IsExistsActiveMOH := call.IsExistsActiveMOH

	if !callLocked {
		call.RUnlock()
	}

	if !call_IsExistsActiveMOH {
		clog(LevelDebug, "call_IsExistsActiveMOH false Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return nil
	}

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)
	currentQueue.RLock()
	currentQueue_MusicClass := currentQueue.MusicClass
	currentQueue.RUnlock()

	if err != nil {
		clog(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
		return err
	}

	musicClass := currentQueue_MusicClass
	if musicClass == "" {
		musicClass = "Default"
		clog(LevelDebug, "currentQueuesicClass is empty. Using Default MOH Class for Call Moh Start , Call Id : %s", call_UniqueId)
	}

	ariClient, found := g.ACM.GetClient(call_ConnectionName)

	if !found {
		clog(LevelError, "ARI Client not found for Call Moh Start , Call Id : %s", call_UniqueId)
		return fmt.Errorf("ARI Client not found for Call Moh Start , Call Id : %s", call_UniqueId)
	}

	ch := ariClient.Channel().Get(call_ChannelKey)

	err = ch.StopMOH()
	if err != nil {
		clog(LevelError, "Error stoping MOH for Call Id : %s , Error : %+v", call_UniqueId, err)
		return err
	}

	if !callLocked {
		call.Lock()
	}

	clog(LevelDebug, "MOH stoped for Call Id : %s", call_UniqueId)
	call.IsExistsActiveMOH = false

	if !callLocked {
		call.Unlock()
	}

	return nil
}

func (cm *CallManager) startAnnounce(announceFile []string, call *Call, callLocked bool, action CALL_SCHEDULED_ACTION) error {

	call_UniqueId, call_ConnectionName, call_QueueName, _, _ := call.getMainInformations(callLocked)

	currentQueue, err := g.QCM.GetQueueByName(call_QueueName)
	if err != nil {
		return clogE(LevelError, "Queue not found for Call: %s , QueueName : %s", call_UniqueId, call_QueueName)
	}
	currentQueue.RLock()
	currentQueue_MusicClass := currentQueue.MusicClass
	currentQueue.RUnlock()

	musicClass := currentQueue_MusicClass
	if musicClass == "" {
		musicClass = "Default"
		clog(LevelDebug, "currentQueuesicClass is empty. Using Default MOH Class for Call Moh Start , Call Id : %s", call_UniqueId)
	}

	ariClient, found := g.ACM.GetClient(call_ConnectionName)

	if !found {
		return clogE(LevelError, "ARI Client not found for Call Moh Start , Call Id : %s", call_UniqueId)
	}

	ch := ariClient.Channel().Get(ari.NewKey(ari.ChannelKey, call_UniqueId))

	if ch.ID() == "" {
		return clogE(LevelError, "[MOH_ERROR]Error stoping MOH for Call Id : %s , Channel is not found", call_UniqueId)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	call.setActionProperties(callLocked, cancelFunc, action)

	var session play.Session

	if len(announceFile) == 1 {
		session = play.Play(ctx, ch, play.URI("sound:"+announceFile[0]))
	} else {

		playFileList := make([]play.OptionFunc, len(announceFile))

		for index, file := range announceFile {
			playFileList[index] = play.URI("sound:" + file)
		}

		session = play.Play(ctx, ch, playFileList...)
	}

	if err := session.Err(); err != nil {
		clog(LevelError, "[ANNOUNCE_SESSION_ERROR] (-1-) action : %s,  call : %s, announce : %s, failed to play announce %+v ", action, call_UniqueId, announceFile, err)
		go cm.onEndAction(call_UniqueId, action)
		return err
	}

	go func() {

		result, err := session.Result()
		if err != nil {
			clog(LevelError, "[ANNOUNCE_SESSION_ERROR] (-2-) action : %s,  call : %s , result :  %+v , error : %+v ", action, call_UniqueId, result, err)
		} else {
			clog(LevelDebug, "[ANNOUNCE_SESSION_RESULT] (-3-) action : %s,  call : %s , result :  %+v", action, call_UniqueId, result)
		}

		go cm.onEndAction(call_UniqueId, action)

	}()

	return nil
}

func (cm *CallManager) onEndAction(call_UniqueId string, action CALL_SCHEDULED_ACTION) {

	call, found := cm.GetCall(call_UniqueId)

	if !found {
		clog(LevelError, "[CALL_NOT_FOUND] in processNextAnnounce, call : %s", call_UniqueId)
		return
	}

	call.Lock()
	defer call.Unlock()

	if call.ActiveActionCancelFunc != nil {
		call.ActiveActionCancelFunc()
	}

	call.ActiveActionCancelFunc = nil

	if call.State == CALL_STATE_InQueue {

		go cm.reScheduleAction(call_UniqueId, action)

		if len(call.WaitingActions) > 0 {
			action := call.WaitingActions[0]
			call.WaitingActions = call.WaitingActions[1:]
			call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_Empty
			call.PreparingCallScheduleAction = action
			go processQueueAction(call_UniqueId, action)
		} else {
			call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_Empty
			call.PreparingCallScheduleAction = CALL_SCHEDULED_ACTION_Empty
			call.IsExistsActiveMOH = false
			go cm.startMoh(call, false)
		}

	} else {
		g.SM.CancelByCallID(call.UniqueId)
		call.WaitingActions = []CALL_SCHEDULED_ACTION{}
	}
}

func isCallAvailForNextAction(call *Call, action CALL_SCHEDULED_ACTION) bool {
	//To DO : Checck Active Announce Processes for Call
	call.Lock()
	defer call.Unlock()

	if call.PreparingCallScheduleAction == action {
		call.CurrentCallScheduleAction = action
		call.PreparingCallScheduleAction = CALL_SCHEDULED_ACTION_Empty
		return true
	}

	if call.CurrentCallScheduleAction == CALL_SCHEDULED_ACTION_Empty && call.PreparingCallScheduleAction == CALL_SCHEDULED_ACTION_Empty {
		call.CurrentCallScheduleAction = action
		return true
	}

	call.WaitingActions = append(call.WaitingActions, action)

	return false
}

// --- Yardımcı Fonksiyonlar ---
func (s *AnnouncementService) SayPosition(pos int, lang LANGUAGE) []string {

	parts := formatNumber(pos, lang)
	for index, part := range parts {
		parts[index] = fmt.Sprintf("%s/digits/%s", strings.ToLower(string(lang)), part)
	}
	return parts
}

func (s *AnnouncementService) SaySeconds(value int, lang LANGUAGE) []string {

	parts := formatNumber(value, lang)
	for index, part := range parts {
		parts[index] = fmt.Sprintf("%s/digits/%s", strings.ToLower(string(lang)), part)
	}

	unit := ""

	switch lang {
	case LANGUAGE_EN:
		if value == 1 {
			unit = "second"
		} else {
			unit = "seconds"
		}
	case LANGUAGE_TR:
		unit = "second"
	}

	return append(parts, fmt.Sprintf("%s/%s", strings.ToLower(string(lang)), unit))
}

func (s *AnnouncementService) SayMinutes(value int, lang LANGUAGE) []string {

	parts := formatNumber(value, lang)
	for index, part := range parts {
		parts[index] = fmt.Sprintf("%s/digits/%s", strings.ToLower(string(lang)), part)
	}

	unit := ""

	switch lang {
	case LANGUAGE_EN:
		if value == 1 {
			unit = "minute"
		} else {
			unit = "minutes"
		}
	case LANGUAGE_TR:
		unit = "minute"
	}

	return append(parts, fmt.Sprintf("%s/%s", strings.ToLower(string(lang)), unit))

}

func (s *AnnouncementService) BuildFileSequence() ([]string, error) {
	var result []string

	if s.queue_AnnouncePosition != "" {
		result = s.buildPositionFileSequence(s.call_Position)
		if s.queue_HoldTime {
			holdFiles := s.buildHoldTimeFileSequence(s.call_EstimationTime)
			result = append(result, holdFiles...)
		}
	} else if s.queue_HoldTime {
		result = s.buildHoldTimeFileSequence(s.call_EstimationTime)
	} else {
		return nil, errors.New("could not build file sequence for disabled position/holdtime announce")
	}

	return result, nil
}

func (s *AnnouncementService) buildPositionFileSequence(position int) []string {
	result := make([]string, 0)

	// Pozisyon 1 ise ve özel dosya varsa
	if position == 1 && isNotBlank(s.queue_FirstPositionAnnounce) {
		result = append(result, s.queue_FirstPositionAnnounce)
		return result
	} else if isNotBlank(s.queue_PrePositionAnnounce) {
		result = append(result, s.queue_PrePositionAnnounce)
	}

	// Sayı okuma dosyalarını ekle
	result = append(result, s.SayPosition(position, s.call_SpellOutLanguage)...)

	if isNotBlank(s.queue_PostPositionAnnounce) {
		result = append(result, s.queue_PostPositionAnnounce)
	}

	return result
}

func (s *AnnouncementService) buildHoldTimeFileSequence(holdTime int) []string {
	result := make([]string, 0)

	// Pointer kullanarak result slice'ını değiştiriyoruz (Java'daki side-effect gibi)
	s.addFile(&result, s.queue_HoldTimeAnnounceFile, "hold time announce")

	// Limit kontrolü
	holdTime = s.limitHoldTime(holdTime, &result)

	mode := s.queue_HoldTimeMode

	if holdTime < 60 || mode == ModeSecondsBy10Sec {
		secs := s.roundSeconds(holdTime, 10.0)
		result = append(result, s.SaySeconds(secs, s.call_SpellOutLanguage)...)
	} else if mode == ModeSecondsBy30Sec {
		secs := s.roundSeconds(holdTime, 30.0)
		result = append(result, s.SaySeconds(secs, s.call_SpellOutLanguage)...)
	} else if mode == ModeMinutesBy1Min {
		mins := s.secondsToMinutes(holdTime)
		result = append(result, s.SayMinutes(mins, s.call_SpellOutLanguage)...)
	} else {
		s.mixMinutesAndSeconds(holdTime, &result, mode)
	}

	return result
}

func (s *AnnouncementService) mixMinutesAndSeconds(holdTime int, filenames *[]string, mode HoldTimeAnnounceMode) {
	minutes := holdTime / 60
	seconds := holdTime % 60

	switch mode {
	case ModeMinutesBy10Sec:
		seconds = s.roundSeconds(seconds, 10.0)
	case ModeMinutesBy30Sec:
		seconds = s.roundSeconds(seconds, 30.0)
	}

	if seconds == 60 {
		// Saniyeler 60'a yuvarlandıysa dakikayı bir artır
		files := s.SayMinutes(minutes+1, s.call_SpellOutLanguage)
		*filenames = append(*filenames, files...)
	} else {
		// Dakikayı ekle
		filesMins := s.SayMinutes(minutes, s.call_SpellOutLanguage)
		*filenames = append(*filenames, filesMins...)

		// Kalan saniye varsa ekle
		if seconds > 0 {
			filesSecs := s.SaySeconds(seconds, s.call_SpellOutLanguage)
			*filenames = append(*filenames, filesSecs...)
		}
	}
}

func (s *AnnouncementService) limitHoldTime(holdTime int, fileSequence *[]string) int {
	// Java'da null kontrolü vardı, Go'da pointer nil check yapıyoruz
	if s.queue_MinAnnouncedHoldTime > 0 && holdTime < s.queue_MinAnnouncedHoldTime {
		holdTime = s.queue_MinAnnouncedHoldTime
		log.Printf("Limiting estimation hold time up to minimal value: %d", holdTime)
		s.addFile(fileSequence, s.queue_LessThanFile, "less than")
	} else if s.queue_MaxAnnouncedHoldTime > 0 && holdTime > s.queue_MaxAnnouncedHoldTime {
		holdTime = s.queue_MaxAnnouncedHoldTime
		log.Printf("Limiting estimation hold time down to maximal value: %d", holdTime)
		s.addFile(fileSequence, s.queue_MoreThanFile, "more than")
	}
	return holdTime
}

func (s *AnnouncementService) roundSeconds(value int, roundBase float64) int {
	// Java: Math.round(value / base) * base
	val := float64(value)
	return int(math.Round(val/roundBase) * roundBase)
}

func (s *AnnouncementService) secondsToMinutes(value int) int {
	val := float64(value)
	return int(math.Round(val / 60.0))
}

func (s *AnnouncementService) addFile(fileSequence *[]string, filename string, description string) {
	if isNotBlank(filename) {
		*fileSequence = append(*fileSequence, filename)
	} else {
		log.Printf("WARN: Missing empty file: %s", description)
	}
}

func isNotBlank(s string) bool {
	return len(strings.TrimSpace(s)) > 0
}

func formatNumber(value int, language LANGUAGE) []string {
	var parts []string

	if value == 0 {
		return []string{"0"}
	}

	// --- BİNLER BASAMAĞI ---
	if value >= 1000 {
		thousands := value / 1000

		// TR/EN FARKI BURADA:
		// İngilizcede her zaman "one thousand", "two thousand" denir.
		// Türkçede "bir bin" denmez, sadece "bin" denir. Ama "iki bin" denir.
		if language == LANGUAGE_EN || thousands > 1 {
			parts = append(parts, formatNumber(thousands, language)...)
		}

		parts = append(parts, "thousand") // Dosya adı: "thousand.wav" veya "bin.wav"
		value %= 1000
	}

	// --- YÜZLER BASAMAĞI ---
	if value >= 100 {
		hundreds := value / 100

		// TR/EN FARKI BURADA:
		// İngilizce: "one hundred"
		// Türkçe: Sadece "yüz" (bir yüz denmez)
		if language == LANGUAGE_EN || hundreds > 1 {
			parts = append(parts, formatNumber(hundreds, language)...)
		}

		parts = append(parts, "hundred") // Dosya adı: "hundred.wav" veya "yuz.wav"
		value %= 100
	}

	// --- ONLAR BASAMAĞI ---
	if value >= 20 {
		tens := (value / 10) * 10
		parts = append(parts, fmt.Sprintf("%d", tens))
		value %= 10
	}

	// --- BİRLER BASAMAĞI ---
	if value > 0 {
		parts = append(parts, fmt.Sprintf("%d", value))
	}

	return parts
}
