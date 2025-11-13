package main

import (
	"database/sql"
	"sync"
	"time"

	"github.com/CyCoreSystems/ari/v6"
)

// DBManager, tüm uygulama için tek bir SQL Server bağlantısını yönetir.
type DBManager struct {
	DB *sql.DB
}

// QueueCacheManager, DB bağlantısını ve Queue önbelleğini yönetir.
type QueueCacheManager struct {
	DB    *sql.DB
	Cache map[string]*WbpQueue // Key: queue_name, Value: *WbpQueue
	mu    sync.RWMutex         // Cache erişimi için kilit
}

// WbpQueue, [dbo].[wbp_queue] tablosunun bir satırını temsil eder.
// json:"..." etiketleri, bu yapıyı bir API üzerinden gönderebilmeniz için eklendi.
type WbpQueue struct {
	ID                              int64          `json:"id"`
	QueueName                       sql.NullString `json:"queue_name"`
	QueueDescription                sql.NullString `json:"queue_description"`
	Enabled                         bool           `json:"enabled"` // NOT NULL
	Deleted                         bool           `json:"deleted"` // NOT NULL
	CreateDate                      sql.NullTime   `json:"create_date"`
	CreateUser                      sql.NullInt64  `json:"create_user"`
	UpdateDate                      sql.NullTime   `json:"update_date"`
	UpdateUser                      sql.NullInt64  `json:"update_user"`
	MediaArchivePeriod              sql.NullInt32  `json:"media_archive_period"`
	MediaDeletePeriod               sql.NullInt32  `json:"media_delete_period"`
	TenantID                        sql.NullInt64  `json:"tenant_id"`
	TargetServiceLevel              int32          `json:"target_service_level"`           // NOT NULL
	TargetServiceLevelThreshold     int32          `json:"target_service_level_threshold"` // NOT NULL
	MusicClass                      sql.NullString `json:"music_class"`
	Announce                        sql.NullString `json:"announce"`
	Context                         sql.NullString `json:"context"`
	Timeout                         sql.NullInt32  `json:"timeout"`
	MonitorFormat                   sql.NullString `json:"monitor_format"`
	Strategy                        sql.NullString `json:"strategy"`
	ServiceLevel                    sql.NullInt32  `json:"service_level"`
	Retry                           sql.NullInt32  `json:"retry"`
	Maxlen                          sql.NullInt32  `json:"maxlen"`
	MonitorType                     sql.NullString `json:"monitor_type"`
	ReportHoldTime                  sql.NullBool   `json:"report_hold_time"`
	MemberDelay                     sql.NullInt32  `json:"member_delay"`
	MemberMacro                     sql.NullString `json:"member_macro"`
	Autofill                        sql.NullBool   `json:"autofill"`
	Weight                          sql.NullInt32  `json:"weight"`
	LeaveWhenEmpty                  sql.NullString `json:"leave_when_empty"`
	JoinEmpty                       sql.NullString `json:"join_empty"`
	AnnounceFrequency               sql.NullInt32  `json:"announce_frequency"`
	MinAnnounceFrequency            sql.NullInt32  `json:"min_announce_frequency"`
	PeriodicAnnounceFrequency       sql.NullInt32  `json:"periodic_announce_frequency"`
	RelativePeriodAnnounce          sql.NullBool   `json:"relative_period_announce"`
	AnnounceHoldTime                sql.NullString `json:"announce_hold_time"`
	AnnouncePosition                sql.NullString `json:"announce_position"`
	AnnounceRoundSeconds            sql.NullInt32  `json:"announce_round_seconds"`
	QueueYouAreNext                 sql.NullString `json:"queue_you_are_next"`
	QueueThereAre                   sql.NullString `json:"queue_there_are"`
	QueueCallsWaiting               sql.NullString `json:"queue_calls_waiting"`
	QueueHoldTime                   sql.NullString `json:"queue_hold_time"`
	QueueMinutes                    sql.NullString `json:"queue_minutes"`
	QueueSeconds                    sql.NullString `json:"queue_seconds"`
	QueueThankYou                   sql.NullString `json:"queue_thank_you"`
	QueueLessThan                   sql.NullString `json:"queue_less_than"`
	QueueReportHold                 sql.NullString `json:"queue_report_hold"`
	PeriodicAnnounce                sql.NullString `json:"periodic_announce"`
	SetInterfaceVar                 sql.NullBool   `json:"set_interface_var"`
	EventWhenCalled                 sql.NullBool   `json:"event_when_called"`
	RingInUse                       sql.NullBool   `json:"ring_in_use"`
	TimeoutRestart                  sql.NullBool   `json:"timeout_restart"`
	SetQueueVar                     sql.NullBool   `json:"set_queue_var"`
	SetQueueEntryVar                sql.NullBool   `json:"set_queue_entry_var"`
	EventMemberStatus               sql.NullBool   `json:"event_member_status"`
	ShortAbandonedThreshold         int64          `json:"short_abandoned_threshold"` // NOT NULL
	ResultCodeTimer                 sql.NullInt32  `json:"result_code_timer"`
	ResultCodeTimerStatus           sql.NullInt64  `json:"result_code_timer_status"`
	Type                            int32          `json:"type"` // NOT NULL
	RelaxTimer                      sql.NullInt32  `json:"relax_timer"`
	RelaxTimerEnabled               sql.NullBool   `json:"relax_timer_enabled"`
	ResultCodeTimerEnabled          sql.NullBool   `json:"result_code_timer_enabled"`
	SuspendTransferTime             sql.NullInt32  `json:"suspend_transfer_time"`
	MusicClassOnHold                sql.NullString `json:"music_class_on_hold"`
	WaitTimeout                     sql.NullInt32  `json:"wait_timeout"`
	PeriodicAnnounceInitialDelay    sql.NullInt32  `json:"periodic_announce_initial_delay"`
	PeriodicAnnounceMaxPlayCount    sql.NullInt32  `json:"periodic_announce_max_play_count"`
	ClientAnnounceSoundFile         sql.NullString `json:"client_announce_sound_file"`
	ClientAnnounceMinEstimationTime sql.NullInt32  `json:"client_announce_min_estimation_time"`
	ActionAnnounceSoundFile         sql.NullString `json:"action_announce_sound_file"`
	ActionAnnounceInitialDelay      sql.NullInt32  `json:"action_announce_initial_delay"`
	ActionAnnounceFrequency         sql.NullInt32  `json:"action_announce_frequency"`
	ActionAnnounceMaxPlayCount      sql.NullInt32  `json:"action_announce_max_play_count"`
	ActionAnnounceWaitTime          sql.NullInt32  `json:"action_announce_wait_time"`
	ActionAnnounceAllowedDtmf       sql.NullString `json:"action_announce_allowed_dtmf"`
	PositionAnnounceInitialDelay    sql.NullInt32  `json:"position_announce_initial_delay"`
	MinAnnouncedHoldTime            sql.NullInt32  `json:"min_announced_hold_time"`
	MaxAnnouncedHoldTime            sql.NullInt32  `json:"max_announced_hold_time"`
	HoldTimeAnnounceCalculationMode sql.NullInt32  `json:"hold_time_announce_calculation_mode"`
	QueueMoreThan                   sql.NullString `json:"queue_more_than"`
	ReportPosition                  sql.NullBool   `json:"report_position"`
	ActionAnnounceWrongDtmfHandling sql.NullBool   `json:"action_announce_wrong_dtmf_handling"`
	Migration                       sql.NullInt32  `json:"migration"`
}

