package main

import (
	"encoding/json"
	"log"
	"strings"
)

func handleOnAidDistributionMessage(payload string) {

	var rcdMessage RedisCallDistributionMessage

	if err := json.Unmarshal([]byte(payload), &rcdMessage); err != nil {
		clog(LevelError, "[REDIS] Hata : handleOnAidDistributionMessage %v %s", err, payload)
		return
	}

	if rcdMessage.InstanceID == "" || rcdMessage.InteractionID == "" || rcdMessage.QueueName == "" || len(rcdMessage.Users) == 0 || rcdMessage.Users[0].Username == "" || rcdMessage.Users[0].ID == 0 {
		clog(LevelError, "[REDIS] Hata : handleOnAidDistributionMessage eksik alanlar var %s", payload)
		return
	}
	g.Cfg.RLock()
	instanceIDs := g.Cfg.InstanceIDs
	g.Cfg.RUnlock()

	if !containsString(instanceIDs, rcdMessage.InstanceID) {
		clog(LevelDebug, "[AID_DIST] Bu instance bu sunucuya ait değil, InstanceId : %s", rcdMessage.InstanceID)
		return
	}

	call, found := g.CM.GetCall(rcdMessage.InteractionID)

	if !found {
		clog(LevelError, "[REDIS] Hata : handleOnAidDistributionMessage çağrı bulunamadı %s", payload)
		return
	}

	clog(LevelInfo, "[AID_DIST] Çağrı bulundu %s", payload)

	go g.CM.onAidDistributionMessage(call, &rcdMessage)

}

func PublishNewInteractionMessage(newInteraction NewCallInteraction) error {

	payloadBytes, err := json.Marshal(newInteraction)
	if err != nil {
		clog(LevelError, "[REDIS PUBLISH] Hata : PublishNewInterActionMessage json marshal %v", err)
		return err
	}
	redisChannelName := REDIS_NEW_INTERACTION_CHANNEL

	return PublishMessageViaRedis(redisChannelName, payloadBytes)

}

func PublishInteractionStateMessage(interactionState InteractionState) error {

	payloadBytes, err := json.Marshal(interactionState)
	if err != nil {
		clog(LevelError, "[REDIS PUBLISH] Hata : PublishInteractionStateMessage json marshal %v", err)
		return err
	}
	redisChannelName := REDIS_INTERACTION_STATE_CHANNEL

	return PublishMessageViaRedis(redisChannelName, payloadBytes)

}

func PublishMessageViaRedis(redisChannelName string, payload []byte) error {

	if g.RPubs == nil {
		clog(LevelFatal, "[REDIS SUBSCRIBE] Redis istemcisi atanmamış (rdb is nil)")
		return nil
	}

	// 2. Mesajı Redis'e yayımlama
	cmd := g.RPubs.Publish(g.Ctx, redisChannelName, payload)

	// Hata kontrolü
	if cmd.Err() != nil {
		clog(LevelError, "[REDIS PUBLISH ERROR] Mesaj yayınlama hatası: Kanal=%s, Hata=%+v", redisChannelName, cmd.Err())
		return cmd.Err()
	}

	clog(LevelInfo, "[REDIS PUBLISH] Başarılı. Kanal: %s, Payload: %s", redisChannelName, string(payload))
	return nil
}

func startRedisListener() {

	var channels = []string{}

	instanceIds := g.Cfg.InstanceIDs

	for _, instanceId := range instanceIds {
		channels = append(channels, REDIS_DISTIRIBITION_CHANNEL_PREFIX+instanceId)
	}

	channels = append(channels, REDIS_GBWEBPHONE_CHANNEL)

	pubsub := g.RSubs.Subscribe(g.Ctx, channels...)

	go func() {
		defer pubsub.Close() // Goroutine sonlandığında aboneliği kapatır.

		clog(LevelInfo, "[REDIS] %s kanallarına abone olundu.", channels)
		ch := pubsub.Channel()

		for msg := range ch {
			clog(LevelInfo, "[REDIS] 1 Mesaj Alındı. Kanal: %s, Payload: %s", msg.Channel, msg.Payload)

			redisMessageChannelName := msg.Channel

			//Birden faz instance oalbilir ona göre işlem yapacak sekilde kanalı normalize et
			if strings.HasPrefix(redisMessageChannelName, REDIS_DISTIRIBITION_CHANNEL_PREFIX) {
				go handleOnAidDistributionMessage(msg.Payload)
			} else {

				// Kanal tipine göre mesaj işleme mantığı
				switch msg.Channel {

				case REDIS_GBWEBPHONE_CHANNEL:
					go handleWebphoneMessage(msg.Payload)
				default:
					clog(LevelError, "[REDIS] Bilinmeyen kanaldan mesaj alındı: %s", msg.Channel)
				}
			}
		}
	}()
}

func handleWebphoneMessage(payload string) {
	// Go'da tip kontrolleri için switch-case daha yaygın ve okunaklıdır

	message := &WebphoneEntityMessage{}

	err := json.Unmarshal([]byte(payload), message)
	if err != nil {
		log.Printf("Could not handle WebphoneEntityMessage entity message: %v", err)
		return
	}

	switch message.EntityType {
	case EntityServer:
		go handleServerEntityMessage(message)
	case EntityQueue:
		go handleQueueEntityMessage(message)
	default:
		// Opsiyonel: Beklenmeyen bir tür gelirse burası çalışır.
		// fmt.Printf("Bilinmeyen EntityType: %s\n", message.EntityType)
	}
}

func handleServerEntityMessage(message *WebphoneEntityMessage) {
	var server WbpServer

	// 1. JSON Parse İşlemi (Gson.fromJson karşılığı)
	// message.SerializableObject []byte olduğu için direkt Unmarshal edilebilir.
	err := json.Unmarshal(message.SerializableObject, &server)
	if err != nil {
		log.Printf("Could not handle server entity message: %v", err)
		return
	}

	// 2. Action Type Kontrolü
	switch message.Type {
	case ActionAdd, ActionEdit:
		g.ServerManager.addServer(server.ID)
		// ariConnectionManager.OnCreateServer(server.ID)
		log.Printf("Server is creating/recreating: %d", server.ID)
	case ActionDelete:
		log.Printf("Server is deleting: %d", server.ID)
		g.ServerManager.deleteServer(server.ID)
	default:
		log.Printf("Bilinmeyen Server Action: %s", message.Type)
	}
}

func handleQueueEntityMessage(message *WebphoneEntityMessage) {
	var queue Queue

	// 1. JSON Parse İşlemi
	err := json.Unmarshal(message.SerializableObject, &queue)
	if err != nil {
		log.Printf("Could not handle queue entity message: %v", err)
		return
	}

	// 2. Action Type Kontrolü
	switch message.Type {
	case ActionAdd, ActionEdit:
		g.QCM.LoadQueue(queue.QueueName)
		clog(LevelInfo, "Queue qefinition updated: %s", queue.QueueName)
	case ActionDelete:
		g.QCM.RemoveQueue(queue.QueueName)
		clog(LevelInfo, "Queue qefinition deleted: %s", message.Type)
	default:
		clog(LevelInfo, "Unknown Queue Action: %s", message.Type)
	}
}
