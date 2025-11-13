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

// InitDBConnection, SQL Server bağlantısını kurar ve globalDB'yi ayarlar.
func InitDBConnection(instance, dbName, user string) error {

	// Windows kimlik doğrulaması (Integrated Security) için bağlantı dizesi
	connString := fmt.Sprintf("server=%s;database=%s;integrated security=true", instance, dbName)

	// !!! SÜRÜCÜ ADI "mssql" OLARAK DEĞİŞTİ !!!
	db, err := sql.Open("mssql", connString)
	if err != nil {
		return fmt.Errorf("SQL Server'a bağlanırken hata: %w", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("SQL Server bağlantı testi başarısız: %w", err)
	}

	// Temel bağlantıyı global alana atama
	globalDB = db

	CustomLog(LevelInfo, "SQL Server bağlantısı başarılı.")
	return nil
}

// InitQueueManager: Sadece QueueManager nesnesini başlatır.
func InitQueueManager() {
	// DB bağlantısı zaten globalDB'de olmalı.
	if globalDB == nil {
		CustomLog(LevelFatal, "QueueManager başlatılamadı: Global DB bağlantısı mevcut değil.")
		return
	}

	// Queue Manager'ı globalDB bağlantısını kullanarak başlat
	globalQueueManager = &QueueCacheManager{
		DB:    globalDB, // Artık globalDB'yi kullanıyoruz
		Cache: make(map[string]*WbpQueue),
		// Mutex, struct tanımında kalır.
	}
	CustomLog(LevelInfo, "Queue önbellek yöneticisi başlatıldı.")
}

// GetQueueByName, önce önbelleği kontrol eder, yoksa veritabanından çeker (Lazy Loading).
func (m *QueueCacheManager) GetQueueByName(queueName string) (*WbpQueue, error) {
	if m.DB == nil {
		return nil, fmt.Errorf("veri tabanı bağlantısı mevcut değil")
	}

	// --- 1. Önbellek Kontrolü (Okuma Kilidi) ---
	m.mu.RLock()
	queue, ok := m.Cache[queueName]
	m.mu.RUnlock()

	if ok {
		CustomLog(LevelDebug, "Kuyruk tanımı önbellekten okundu: %s", queueName)
		return queue, nil
	}

	// --- 2. Veritabanından Çekme ---
	CustomLog(LevelDebug, "Kuyruk tanımı veritabanından çekiliyor: %s", queueName)

	queue = &WbpQueue{}
	query := "SELECT id, queue_name, queue_description, enabled, deleted, create_date, create_user, update_date, update_user, media_archive_period, media_delete_period, tenant_id, target_service_level, target_service_level_threshold, music_class, announce, context, timeout, monitor_format, strategy, service_level, retry, maxlen, monitor_type, report_hold_time, member_delay, member_macro, autofill, weight, leave_when_empty, join_empty, announce_frequency, min_announce_frequency, periodic_announce_frequency, relative_period_announce, announce_hold_time, announce_position, announce_round_seconds, queue_you_are_next, queue_there_are, queue_calls_waiting, queue_hold_time, queue_minutes, queue_seconds, queue_thank_you, queue_less_than, queue_report_hold, periodic_announce, set_interface_var, event_when_called, ring_in_use, timeout_restart, set_queue_var, set_queue_entry_var, event_member_status, short_abandoned_threshold, result_code_timer, result_code_timer_status, type, relax_timer, relax_timer_enabled, result_code_timer_enabled, suspend_transfer_time, music_class_on_hold, wait_timeout, periodic_announce_initial_delay, periodic_announce_max_play_count, client_announce_sound_file, client_announce_min_estimation_time, action_announce_sound_file, action_announce_initial_delay, action_announce_frequency, action_announce_max_play_count, action_announce_wait_time, action_announce_allowed_dtmf, position_announce_initial_delay, min_announced_hold_time, max_announced_hold_time, hold_time_announce_calculation_mode, queue_more_than, report_position, action_announce_wrong_dtmf_handling, migration FROM dbo.wbp_queue WHERE queue_name = ?"

	row := m.DB.QueryRow(query, queueName)

	// Tüm 80+ alanı doğru sırayla tarama (Bu kısım kritiktir ve kopyala yapıştır ile gelmelidir)
	err := row.Scan(
		&queue.ID, &queue.QueueName, &queue.QueueDescription, &queue.Enabled, &queue.Deleted,
		&queue.CreateDate, &queue.CreateUser, &queue.UpdateDate, &queue.UpdateUser,
		&queue.MediaArchivePeriod, &queue.MediaDeletePeriod, &queue.TenantID, &queue.TargetServiceLevel, &queue.TargetServiceLevelThreshold,
		&queue.MusicClass, &queue.Announce, &queue.Context, &queue.Timeout, &queue.MonitorFormat, &queue.Strategy, &queue.ServiceLevel, &queue.Retry, &queue.Maxlen,
		&queue.MonitorType, &queue.ReportHoldTime, &queue.MemberDelay, &queue.MemberMacro, &queue.Autofill, &queue.Weight, &queue.LeaveWhenEmpty, &queue.JoinEmpty,
		&queue.AnnounceFrequency, &queue.MinAnnounceFrequency, &queue.PeriodicAnnounceFrequency, &queue.RelativePeriodAnnounce, &queue.AnnounceHoldTime, &queue.AnnouncePosition,
		&queue.AnnounceRoundSeconds, &queue.QueueYouAreNext, &queue.QueueThereAre, &queue.QueueCallsWaiting, &queue.QueueHoldTime, &queue.QueueMinutes, &queue.QueueSeconds,
		&queue.QueueThankYou, &queue.QueueLessThan, &queue.QueueReportHold, &queue.PeriodicAnnounce, &queue.SetInterfaceVar, &queue.EventWhenCalled, &queue.RingInUse,
		&queue.TimeoutRestart, &queue.SetQueueVar, &queue.SetQueueEntryVar, &queue.EventMemberStatus, &queue.ShortAbandonedThreshold, &queue.ResultCodeTimer,
		&queue.ResultCodeTimerStatus, &queue.Type, &queue.RelaxTimer, &queue.RelaxTimerEnabled, &queue.ResultCodeTimerEnabled, &queue.SuspendTransferTime,
		&queue.MusicClassOnHold, &queue.WaitTimeout, &queue.PeriodicAnnounceInitialDelay, &queue.PeriodicAnnounceMaxPlayCount, &queue.ClientAnnounceSoundFile,
		&queue.ClientAnnounceMinEstimationTime, &queue.ActionAnnounceSoundFile, &queue.ActionAnnounceInitialDelay, &queue.ActionAnnounceFrequency, &queue.ActionAnnounceMaxPlayCount,
		&queue.ActionAnnounceWaitTime, &queue.ActionAnnounceAllowedDtmf, &queue.PositionAnnounceInitialDelay, &queue.MinAnnouncedHoldTime, &queue.MaxAnnouncedHoldTime,
		&queue.HoldTimeAnnounceCalculationMode, &queue.QueueMoreThan, &queue.ReportPosition, &queue.ActionAnnounceWrongDtmfHandling, &queue.Migration,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("'%s' adında kuyruk bulunamadı", queueName)
	}
	if err != nil {
		return nil, fmt.Errorf("kuyruk verisi okunurken hata: %w", err)
	}

	// --- 3. Önbelleğe Yazma (Yazma Kilidi) ---
	m.mu.Lock()
	m.Cache[queueName] = queue
	m.mu.Unlock()

	return queue, nil
}
