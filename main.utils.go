package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	arislog "golang.org/x/exp/slog"
)

// NewSlogLogger, artık ARI'nın beklediği eski tipi döndürür.
func NewSlogLogger(appName string) *arislog.Logger { // <-- Dönüş tipi değişti

	// Eski arislog handler'ı kullan
	handler := arislog.NewTextHandler(os.Stderr, &arislog.HandlerOptions{
		Level: arislog.LevelDebug,
	})

	// Eski arislog.New() çağrılır
	return arislog.New(handler).With("app", appName)
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		// Hata oluşursa, logla ve varsayılan bir değer döndür
		fmt.Printf("Hostname alınamadı: %v\n", err)
		return "UNKNOWN_HOST"
	}
	return name
}

func isInboundApplication(applicationName string) bool {
	return strings.HasPrefix(applicationName, INBOUND_ARI_APPLICATION_PREFIX)
}

func isOutboundApplication(applicationName string) bool {
	return strings.HasPrefix(applicationName, OUTBOUND_ARI_APPLICATION_PREFIX)
}

func wbpQueueToQueue(wbp WbpQueue) *Queue {
	q := Queue{
		ID:                          wbp.ID,
		Enabled:                     wbp.Enabled,
		Deleted:                     wbp.Deleted,
		TargetServiceLevel:          wbp.TargetServiceLevel,
		TargetServiceLevelThreshold: wbp.TargetServiceLevelThreshold,
		ShortAbandonedThreshold:     wbp.ShortAbandonedThreshold,
		Type:                        wbp.Type,
	}

	q.Lock()

	// Nullable String alanlar (Valid değilse -> "")
	q.QueueName = getSimpleString(wbp.QueueName)
	q.QueueDescription = getSimpleString(wbp.QueueDescription)
	q.MusicClass = getSimpleString(wbp.MusicClass)
	q.Announce = getSimpleString(wbp.Announce)
	q.Context = getSimpleString(wbp.Context)
	q.MonitorFormat = getSimpleString(wbp.MonitorFormat)
	q.Strategy = getSimpleString(wbp.Strategy)
	q.MonitorType = getSimpleString(wbp.MonitorType)
	q.MemberMacro = getSimpleString(wbp.MemberMacro)
	q.LeaveWhenEmpty = getSimpleString(wbp.LeaveWhenEmpty)
	q.JoinEmpty = getSimpleString(wbp.JoinEmpty)
	q.AnnounceHoldTime = getSimpleString(wbp.AnnounceHoldTime)
	q.AnnouncePosition = getSimpleString(wbp.AnnouncePosition)
	q.QueueYouAreNext = getSimpleString(wbp.QueueYouAreNext)
	q.QueueThereAre = getSimpleString(wbp.QueueThereAre)
	q.QueueCallsWaiting = getSimpleString(wbp.QueueCallsWaiting)
	q.QueueHoldTime = getSimpleString(wbp.QueueHoldTime)
	q.QueueMinutes = getSimpleString(wbp.QueueMinutes)
	q.QueueSeconds = getSimpleString(wbp.QueueSeconds)
	q.QueueThankYou = getSimpleString(wbp.QueueThankYou)
	q.QueueLessThan = getSimpleString(wbp.QueueLessThan)
	q.QueueReportHold = getSimpleString(wbp.QueueReportHold)
	q.PeriodicAnnounce = getSimpleString(wbp.PeriodicAnnounce)
	q.MusicClassOnHold = getSimpleString(wbp.MusicClassOnHold)
	q.ClientAnnounceSoundFile = getSimpleString(wbp.ClientAnnounceSoundFile)
	q.ActionAnnounceSoundFile = getSimpleString(wbp.ActionAnnounceSoundFile)
	q.ActionAnnounceAllowedDtmf = getSimpleString(wbp.ActionAnnounceAllowedDtmf)
	q.QueueMoreThan = getSimpleString(wbp.QueueMoreThan)

	// Nullable Int32 alanlar (Valid değilse -> 0)
	q.Timeout = getSimpleInt32(wbp.Timeout)
	q.MediaArchivePeriod = getSimpleInt32(wbp.MediaArchivePeriod)
	q.MediaDeletePeriod = getSimpleInt32(wbp.MediaDeletePeriod)
	q.ServiceLevel = getSimpleInt32(wbp.ServiceLevel)
	q.Retry = getSimpleInt32(wbp.Retry)
	q.Maxlen = getSimpleInt32(wbp.Maxlen)
	q.MemberDelay = getSimpleInt32(wbp.MemberDelay)
	q.AnnounceFrequency = getSimpleInt32(wbp.AnnounceFrequency)
	q.MinAnnounceFrequency = getSimpleInt32(wbp.MinAnnounceFrequency)
	q.PeriodicAnnounceFrequency = getSimpleInt32(wbp.PeriodicAnnounceFrequency)
	q.AnnounceRoundSeconds = getSimpleInt32(wbp.AnnounceRoundSeconds)
	q.ResultCodeTimer = getSimpleInt32(wbp.ResultCodeTimer)
	q.RelaxTimer = getSimpleInt32(wbp.RelaxTimer)
	q.SuspendTransferTime = getSimpleInt32(wbp.SuspendTransferTime)
	q.WaitTimeout = getSimpleInt32(wbp.WaitTimeout)
	q.PeriodicAnnounceInitialDelay = getSimpleInt32(wbp.PeriodicAnnounceInitialDelay)
	q.PeriodicAnnounceMaxPlayCount = getSimpleInt32(wbp.PeriodicAnnounceMaxPlayCount)
	q.ClientAnnounceMinEstimationTime = getSimpleInt32(wbp.ClientAnnounceMinEstimationTime)
	q.ActionAnnounceInitialDelay = getSimpleInt32(wbp.ActionAnnounceInitialDelay)
	q.ActionAnnounceFrequency = getSimpleInt32(wbp.ActionAnnounceFrequency)
	q.ActionAnnounceMaxPlayCount = getSimpleInt32(wbp.ActionAnnounceMaxPlayCount)
	q.ActionAnnounceWaitTime = getSimpleInt32(wbp.ActionAnnounceWaitTime)
	q.PositionAnnounceInitialDelay = getSimpleInt32(wbp.PositionAnnounceInitialDelay)
	q.MinAnnouncedHoldTime = getSimpleInt32(wbp.MinAnnouncedHoldTime)
	q.MaxAnnouncedHoldTime = getSimpleInt32(wbp.MaxAnnouncedHoldTime)
	q.HoldTimeAnnounceCalculationMode = getSimpleInt32(wbp.HoldTimeAnnounceCalculationMode)
	q.Migration = getSimpleInt32(wbp.Migration)

	// Nullable Int64 alanlar (Valid değilse -> 0)
	q.CreateUser = getSimpleInt64(wbp.CreateUser)
	q.UpdateUser = getSimpleInt64(wbp.UpdateUser)
	q.TenantID = getSimpleInt64(wbp.TenantID)
	q.ResultCodeTimerStatus = getSimpleInt64(wbp.ResultCodeTimerStatus)

	// Nullable Time alanlar (Valid değilse -> time.Time{})
	q.CreateDate = getSimpleTime(wbp.CreateDate)
	q.UpdateDate = getSimpleTime(wbp.UpdateDate)

	// Nullable Bool alanlar (Valid değilse -> false)
	q.ReportPosition = getSimpleBool(wbp.ReportPosition)
	q.ReportHoldTime = getSimpleBool(wbp.ReportHoldTime)
	q.Autofill = getSimpleBool(wbp.Autofill)
	q.RelativePeriodAnnounce = getSimpleBool(wbp.RelativePeriodAnnounce)
	q.SetInterfaceVar = getSimpleBool(wbp.SetInterfaceVar)
	q.EventWhenCalled = getSimpleBool(wbp.EventWhenCalled)
	q.RingInUse = getSimpleBool(wbp.RingInUse)
	q.TimeoutRestart = getSimpleBool(wbp.TimeoutRestart)
	q.SetQueueVar = getSimpleBool(wbp.SetQueueVar)
	q.SetQueueEntryVar = getSimpleBool(wbp.SetQueueEntryVar)
	q.EventMemberStatus = getSimpleBool(wbp.EventMemberStatus)
	q.RelaxTimerEnabled = getSimpleBool(wbp.RelaxTimerEnabled)
	q.ResultCodeTimerEnabled = getSimpleBool(wbp.ResultCodeTimerEnabled)
	q.ActionAnnounceWrongDtmfHandling = getSimpleBool(wbp.ActionAnnounceWrongDtmfHandling)

	q.Unlock()

	return &q
}