// Daha Sonra Kullanılacak...
type HttpEvent struct {
	Type          string   `json:"type"`
	AgentID       int      `json:"agent_id,omitempty"`
	CallID        string   `json:"call_id,omitempty"`
	QueueName     string   `json:"queue_name"`
	AgentUserName string   `json:"username,omitempty"`
	InstanceID    string   `json:"instance_id,omitempty"`
	QueueNames    []string `json:"queue_names,omitempty"`
}

// ScheduledTask: Zamanlanmış işi ve ilişkili verileri tutar.
type ScheduledTask struct {
	CallID    string    // Hangi çağrıya ait olduğunu gösteren ID
	ExecuteAt time.Time // İşin çalışacağı zaman (Heap için öncelik)
	Action    func()    // Çalıştırılacak fonksiyon
	index     int       // Heap içindeki pozisyonu (silme/güncelleme için kritik)
}

type LogLevel int

const (
	LevelFatal LogLevel = iota // 0: Kritik sistem hatası (Sistemden çıkar)
	LevelError                 // 1: Hata durumları
	LevelWarn                  // 2: Uyarılar
	LevelInfo                  // 3: Genel Bilgi
	LevelDebug                 // 4: Detaylı hata ayıklama
	LevelTrace                 // 5: Herşey
)

// ClientManager, tüm aktif ARI istemcilerini yönetir.
type ClientManager struct {
	clients map[string]ari.Client
	mu      sync.RWMutex
}

type Config struct {
	Environment         string        `json:"Environment"`
	RedisAddresses      []string      `json:"RedisAddresses"`
	RedisPassword       string        `json:"RedisPassword"`
	LoadSnapshotOnStart bool          `json:"LoadSnapshotOnStart"`
	MinLogLevel         LogLevel      `json:"MinLogLevel"`
	PublisherHostName   string        `json:"PublisherHostName"`
	LogDirectory        string        `json:"LogDirectory"`
	HttpServerEnabled   bool          `json:"HttpServerEnabled"`
	Version             int           `json:"Version"`
	HttpPort            int           `json:"HttpPort"`
	mu                  *sync.RWMutex // Yapılandırma okuma/yazma işlemleri için
	AriConnections      []AriConfig   `json:"ari_connections"`
}

type AriConfig struct {
	Id           string `json:"Id"`
	Application  string `json:"Application"`
	WebsocketURL string `json:"WebsocketUrl"`
	RestURL      string `json:"RestUrl"`
	Username     string `json:"Username"`
	Password     string `json:"Password"`
}

type ChannelGetter interface { // <--- Hata burada!
	ari.Event // Temel metotları miras alır
	GetChannel() *ari.ChannelData
}
