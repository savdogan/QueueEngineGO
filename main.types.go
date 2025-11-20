package main

import (
	"database/sql"
	"sync"
	"time"

	"github.com/CyCoreSystems/ari/v6"
)

const AGENT_LEG_CALL_TYPE = "agent_leg"
const REDIS_DISTIRIBITION_CHANNEL_PREFIX = "channelAidDistribution"
const REDIS_NEW_INTERACTION_CHANNEL = "channelInteractionForNAID"
const REDIS_INTERACTION_STATE_CHANNEL = "channelQEInteractionState"

type NewCallInteraction struct {
	// Ajanın yetkili olduğu grup ID'lerinin listesi
	Groups []int `json:"groups"`

	// Etkileşimin çalıştığı sunucu örneği/instance ID'si
	InstanceID string `json:"instanceId"`

	// Etkileşimin benzersiz kimliği (UUID/GUID)
	InteractionID string `json:"interactionId"`

	// Etkileşimin öncelik seviyesi (örneğin 0-100 arası)
	InteractionPriority int `json:"interactionPriority"`

	// Etkileşimin yönlendirileceği kuyruk adı
	InteractionQueue string `json:"interactionQueue"`

	// Etkileşim tipi (örneğin 1: Çağrı, 2: Chat, 3: Email)
	InteractionType int `json:"interactionType"`

	// Mümkünse etkileşimin atanması istenen belirli bir ajan
	PreferredAgent string `json:"preferredAgent"`

	// Etkileşimin gerektirdiği yetenek ID'lerinin listesi
	RequiredSkills []int `json:"requiredSkills"`

	// Olayın zaman damgası (milisaniseler)
	Timestamp int64 `json:"timestamp"`
}

// DistributionMessage, yayınlayacağımız mesajın ana yapısıdır
type RedisCallDistributionMessage struct {
	InstanceID        string     `json:"instanceId"`
	InteractionID     string     `json:"interactionId"`
	Timestamp         int64      `json:"timestamp"`
	Users             []UserDist `json:"users"`
	SourceSystem      string     `json:"sourceSystem"`
	QueueName         string     `json:"queueName"`
	CurrentAttempt    int        `json:"currentAttempt"`
	PublisherHostName string     `json:"publisherHostName"`
}