func getSimpleString(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

func getSimpleInt32(n sql.NullInt32) int32 {
	if n.Valid {
		return n.Int32
	}
	return 0
}

func getSimpleInt64(n sql.NullInt64) int64 {
	if n.Valid {
		return n.Int64
	}
	return 0
}

func getSimpleTime(t sql.NullTime) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{} // time.Time'ın sıfır değeri (Epoch başlangıcı)
}

func getSimpleBool(b sql.NullBool) bool {
	if b.Valid {
		return b.Bool
	}
	return false
}

func logCallInfo(call *Call, addedText string) {
	call.RLock()
	callJSON, _ := json.MarshalIndent(call, "", "  ")
	call.RUnlock()
	clog(LevelInfo, "[CALL]%s, %s", addedText, callJSON)
}

func logAnyInfo(data any, explainText string) {
	// 1. Kilit mekanizması (Aynen kalıyor)
	if locker, ok := data.(interface {
		RLock()
		RUnlock()
	}); ok {
		locker.RLock()
		defer locker.RUnlock()
	}

	// 2. Tip İsmini Bulma (YENİ KISIM)
	t := reflect.TypeOf(data)

	// Eğer data bir pointer ise (örn: *Call), asıl struct'ın ismini almak için .Elem() kullanırız.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	typeName := t.Name() // Örneğin: "Call", "Customer", "Order"

	// 3. JSON Dönüştürme
	jsonData, _ := json.MarshalIndent(data, "", "  ")

	// 4. Loglama
	// [ANY] yerine dinamik olarak [%s] ile tip ismini yazdırıyoruz.
	clog(LevelInfo, "[%s]%s, %s", typeName, explainText, string(jsonData))
}

