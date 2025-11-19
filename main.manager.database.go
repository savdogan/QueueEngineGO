package main

import (
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"
)

// CloseDBConnection güncellendi
func CloseDBConnection() {
	if globalQueueManager != nil && globalQueueManager.DB != nil {
		globalQueueManager.DB.Close()
		CustomLog(LevelInfo, "SQL Server bağlantısı kapatıldı.")
	}
}

// GetQueueByName, önce önbelleği kontrol eder, yoksa veritabanından çeker (Lazy Loading).
func (m *QueueCacheManager) GetQueueByName(queueName string) (*Queue, error) {

	//
	inQueueName := queueName

	//	if inQueueName != "Yuktesti" {
	//		inQueueName = "Yuktesti"
	//	}

	// --- 1. Önbellek Kontrolü (Okuma Kilidi) ---
	m.mu.RLock()
	queue, ok := m.Cache[inQueueName]
	m.mu.RUnlock()

	if ok {
		CustomLog(LevelDebug, "Kuyruk tanımı önbellekten okundu: %s", inQueueName)
		return queue, nil
	}

	// --- 2. Veritabanından Çekme ---
	CustomLog(LevelDebug, "Kuyruk tanımı veritabanından çekiliyor: %s", inQueueName)

	wbpQueue := &WbpQueue{}
	query := "SELECT id, queue_name, queue_description, enabled, deleted, create_date, create_user, update_date, update_user, media_archive_period, media_delete_period, tenant_id, target_service_level, target_service_level_threshold, music_class, announce, context, timeout, monitor_format, strategy, service_level, retry, maxlen, monitor_type, report_hold_time, member_delay, member_macro, autofill, weight, leave_when_empty, join_empty, announce_frequency, min_announce_frequency, periodic_announce_frequency, relative_period_announce, announce_hold_time, announce_position, announce_round_seconds, queue_you_are_next, queue_there_are, queue_calls_waiting, queue_hold_time, queue_minutes, queue_seconds, queue_thank_you, queue_less_than, queue_report_hold, periodic_announce, set_interface_var, event_when_called, ring_in_use, timeout_restart, set_queue_var, set_queue_entry_var, event_member_status, short_abandoned_threshold, result_code_timer, result_code_timer_status, type, relax_timer, relax_timer_enabled, result_code_timer_enabled, suspend_transfer_time, music_class_on_hold, wait_timeout, periodic_announce_initial_delay, periodic_announce_max_play_count, client_announce_sound_file, client_announce_min_estimation_time, action_announce_sound_file, action_announce_initial_delay, action_announce_frequency, action_announce_max_play_count, action_announce_wait_time, action_announce_allowed_dtmf, position_announce_initial_delay, min_announced_hold_time, max_announced_hold_time, hold_time_announce_calculation_mode, queue_more_than, report_position, action_announce_wrong_dtmf_handling, migration FROM dbo.wbp_queue WHERE queue_name = ?"

	row := globalDB.QueryRow(query, inQueueName)

	// Tüm 80+ alanı doğru sırayla tarama (Bu kısım kritiktir ve kopyala yapıştır ile gelmelidir)
	err := row.Scan(
		&wbpQueue.ID, &wbpQueue.QueueName, &wbpQueue.QueueDescription, &wbpQueue.Enabled, &wbpQueue.Deleted,
		&wbpQueue.CreateDate, &wbpQueue.CreateUser, &wbpQueue.UpdateDate, &wbpQueue.UpdateUser,
		&wbpQueue.MediaArchivePeriod, &wbpQueue.MediaDeletePeriod, &wbpQueue.TenantID, &wbpQueue.TargetServiceLevel, &wbpQueue.TargetServiceLevelThreshold,
		&wbpQueue.MusicClass, &wbpQueue.Announce, &wbpQueue.Context, &wbpQueue.Timeout, &wbpQueue.MonitorFormat, &wbpQueue.Strategy, &wbpQueue.ServiceLevel, &wbpQueue.Retry, &wbpQueue.Maxlen,
		&wbpQueue.MonitorType, &wbpQueue.ReportHoldTime, &wbpQueue.MemberDelay, &wbpQueue.MemberMacro, &wbpQueue.Autofill, &wbpQueue.Weight, &wbpQueue.LeaveWhenEmpty, &wbpQueue.JoinEmpty,
		&wbpQueue.AnnounceFrequency, &wbpQueue.MinAnnounceFrequency, &wbpQueue.PeriodicAnnounceFrequency, &wbpQueue.RelativePeriodAnnounce, &wbpQueue.AnnounceHoldTime, &wbpQueue.AnnouncePosition,
		&wbpQueue.AnnounceRoundSeconds, &wbpQueue.QueueYouAreNext, &wbpQueue.QueueThereAre, &wbpQueue.QueueCallsWaiting, &wbpQueue.QueueHoldTime, &wbpQueue.QueueMinutes, &wbpQueue.QueueSeconds,
		&wbpQueue.QueueThankYou, &wbpQueue.QueueLessThan, &wbpQueue.QueueReportHold, &wbpQueue.PeriodicAnnounce, &wbpQueue.SetInterfaceVar, &wbpQueue.EventWhenCalled, &wbpQueue.RingInUse,
		&wbpQueue.TimeoutRestart, &wbpQueue.SetQueueVar, &wbpQueue.SetQueueEntryVar, &wbpQueue.EventMemberStatus, &wbpQueue.ShortAbandonedThreshold, &wbpQueue.ResultCodeTimer,
		&wbpQueue.ResultCodeTimerStatus, &wbpQueue.Type, &wbpQueue.RelaxTimer, &wbpQueue.RelaxTimerEnabled, &wbpQueue.ResultCodeTimerEnabled, &wbpQueue.SuspendTransferTime,
		&wbpQueue.MusicClassOnHold, &wbpQueue.WaitTimeout, &wbpQueue.PeriodicAnnounceInitialDelay, &wbpQueue.PeriodicAnnounceMaxPlayCount, &wbpQueue.ClientAnnounceSoundFile,
		&wbpQueue.ClientAnnounceMinEstimationTime, &wbpQueue.ActionAnnounceSoundFile, &wbpQueue.ActionAnnounceInitialDelay, &wbpQueue.ActionAnnounceFrequency, &wbpQueue.ActionAnnounceMaxPlayCount,
		&wbpQueue.ActionAnnounceWaitTime, &wbpQueue.ActionAnnounceAllowedDtmf, &wbpQueue.PositionAnnounceInitialDelay, &wbpQueue.MinAnnouncedHoldTime, &wbpQueue.MaxAnnouncedHoldTime,
		&wbpQueue.HoldTimeAnnounceCalculationMode, &wbpQueue.QueueMoreThan, &wbpQueue.ReportPosition, &wbpQueue.ActionAnnounceWrongDtmfHandling, &wbpQueue.Migration,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("'%s' adında kuyruk bulunamadı", inQueueName)
	}
	if err != nil {
		return nil, fmt.Errorf("kuyruk verisi okunurken hata: %w", err)
	}

	newQueue := wbpQueueToQueue(*wbpQueue)

	/*
		//To DO : aşağıdaki kısım test için bunu sil..
		newQueue.mu.Lock()
		newQueue.PeriodicAnnounceMaxPlayCount = 10
		newQueue.PeriodicAnnounceInitialDelay = 5
		newQueue.PeriodicAnnounceFrequency = 20
		newQueue.WaitTimeout = 100
		newQueue.ClientAnnounceSoundFile = "custom/client_announce"
		newQueue.ClientAnnounceMinEstimationTime = 100
		newQueue.ActionAnnounceSoundFile = "custom/action_announce"
		newQueue.PeriodicAnnounce = "custom/announcement"
		newQueue.MusicClass = "default"
		newQueue.mu.Unlock()
		//Silinecek kısmın sonu
	*/

	// --- 3. Önbelleğe Yazma (Yazma Kilidi) ---
	m.mu.Lock()
	m.Cache[inQueueName] = newQueue
	m.mu.Unlock()

	CustomLog(LevelDebug, "Kuyruk tanımı veritabanında okunup ön belleğe yüklendi: %s", inQueueName)

	return newQueue, nil
}