// User struct'ı mevcut kodunuzda zaten var
type UserDist struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// DistributionMessage, yayınlayacağımız mesajın ana yapısıdır
type AriAppInfo struct {
	ConnectionName        string `json:"connectionId"`
	InboundAppName        string `json:"inboundAppName"`
	OutboundAppName       string `json:"outboundAppName"`
	IsOutboundApplication bool   `json:"-"`
	InstanceID            string `json:"instanceId"`
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

type Queue struct {
	mu                              sync.RWMutex `json:"-"` // Kuyruk yapısının eşzamanlı erişimi için kilit
	ID                              int64
	QueueName                       string
	QueueDescription                string
	Enabled                         bool
	Deleted                         bool
	CreateDate                      time.Time // sql.NullTime -> time.Time (Null ise sıfır zaman)
	CreateUser                      int64     // sql.NullInt64 -> int64 (Null ise 0)
	UpdateDate                      time.Time
	UpdateUser                      int64
	MediaArchivePeriod              int32
	MediaDeletePeriod               int32
	TenantID                        int64
	TargetServiceLevel              int32
	TargetServiceLevelThreshold     int32
	MusicClass                      string
	Announce                        string
	Context                         string
	Timeout                         int32
	MonitorFormat                   string
	Strategy                        string
	ServiceLevel                    int32
	Retry                           int32
	Maxlen                          int32
	MonitorType                     string
	ReportHoldTime                  bool // sql.NullBool -> bool (Null ise false)
	MemberDelay                     int32
	MemberMacro                     string
	Autofill                        bool
	Weight                          int32
	LeaveWhenEmpty                  string
	JoinEmpty                       string
	AnnounceFrequency               int32
	MinAnnounceFrequency            int32
	PeriodicAnnounceFrequency       int32
	RelativePeriodAnnounce          bool
	AnnounceHoldTime                string
	AnnouncePosition                string
	AnnounceRoundSeconds            int32
	QueueYouAreNext                 string
	QueueThereAre                   string
	QueueCallsWaiting               string
	QueueHoldTime                   string
	QueueMinutes                    string
	QueueSeconds                    string
	QueueThankYou                   string
	QueueLessThan                   string
	QueueReportHold                 string
	PeriodicAnnounce                string
	SetInterfaceVar                 bool
	EventWhenCalled                 bool
	RingInUse                       bool
	TimeoutRestart                  bool
	SetQueueVar                     bool
	SetQueueEntryVar                bool
	EventMemberStatus               bool
	ShortAbandonedThreshold         int64
	ResultCodeTimer                 int32
	ResultCodeTimerStatus           int64
	Type                            int32
	RelaxTimer                      int32
	RelaxTimerEnabled               bool
	ResultCodeTimerEnabled          bool
	SuspendTransferTime             int32
	MusicClassOnHold                string
	WaitTimeout                     int32
	PeriodicAnnounceInitialDelay    int32
	PeriodicAnnounceMaxPlayCount    int32
	ClientAnnounceSoundFile         string
	ClientAnnounceMinEstimationTime int32
	ActionAnnounceSoundFile         string
	ActionAnnounceInitialDelay      int32
	ActionAnnounceFrequency         int32
	ActionAnnounceMaxPlayCount      int32
	ActionAnnounceWaitTime          int32
	ActionAnnounceAllowedDtmf       string
	PositionAnnounceInitialDelay    int32
	MinAnnouncedHoldTime            int32
	MaxAnnouncedHoldTime            int32
	HoldTimeAnnounceCalculationMode int32
	QueueMoreThan                   string
	ReportPosition                  bool
	ActionAnnounceWrongDtmfHandling bool
	Migration                       int32
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

// InteractionState, sağlanan JSON yapısını temsil eder.
type InteractionState struct {
	InteractionID        string `json:"interactionId"`
	RelatedAgentUsername string `json:"relatedAgentUsername"`
	State                string `json:"state"`
	InstanceID           string `json:"instanceId"`
	QueueName            string `json:"queueName"`
}

type Config struct {
	sync.RWMutex                    // Yapılandırma okuma/yazma işlemleri için
	Environment         string      `json:"Environment"`
	RedisAddresses      []string    `json:"RedisAddresses"`
	InstanceIDs         []string    `json:"InstanceIDs"`
	RedisPassword       string      `json:"RedisPassword"`
	LoadSnapshotOnStart bool        `json:"LoadSnapshotOnStart"`
	MinLogLevel         LogLevel    `json:"MinLogLevel"`
	PublisherHostName   string      `json:"PublisherHostName"`
	LogDirectory        string      `json:"LogDirectory"`
	HttpServerEnabled   bool        `json:"HttpServerEnabled"`
	Version             int         `json:"Version"`
	HttpPort            int         `json:"HttpPort"`
	AriConnections      []AriConfig `json:"ari_connections"`
	DBConnectingString  string      `json:"DBConnectingString"`
}

type AriConfig struct {
	Id           string `json:"Id"`
	ConnectionId string `json:"-"`
	WebsocketURL string `json:"WebsocketUrl"`
	RestURL      string `json:"RestUrl"`
	Username     string `json:"Username"`
	Password     string `json:"Password"`
}

type ChannelGetter interface { // <--- Hata burada!
	ari.Event // Temel metotları miras alır
	GetChannel() *ari.ChannelData
}

const PING_INTERVAL_SEC = 10
const CONNECTOR_STOPPING_TIMEOUT_SEC = 30

const ARI_MESSAGE_LOG_PREFIX = "[ARI MSG] "

const DEFAULT_ARI_APPLICATION_PREFIX = "qbqe"

const (
	INBOUND_ARI_APPLICATION_PREFIX  = "gbqe-client"
	OUTBOUND_ARI_APPLICATION_PREFIX = "gbqe-outbound"
	CALL_MAX_TTL_SEC                = 21600
	REDIS_PING_MESSAGE              = "ping"
	COMPOSER_RESOURCE_KEY           = "description"
	DEFAULT_IVR_ITERATION           = 1
)

// Loglama Seviyeleri (main.go'dan taşındı)
type LogLevel int

const (
	LevelFatal LogLevel = iota // 0
	LevelError                 // 1
	LevelWarn                  // 2
	LevelInfo                  // 3
	LevelDebug                 // 4
	LevelTrace                 // 5
)

// --- LANGUAGE Enum ---
type LANGUAGE string

const (
	LANGUAGE_BY LANGUAGE = "BY"
	LANGUAGE_EN LANGUAGE = "EN"
	LANGUAGE_RU LANGUAGE = "RU"
	LANGUAGE_TR LANGUAGE = "TR"
	LANGUAGE_UA LANGUAGE = "UA"
)

const DEFAULT_LANGUAGE = LANGUAGE_EN

// --- APPLICATION_SCENARIO Enum ---
type APPLICATION_SCENARIO string

const (
	APPLICATION_SCENARIO_Queue      APPLICATION_SCENARIO = "queue"
	APPLICATION_SCENARIO_Conference APPLICATION_SCENARIO = "conference"
)

const DEFAULT_APPLICATION_SCENARIO = APPLICATION_SCENARIO_Queue

// --- CALL_STATE Enum ---
type CALL_STATE string

const (
	CALL_STATE_New              CALL_STATE = "NEW"
	CALL_STATE_InQueue          CALL_STATE = "IN_QUEUE"
	CALL_STATE_BridgedWithAgent CALL_STATE = "BRIDGED_WITH_AGENT"
	CALL_STATE_Terminated       CALL_STATE = "TERMINATED"
)

// --- CALL_STATE Enum ---
type PROCESS_NAME string

const (
	PROCESS_NAME_queue      PROCESS_NAME = "QUEUE"
	PROCESS_NAME_conference PROCESS_NAME = "CONFERENCE"
)

// --- CALL_SCHEDULED_ACTION Enum ---
type CALL_SCHEDULED_ACTION string

const (
	CALL_SCHEDULED_ACTION_Empty                 CALL_SCHEDULED_ACTION = "EMPTY"
	CALL_SCHEDULED_ACTION_QueueTimeout          CALL_SCHEDULED_ACTION = "QUEUE_TIMEOUT"
	CALL_SCHEDULED_ACTION_ActionAnnounce        CALL_SCHEDULED_ACTION = "ACTION_ANNOUNCE"
	CALL_SCHEDULED_ACTION_ActionAnnounceTimeout CALL_SCHEDULED_ACTION = "ACTION_ANNOUNCE_TIMEOUT"
	CALL_SCHEDULED_ACTION_ClientAnnounce        CALL_SCHEDULED_ACTION = "CLIENT_ANNOUNCE"
	CALL_SCHEDULED_ACTION_PeriodicAnnounce      CALL_SCHEDULED_ACTION = "PERIODIC_ANNOUNCE"
	CALL_SCHEDULED_ACTION_PositionAnnounce      CALL_SCHEDULED_ACTION = "POSITION_ANNOUNCE"
)

// --- QE_RESULT Enum ---
type QE_RESULT string

const (
	QE_RESULT_MediaServerDisabled    QE_RESULT = "MEDIA_SERVER_DISABLED"
	QE_RESULT_CallInstantiationError QE_RESULT = "CALL_INSTANTIATION_ERROR"
	QE_RESULT_AidError               QE_RESULT = "AID_ERROR"
	QE_RESULT_RedisError             QE_RESULT = "REDIS_ERROR"
)

// --- QUEUE_RESULT Enum (ID'li) ---
type QUEUE_RESULT int

const (
	QUEUE_RESULT_UnknownError           QUEUE_RESULT = 1
	QUEUE_RESULT_ClientHangup           QUEUE_RESULT = 2
	QUEUE_RESULT_AgentHangup            QUEUE_RESULT = 3
	QUEUE_RESULT_AidEngineReject        QUEUE_RESULT = 4
	QUEUE_RESULT_AidEngineError         QUEUE_RESULT = 5
	QUEUE_RESULT_Timeout                QUEUE_RESULT = 6
	QUEUE_RESULT_ActionAnnounceRedirect QUEUE_RESULT = 7
	QUEUE_RESULT_Abandon                QUEUE_RESULT = 8
	QUEUE_RESULT_QeRestart              QUEUE_RESULT = 9
	QUEUE_RESULT_MediaServerDisabled    QUEUE_RESULT = 10
)

// --- DIAL_STATUS Enum (ID'li) ---
type DIAL_STATUS string

const (
	DIAL_STATUS_Answer      DIAL_STATUS = "ANSWER"
	DIAL_STATUS_Busy        DIAL_STATUS = "BUSY"
	DIAL_STATUS_Cancel      DIAL_STATUS = "CANCEL"
	DIAL_STATUS_Chanunavail DIAL_STATUS = "CHANUNAVAIL"
	DIAL_STATUS_Congestion  DIAL_STATUS = "CONGESTION"
	DIAL_STATUS_Dontcall    DIAL_STATUS = "DONTCALL"
	DIAL_STATUS_Invalidargs DIAL_STATUS = "INVALIDARGS"
	DIAL_STATUS_Noanswer    DIAL_STATUS = "NOANSWER"
	DIAL_STATUS_Ringing     DIAL_STATUS = "RINGING"
	DIAL_STATUS_Torture     DIAL_STATUS = "TORTURE"
	DIAL_STATUS_Unknown     DIAL_STATUS = "UNKNOWN"
	DIAL_STATUS_Progress    DIAL_STATUS = "PROGRESS"
)

// --- AID_DISTRIBUTION_STATE Enum ---
type AID_DISTRIBUTION_STATE string

const (
	AID_DISTRIBUTION_STATE_Accepted   AID_DISTRIBUTION_STATE = "ACCEPTED"
	AID_DISTRIBUTION_STATE_Rejected   AID_DISTRIBUTION_STATE = "REJECTED"
	AID_DISTRIBUTION_STATE_Terminated AID_DISTRIBUTION_STATE = "TERMINATED"
	AID_DISTRIBUTION_STATE_Dismissed  AID_DISTRIBUTION_STATE = "DISMISSED"
)

// --- DIALPLAN_CONTEXT Enum ---
type DIALPLAN_CONTEXT string

const (
	DIALPLAN_CONTEXT_QeDialAgent DIALPLAN_CONTEXT = "qe_dial_agent"
	DIALPLAN_CONTEXT_QueueAction DIALPLAN_CONTEXT = "queue_action"
)

type DIALPLAN_VARIABLE string

const (
	DIALPLAN_VARIABLE_AgentAnnounce            DIALPLAN_VARIABLE = "agentAnnounce"
	DIALPLAN_VARIABLE_DialTimeout              DIALPLAN_VARIABLE = "dialTimeout"
	DIALPLAN_VARIABLE_DialUri                  DIALPLAN_VARIABLE = "dialUri"
	DIALPLAN_VARIABLE_QeResult                 DIALPLAN_VARIABLE = "QE_RESULT"
	DIALPLAN_VARIABLE_QeQueueName              DIALPLAN_VARIABLE = "QE_QUEUE_NAME"
	DIALPLAN_VARIABLE_QeQueueResult            DIALPLAN_VARIABLE = "QE_QUEUE_RESULT"
	DIALPLAN_VARIABLE_QeQueueMember            DIALPLAN_VARIABLE = "QE_QUEUE_MEMBER"
	DIALPLAN_VARIABLE_QeQueuePosition          DIALPLAN_VARIABLE = "QE_QUEUE_POSITION"
	DIALPLAN_VARIABLE_QeQueuePositionTimestamp DIALPLAN_VARIABLE = "QE_QUEUE_POSITION_TIMESTAMP"
	DIALPLAN_VARIABLE_QeDtmfValue              DIALPLAN_VARIABLE = "QE_DTMF_VALUE"
	DIALPLAN_VARIABLE_QePingTimestamp          DIALPLAN_VARIABLE = "QE_PING_TIMESTAMP"
	DIALPLAN_VARIABLE_CallType                 DIALPLAN_VARIABLE = "call_type"
)

// --- SIP_HEADER Struct ---
type SIPHeader struct {
	SequenceValue string
	HeaderName    string
}

// SIP_HEADERS map'i, Java enum'un davranışını taklit eder.
var SIP_HEADERS = map[string]SIPHeader{
	"DISTRIBUTION_ATTEMPT_NUMBER": {SequenceValue: "99", HeaderName: "X-QE-Distribution-Attempt"},
	"HOLD_MOH_CLASS":              {SequenceValue: "98", HeaderName: "X-QE-Hold-Moh-Class"},
	"QUEUE_NAME":                  {SequenceValue: "97", HeaderName: "X-Asterisk-PP-QueueName"},
	"QUEUE_ID":                    {SequenceValue: "96", HeaderName: "X-Asterisk-PP-QueueId"},
}

// --- CALL_TERMINATION_REASON Enum ---
type CALL_TERMINATION_REASON string

const (
	CALL_TERMINATION_REASON_ClientHangup   CALL_TERMINATION_REASON = "CLIENT_HANGUP"
	CALL_TERMINATION_REASON_DialPlanHangup CALL_TERMINATION_REASON = "DIAL_PLAN_HANGUP"
	CALL_TERMINATION_REASON_DeadChannel    CALL_TERMINATION_REASON = "DEAD_CHANNEL"
)

type CallDistributionAttempt struct {
	ID            int64       // Long -> int64
	StartTime     time.Time   // Date -> time.Time
	EndTime       time.Time   // Date -> time.Time
	DialTimeout   int         // Integer -> int (Null ihmal edildi)
	Username      string      // String
	UserID        int64       // Long -> int64
	AttemptNumber int64       // Long -> int64
	DialStatus    DIAL_STATUS // DIAL_STATUS enum -> constants.DIAL_STATUS
}
