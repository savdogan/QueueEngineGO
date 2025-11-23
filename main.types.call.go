package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/CyCoreSystems/ari/v6"
)

type Call struct {
	sync.RWMutex `json:"-"` // Call yapısının eşzamanlı erişimi için kilit
	// --- Veri Alanları (Sayısal, String ve Listeler) ---
	InstanceID              string       `json:"instanceId,omitempty"`
	ConnectionName          string       `json:"connectionId,omitempty"`
	OutBoundApplicationName string       `json:"outboundApplicationName,omitempty"`
	UniqueId                string       `json:"uniqueId"`
	ParentId                string       `json:"parentId,omitempty"`
	ExternalId              string       `json:"externalId,omitempty"`
	ServerId                int64        `json:"serverId"`
	QueueName               string       `json:"queueName"`
	Priority                int          `json:"priority"`
	PreferredAgent          string       `json:"preferredAgent,omitempty"`
	LastDtmf                string       `json:"lastDtmf,omitempty"`
	Skills                  []int64      `json:"skills,omitempty"`
	QeueuLog                *WbpQueueLog `json:"-"`

	// --- Types -> Constant Tipleri ---
	State               CALL_STATE              `json:"state"`
	QueueResult         QUEUE_RESULT            `json:"queueResult,omitempty"`
	CurrentApplication  string                  `json:"currentApplication,omitempty"`
	SpellOutLanguage    LANGUAGE                `json:"spellOutLanguage"`
	TerminationReason   CALL_TERMINATION_REASON `json:"terminationReason,omitempty"`
	ApplicationScenario APPLICATION_SCENARIO    `json:"applicationScenario,omitempty"`

	// --- Durum ve Sayaçlar (Null İhmal Edildi) ---
	EstimationTime            int    `json:"estimationTime,omitempty"`
	Position                  int    `json:"position,omitempty"`
	FinalPositionObtained     bool   `json:"finalPositionObtained,omitempty"`
	BridgedChannel            string `json:"bridgedChannel,omitempty"`
	Bridge                    string `json:"bridge,omitempty"`
	ReturnOnHangup            bool   `json:"returnOnHangup"`
	PositionTimestamp         int64  `json:"positionTimestamp"`
	ActionAnnounceProhibition bool   `json:"actionAnnounceProhibition"`
	DialContext               string `json:"dialContext,omitempty"`
	ActionContext             string `json:"actionContext,omitempty"`
	QueueTimeout              int    `json:"queueTimeout,omitempty"`
	ConferenceRoomId          string `json:"conferenceRoomId,omitempty"`
	IvrIteration              int    `json:"ivrIteration,omitempty"`

	ConnectedAgentId        int64  `json:"connectedAgentId,omitempty"`
	ConnectedUserName       string `json:"ConnectedUserName,omitempty"`
	ConnectedAgentChannelID string `json:"connectedAgentChannelID,omitempty"`
	IsCallInDistribution    bool   `json:"isCallInDistribution,omitempty"`

	// --- Eşzamanlılık Alanları (JSON'dan hariç tutulur) ---
	lock                      sync.Mutex `json:"-"`
	DistributionAttemptNumber int64      `json:"distributionAttemptNumber,omitempty"`
	AttempStartTime           time.Time  `json:"attempStartTime"`
	AttempEndTime             time.Time  `json:"attempEndTime"`
	FailedPingAttempts        int        `json:"-"`

	ChannelKey               *ari.Key `json:"channelKey,omitempty"`
	ChannelId                string   `json:"channelId,omitempty"`
	ChannelState             string   `json:"channelState,omitempty"`
	ChannelCallerNumber      string   `json:"channelCallerNumber,omitempty"`
	ChannelCreationtime      string   `json:"channelCreationTime,omitempty"`
	ChannelDialplanExten     string   `json:"channelDialplanExten,omitempty"`
	ChannelDialplanContext   string   `json:"channelDialplanContext,omitempty"`
	ChannelDialplanPriority  int64    `json:"channelDialplanPriority,omitempty"`
	ChannelDialplanAppName   string   `json:"channelDialplanAppName,omitempty"`
	ChannelDialplanQueueName string   `json:"channelDialplanQueueName,omitempty"`
	// ChannelDialplanParentId iki kez tekrar ettiği için birini kullandık.
	ChannelDialplanParentId         string                  `json:"channelDialplanParentId,omitempty"`
	ChannelDialplanSpellOutLanguage string                  `json:"channelDialplanSpellOutLanguage,omitempty"`
	Application                     string                  `json:"application,omitempty"`
	ChannelName                     string                  `json:"channelName,omitempty"`
	CurrentProcessName              PROCESS_NAME            `json:"currentProcessName,omitempty"`
	CurrentCallScheduleAction       CALL_SCHEDULED_ACTION   `json:"currentCallScheduledAction,omitempty"`
	WaitingActions                  []CALL_SCHEDULED_ACTION `json:"waitingActions,omitempty"`
	PeriodicPlayAnnounceCount       int                     `json:"periodicAnnouncePlayCount,omitempty"`
	PositionAnnouncePlayCount       int                     `json:"positionAnnouncePlayCount,omitempty"`
	ActionAnnouncePlayCount         int                     `json:"actionAnnouncePlayCount,omitempty"`
}

