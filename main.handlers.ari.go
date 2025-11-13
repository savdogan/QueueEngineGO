package main

import (
	"log"

	"github.com/CyCoreSystems/ari/v6"
)

// handleStasisStartMessage, Java kodunun Go dilindeki karşılığıdır.

func handleStasisStartMessage(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle) {

	CustomLog(LevelInfo, "[%s] StasisStart (application: %s, channel: %v, args: %v)", ARI_MESSAGE_LOG_PREFIX, message.Application, message.Channel, message.Args)

	if message.Channel.ID == "" {
		log.Printf("ERROR: StasisStart message has no channel: %v", message)
		return // Kanal yoksa işlemi durdur
	}

	// Args kontrolü. Go'da boş string dizisi kontrolü: len(message.Args) == 0
	if len(message.Args) == 0 {
		log.Printf("ERROR: StasisStart message has no arguments: %v", message)
		return // Argüman yoksa işlemi durdur
	}

	// Client Uygulamasına Giriş
	if isInboundApplication(message.Application) {
		OnClientChannelEnter(message, cl, h)
	} else if isOutboundApplication(message.Application) {
		OnOutboundChannelEnter(message, cl, h)
	} else {
		log.Printf("ERROR: Got StasisStart for unknown ARI application: %s", message.Application)
	}
}

// Not: Bu kodun çalışması için, ari.Channel gibi tipleri
// kullanılan Go ARI kütüphanesine göre doğru bir şekilde ayarlamanız gerekir.

func OnClientChannelEnter(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle) {
	CustomLog(LevelInfo, "Inbound kanalından giriş yapıldı..")
	call := CreateCall(message, message.Channel.ID, message.Channel.Name, message.Args[0], 1)

	if call == nil {
		CustomLog(LevelInfo, "Call is empty : %s", message.Channel.ID)
		return
	}

	CustomLog(LevelInfo, "[CALL_CREATED] %+v", call)
	logCallInfo(call)
}

func OnOutboundChannelEnter(message *ari.StasisStart, cl ari.Client, h *ari.ChannelHandle) {
	CustomLog(LevelInfo, "Outbound kanalından giriş yapıldı..")
}
