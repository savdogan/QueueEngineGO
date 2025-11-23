package main

import (
	"database/sql"
	"fmt"
	"sync"
)

// QueueCacheManager, DB bağlantısını ve Queue önbelleğini yönetir.
type QueueCacheManager struct {
	sync.RWMutex // Cache erişimi için kilit
	DB           *sql.DB
	Cache        map[string]*Queue // Key: queue_name, Value: *WbpQueue
}

func NewQueueCacheManager(DB *sql.DB) *QueueCacheManager {
	return &QueueCacheManager{
		DB:    DB, // Artık g.DB'yi kullanıyoruz
		Cache: make(map[string]*Queue),
		// Mutex, struct tanımında kalır.
	}
}

// GetQueueByName, önce önbelleği kontrol eder, yoksa veritabanından çeker (Lazy Loading).
func (m *QueueCacheManager) GetQueueByName(queueName string) (*Queue, error) {

	//
	inQueueName := queueName

	// --- 1. Önbellek Kontrolü (Okuma Kilidi) ---
	m.RLock()
	queue, ok := m.Cache[inQueueName]
	m.RUnlock()

	if ok {
		clog(LevelDebug, "Kuyruk tanımı önbellekten okundu: %s", inQueueName)
		return queue, nil
	}

	// --- 2. Veritabanından Çekme ---
	clog(LevelDebug, "Kuyruk tanımı veritabanından çekiliyor: %s", inQueueName)

	wbpQueue := &WbpQueue{}
	query := "SELECT id, queue_name, queue_description, enabled, deleted, create_date, create_user, update_date, update_user, media_archive_period, media_delete_period, tenant_id, target_service_level, target_service_level_threshold, music_class, announce, context, timeout, monitor_format, strategy, service_level, retry, maxlen, monitor_type, report_hold_time, member_delay, member_macro, autofill, weight, leave_when_empty, join_empty, announce_frequency, min_announce_frequency, periodic_announce_frequency, relative_period_announce, announce_hold_time, announce_position, announce_round_seconds, queue_you_are_next, queue_there_are, queue_calls_waiting, queue_hold_time, queue_minutes, queue_seconds, queue_thank_you, queue_less_than, queue_report_hold, periodic_announce, set_interface_var, event_when_called, ring_in_use, timeout_restart, set_queue_var, set_queue_entry_var, event_member_status, short_abandoned_threshold, result_code_timer, result_code_timer_status, type, relax_timer, relax_timer_enabled, result_code_timer_enabled, suspend_transfer_time, music_class_on_hold, wait_timeout, periodic_announce_initial_delay, periodic_announce_max_play_count, client_announce_sound_file, client_announce_min_estimation_time, action_announce_sound_file, action_announce_initial_delay, action_announce_frequency, action_announce_max_play_count, action_announce_wait_time, action_announce_allowed_dtmf, position_announce_initial_delay, min_announced_hold_time, max_announced_hold_time, hold_time_announce_calculation_mode, queue_more_than, report_position, action_announce_wrong_dtmf_handling, migration FROM dbo.wbp_queue WHERE queue_name = @QueueName"

	row := g.DB.QueryRow(query, sql.Named("QueueName", queueName))

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

	// --- 3. Önbelleğe Yazma (Yazma Kilidi) ---
	m.Lock()
	m.Cache[inQueueName] = newQueue
	m.Unlock()

	clog(LevelDebug, "Kuyruk tanımı veritabanında okunup ön belleğe yüklendi: %s", inQueueName)

	return newQueue, nil
}

// InsertQueueLog, kuyruk logunu veritabanına kaydeder.
func (m *QueueCacheManager) InsertQueueLog(queueLog *WbpQueueLog) error {

	clog(LevelDebug, "Kuyruk logu veritabanına yazılıyor. UniqueID: %s", queueLog.UniqueID)

	if queueLog.QueueResult == 0 {
		queueLog.QueueResult = 1
	}

	// Not: [id] identity olduğu için ve [media_unique_id] default(newid()) olduğu için sorguya dahil edilmedi.
	query := `
		INSERT INTO dbo.wbp_queue_log (
			queue_name, 
			queue_id, 
			uniqueid, 
			parentid, 
			arrival_time, 
			connect_time, 
			leave_time, 
			initial_position, 
			initial_wait_estimation, 
			final_position, 
			priority, 
			skills, 
			preferred_agent, 
			handled, 
			connected_agent, 
			connected_agent_id, 
			connect_attempts, 
			queue_wait_duration, 
			connect_attempts_duration, 
			in_call_duration, 
			queue_result, 
			media_server_id, 
			queue_service_id, 
			servicelevel, 
			shortabandoned,
			media_unique_id
		) VALUES (
			@queue_name, 
			@queue_id, 
			@uniqueid, 
			@parentid, 
			@arrival_time, 
			@connect_time, 
			@leave_time, 
			@initial_position, 
			@initial_wait_estimation, 
			@final_position, 
			@priority, 
			@skills, 
			@preferred_agent, 
			@handled, 
			@connected_agent, 
			@connected_agent_id, 
			@connect_attempts, 
			@queue_wait_duration, 
			@connect_attempts_duration, 
			@in_call_duration, 
			@queue_result, 
			@media_server_id, 
			@queue_service_id, 
			@servicelevel, 
			@shortabandoned,
			@media_unique_id
		)`

	// Parametrelerin hazırlanması
	_, err := g.DB.Exec(query,
		sql.Named("queue_name", queueLog.QueueName),
		sql.Named("queue_id", queueLog.QueueID),
		sql.Named("uniqueid", queueLog.UniqueID),
		sql.Named("parentid", queueLog.ParentID),
		sql.Named("arrival_time", queueLog.ArrivalTime),
		sql.Named("connect_time", queueLog.ConnectTime), // Pointer olduğu için nil gelirse NULL yazılır
		sql.Named("leave_time", queueLog.LeaveTime),
		sql.Named("initial_position", queueLog.InitialPosition),
		sql.Named("initial_wait_estimation", queueLog.InitialWaitEstimation),
		sql.Named("final_position", queueLog.FinalPosition),
		sql.Named("priority", queueLog.Priority),
		sql.Named("skills", queueLog.Skills),
		sql.Named("preferred_agent", queueLog.PreferredAgent),
		sql.Named("handled", queueLog.Handled),
		sql.Named("connected_agent", queueLog.ConnectedAgent),
		sql.Named("connected_agent_id", queueLog.ConnectedAgentID),
		sql.Named("connect_attempts", queueLog.ConnectAttempts),
		sql.Named("queue_wait_duration", queueLog.QueueWaitDuration),
		sql.Named("connect_attempts_duration", queueLog.ConnectAttemptsDuration),
		sql.Named("in_call_duration", queueLog.InCallDuration),
		sql.Named("queue_result", queueLog.QueueResult),
		sql.Named("media_server_id", queueLog.MediaServerID),
		sql.Named("queue_service_id", queueLog.QueueServiceID),
		sql.Named("servicelevel", queueLog.ServiceLevel),
		sql.Named("shortabandoned", queueLog.ShortAbandoned),
		sql.Named("media_unique_id", queueLog.MediaUniqueID),
	)

	if err != nil {
		clog(LevelError, "Kuyruk logu insert hatası (UniqueID: %s): %v", queueLog.UniqueID, err)
		return fmt.Errorf("kuyruk logu eklenirken hata: %w", err)
	}

	clog(LevelDebug, "Kuyruk logu başarıyla eklendi. UniqueID: %s", queueLog.UniqueID)
	return nil
}