type CallSetup struct {
	QueueName  string `json:"queueName"`
	ParentId   string `json:"parentId"`
	ExternalId string `json:"externalId"`

	// Pointer alanlar
	Priority         *int      `json:"priority,omitempty"` // Null ise JSON'dan atlanır (omitempty)
	SpellOutLanguage *LANGUAGE `json:"spellOutLanguage,omitempty"`

	PreferredAgent            string  `json:"preferredAgent"`
	Skills                    []int64 `json:"skills"`
	ReturnOnHangup            bool    `json:"returnOnHangup"`
	PositionTimestamp         int64   `json:"positionTimestamp"`
	ActionAnnounceProhibition bool    `json:"actionAnnounceProhibition"`
	DialContext               string  `json:"dialContext"`
	ActionContext             string  `json:"actionContext"`
	QueueTimeout              int     `json:"queueTimeout"`

	// Pointer alanlar
	Scenario     *APPLICATION_SCENARIO `json:"scenario,omitempty"`
	IvrIteration *int                  `json:"ivrIteration,omitempty"`

	ConferenceRoomId string `json:"conferenceRoomId"`
}

// --- METOTLAR ---

// IsTerminated: Durum kontrolü
func (c *Call) IsTerminated() bool {
	return c.State == CALL_STATE_Terminated
}

// GetMediaUniqueId: UniqueId'yi oluşturur.
func (c *Call) GetMediaUniqueId() string {
	return fmt.Sprintf("%s-%d", c.UniqueId, c.IvrIteration)
}

// Lock: Kilitleme metodu
func (c *Call) Lock() {
	c.lock.Lock()
}

// Unlock: Kilit açma metodu
func (c *Call) Unlock() {
	c.lock.Unlock()
}

// Stringer Arayüzü (toString() eşleniği)
func (c *Call) String() string {
	return fmt.Sprintf("Call [uniqueId=%s, serverId=%d, state=%s, queueResult=%v]",
		c.UniqueId, c.ServerId, c.State, c.QueueResult)
}

// Orijinal koddaki Atomic metotları taklit eden Go metotları
// Eğer bu alanlara eşzamanlı erişim varsa, *atomic* paketini kullanmalısınız.
// Bu örnekte, basitçe değerleri döndürür.
func (c *Call) GetDistributionAttemptNumber() int64 {
	return c.DistributionAttemptNumber
}
func (c *Call) GetFailedPingAttempts() int {
	return c.FailedPingAttempts
}

func (c *Call) SetTerminationReason(terminationReason CALL_TERMINATION_REASON) {
	c.TerminationReason = terminationReason
	c.QueueResult = GetQueueResultFromTerminationResult(terminationReason)
}

