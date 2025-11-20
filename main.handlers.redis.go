package main

import (
	"context"
	"encoding/json"
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
	AppConfig.RLock()
	instanceIDs := AppConfig.InstanceIDs
	AppConfig.RUnlock()

	if !containsString(instanceIDs, rcdMessage.InstanceID) {
		clog(LevelDebug, "[AID_DIST] Bu instance bu sunucuya ait değil, InstanceId : %s", rcdMessage.InstanceID)
		return
	}

	call, found := globalCallManager.GetCall(rcdMessage.InteractionID)

	if !found {
		clog(LevelError, "[REDIS] Hata : handleOnAidDistributionMessage çağrı bulunamadı %s", payload)
		return
	}

	clog(LevelInfo, "[AID_DIST] Çağrı bulundu %s", payload)

	go globalCallManager.onAidDistributionMessage(call, &rcdMessage)

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

	if redisClientManager.Pubs == nil {
		clog(LevelFatal, "[REDIS SUBSCRIBE] Redis istemcisi atanmamış (rdb is nil)")
		return nil
	}

	// 2. Mesajı Redis'e yayımlama
	cmd := redisClientManager.Pubs.Publish(*redisClientManager.ctx, redisChannelName, payload)

	// Hata kontrolü
	if cmd.Err() != nil {
		clog(LevelError, "[REDIS PUBLISH ERROR] Mesaj yayınlama hatası: Kanal=%s, Hata=%+v", redisChannelName, cmd.Err())
		return cmd.Err()
	}

	clog(LevelInfo, "[REDIS PUBLISH] Başarılı. Kanal: %s, Payload: %s", redisChannelName, string(payload))
	return nil
}

func handleRedisSubsMessages(ctx context.Context, instanceIds []string) {

	var channels = []string{}

	for _, instanceId := range instanceIds {
		channels = append(channels, REDIS_DISTIRIBITION_CHANNEL_PREFIX+instanceId)
	}

	pubsub := redisClientManager.Subs.Subscribe(ctx, channels...)

	go func() {
		defer pubsub.Close() // Goroutine sonlandığında aboneliği kapatır.

		clog(LevelInfo, "[REDIS] %s kanallarına abone olundu.", channels)
		ch := pubsub.Channel()

		for msg := range ch {
			clog(LevelInfo, "[REDIS] 1 Mesaj Alındı. Kanal: %s, Payload: %s", msg.Channel, msg.Payload)

			redisMessageChannelName := msg.Channel

			//Birden faz instance oalbilir ona göre işlem yapacak sekilde kanalı normalize et
			if strings.HasPrefix(redisMessageChannelName, REDIS_DISTIRIBITION_CHANNEL_PREFIX) {
				handleOnAidDistributionMessage(msg.Payload)
			}

			// Kanal tipine göre mesaj işleme mantığı
			switch msg.Channel {

			default:
				clog(LevelError, "[REDIS] Bilinmeyen kanaldan mesaj alındı: %s", msg.Channel)
			}
		}
	}()
}