func parseNonStandardFormat(input string) (map[string]interface{}, error) {
	// 1. Dış Parantezleri Temizleme (Opsiyonel ama önerilir: { ... })
	input = strings.TrimSpace(input)
	if len(input) > 1 && input[0] == '{' && input[len(input)-1] == '}' {
		input = input[1 : len(input)-1]
	}

	resultMap := make(map[string]interface{})

	// 2. Virgül (,) ile anahtar-değer çiftlerine ayırma
	pairs := strings.Split(input, ",")

	for _, pair := range pairs {
		// 3. İki nokta üst üste (:) ile anahtar ve değeri ayırma
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)

		if len(parts) != 2 {
			log.Printf("WARNING: Invalid key-value pair format: %s", pair)
			continue // Geçersiz çifti atla
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		// 4. Veri Türü Dönüşümünü Yapma (Şimdilik her şeyi string olarak tutalım)
		// Not: Eğer int, bool gibi değerleriniz varsa burada dönüşüm yapmalısınız.

		// Eğer tırnaksız string ise, doğrudan atama yapılır.
		resultMap[key] = valueStr
	}
	return resultMap, nil
}

func containsString(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true // Eşleşme bulundu, hemen true döndür
		}
	}
	return false // Döngü tamamlandı, eşleşme yok
}

// DeserializeCallSetupWithParser: Standart dışı veriyi Go Map'e çevirip sonra JSON'a çözümler.
func DeserializeCallSetupWithParser(callSetupString string) *CallSetup {

	// 1. Özel Parser ile standart dışı string'i Go Map'ine dönüştür
	dataMap, err := parseNonStandardFormat(callSetupString)
	if err != nil {
		clog(LevelError, "ERROR: Failed to parse non-standard format: %v", err)
		return nil
	}

	// 2. Map'i Geçerli Bir JSON Byte Dizisine Dönüştürme
	// Bu adım, Go'nun Map'i standart JSON kurallarına uygun olarak tırnaklar.
	jsonBytes, err := json.Marshal(dataMap)
	if err != nil {
		clog(LevelError, "ERROR: Failed to marshal map to JSON: %v", err)
		return nil
	}

	// 3. Oluşturulan Geçerli JSON'u CallSetup yapısına çözümleme
	callSetup := &CallSetup{}
	err = json.Unmarshal(jsonBytes, callSetup)

	if err != nil {
		clog(LevelError, "ERROR: Could not unmarshal valid JSON to CallSetup struct: %v. JSON: %s", err, string(jsonBytes))
		return nil
	}

	return callSetup
}

// GetQueueResultID, verilen string durum kodunun ID karşılığını döner.
// Eğer tanımlı olmayan bir kod gelirse varsayılan olarak 1 (UNKNOWN_ERROR) döner.
func GetQueueResultID(resultStr string) int {
	// Gelen veriyi büyük harfe çevirerek map içinde arıyoruz
	upperStr := strings.ToUpper(strings.TrimSpace(resultStr))

	if id, ok := queueResultMap[upperStr]; ok {
		return id
	}

	// Eğer listede yoksa varsayılan olarak UNKNOWN_ERROR (1) dönüyoruz
	// Loglama mekanizmanız varsa burada "Bilinmeyen Result Code: X" diye log atabilirsiniz.
	return ResultUnknownError
}

func GetQueueResultFromTerminationResult(terminationReason CALL_TERMINATION_REASON) QUEUE_RESULT {

	switch terminationReason {
	case CALL_TERMINATION_REASON_QueueWaitTimeReached:
		return QUEUE_RESULT_Timeout
	case CALL_TERMINATION_REASON_Abandon:
		return QUEUE_RESULT_Abandon
	case CALL_TERMINATION_REASON_AgentHangup:
		return QUEUE_RESULT_AgentHangup
	case CALL_TERMINATION_REASON_DeadChannel:
		return QUEUE_RESULT_UnknownError
	case CALL_TERMINATION_REASON_DialPlanHangup:
		return QUEUE_RESULT_UnknownError
	case CALL_TERMINATION_REASON_MaxAttemptReached:
		return QUEUE_RESULT_MaxRejectLimit
	case CALL_TERMINATION_REASON_ClientHangup:
		return QUEUE_RESULT_ClientHangup
	default:
		return QUEUE_RESULT_UnknownError
	}

}