func InstantiateCall(message *ari.StasisStart, uniqueId string, channel string, callSetup CallSetup, serverId int64) *Call {
	call := &Call{}

	call.State = CALL_STATE_New

	call.UniqueId = uniqueId
	call.ServerId = serverId

	call.ParentId = callSetup.ParentId
	call.ExternalId = callSetup.ExternalId

	// Priority (Null kontrolü)
	if callSetup.Priority != nil {
		call.Priority = *callSetup.Priority
	} else {
		call.Priority = 0
	}

	if callSetup.SpellOutLanguage != nil {
		call.SpellOutLanguage = *callSetup.SpellOutLanguage
	} else {
		call.SpellOutLanguage = DEFAULT_LANGUAGE
	}

	// PreferredAgent (isNotBlank kontrolü)
	if strings.TrimSpace(callSetup.PreferredAgent) != "" {
		call.PreferredAgent = callSetup.PreferredAgent
	}

	// Skills (isEmpty kontrolü ve Listeye ekleme)
	if len(callSetup.Skills) > 0 {
		// Java'daki addAll gibi, Go'da slice ataması veya kopyalaması yapılır.
		// Skills alanı bir slice olduğu için doğrudan atanır.
		call.Skills = callSetup.Skills
	} else {
		// Eğer Skills slice olarak kalacaksa, nil değil boş bir slice olarak kalır.
		call.Skills = []int64{}
	}

	call.ReturnOnHangup = callSetup.ReturnOnHangup
	call.PositionTimestamp = callSetup.PositionTimestamp
	call.ActionAnnounceProhibition = callSetup.ActionAnnounceProhibition
	call.DialContext = callSetup.DialContext
	call.ActionContext = callSetup.ActionContext
	call.QueueTimeout = callSetup.QueueTimeout
	call.ConferenceRoomId = callSetup.ConferenceRoomId

	// ApplicationScenario (Null kontrolü)
	if callSetup.Scenario != nil {
		call.ApplicationScenario = *callSetup.Scenario
	} else {
		call.ApplicationScenario = DEFAULT_APPLICATION_SCENARIO
	}

	// IvrIteration (Null kontrolü)
	if callSetup.IvrIteration != nil {
		call.IvrIteration = *callSetup.IvrIteration
	} else {
		call.IvrIteration = DEFAULT_IVR_ITERATION
	}

	// Loglama (Bu işlevin dışarıda tanımlandığı varsayılmıştır)
	//TO DO : Logu burada yap....
	if message.Channel.ID != "" {
		call.ChannelId = message.Channel.ID
		call.ChannelName = message.Channel.Name
		call.ChannelState = message.Channel.State
		call.ChannelCallerNumber = message.Channel.Caller.Number
		call.ChannelDialplanContext = message.Channel.Dialplan.Context
		call.ChannelDialplanExten = message.Channel.Dialplan.Exten
		call.ChannelDialplanPriority = message.Channel.Dialplan.Priority
	}

	call.Application = message.Application

	call.CurrentCallScheduleAction = CALL_SCHEDULED_ACTION_Empty

	call.QueueName = callSetup.QueueName

	return call

}

// CreateCall metodu, yeni bir Call nesnesi oluşturur.
func CreateCall(message *ari.StasisStart, uniqueId string, channelName string, callSetupJson string, serverId int64) *Call {

	clog(LevelInfo, "Call Setup %s", callSetupJson)

	log.Print(callSetupJson)

	// StringUtils.isBlank kontrolünün Go karşılığı
	// strings.TrimSpace ile baş ve sondaki boşluklar silinir, sonra boş olup olmadığı kontrol edilir.
	if strings.TrimSpace(uniqueId) == "" || strings.TrimSpace(channelName) == "" {
		log.Printf("ERROR: Could not create call: no uniqueId and/or channel")
		return nil // Go'da Java'daki null yerine nil döndürülür
	}

	// CallSetup nesnesini JSON'dan dönüştürme
	// callSetup := DeserializeCallSetupWithParser(callSetupJson)

	callSetup := &CallSetup{}

	err := json.Unmarshal([]byte(strings.ReplaceAll(callSetupJson, "'", "\"")), callSetup)
	if err != nil {
		clog(LevelError, "ERROR: Could not unmarshal valid JSON to CallSetup struct: %v. JSON: %s", err, callSetupJson)
		return nil
	}

	// Null kontrolü
	if callSetup.ParentId != "" {
		// instantiateCall metodu çağrılır (daha önce çevirdiğimiz metot)
		// cm.InstantiateCall, önceki çevirideki metot adıdır.
		return InstantiateCall(message, uniqueId, channelName, *callSetup, serverId)
	}

	// JSON dönüşümü veya diğer işlemler başarısız olursa nil döndürülür.
	return nil
}
