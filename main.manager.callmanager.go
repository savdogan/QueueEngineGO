package main

import "github.com/CyCoreSystems/ari/v6"

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) AddCall(uniqueId string, call *Call) *Call {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.RLock()
	defer cm.RUnlock()

	cm.calls[uniqueId] = call
	return call
}

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) AddOutBoundCall(uniqueId string, channelHandle *ari.ChannelHandle) *ari.ChannelHandle {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.RLock()
	defer cm.RUnlock()

	cm.outChannels[uniqueId] = channelHandle
	return channelHandle
}

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) GetCall(uniqueId string) (*Call, bool) {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.RLock()
	defer cm.RUnlock()

	call, ok := cm.calls[uniqueId]
	return call, ok
}

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) AddAgentCallMap(agentCallUniqeuId string, callUniqueId string) {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.RLock()
	defer cm.RUnlock()

	cm.agentCalltoCall[agentCallUniqeuId] = callUniqueId
}

// GetClientCallByAgentCall, Agent çağrı ID'si üzerinden Call nesnesini getirir.
func (cm *CallManager) GetClientCallByAgentCall(agentCallUniqueId string) (call *Call, callId string, isCallFound bool, isMappingFound bool) {
	cm.RLock()
	defer cm.RUnlock()

	// callId ve ok değişkenlerini doğrudan tanımlamaya gerek kalmadan,
	// yukarıda isimlendirdiğimiz değişkenleri kullanabiliriz veya içeride atayabiliriz.

	var ok bool
	callId, ok = cm.agentCalltoCall[agentCallUniqueId]

	if !ok {
		// Mapping bulunamadı: Her şey boş/false döner
		return nil, "", false, false
	}

	call, ok = cm.calls[callId]

	if !ok {
		// Mapping bulundu (isMappingFound=true) ama Call nesnesi yok (isCallFound=false)
		return nil, callId, false, true
	}

	// Her şey bulundu
	return call, callId, true, true
}

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) GetOutBoundCall(uniqueId string) (*ari.ChannelHandle, bool) {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.RLock()
	defer cm.RUnlock()

	outboundChannel, ok := cm.outChannels[uniqueId]
	return outboundChannel, ok
}

// RemoveCall, bir Call nesnesini UniqueId kullanarak listeden siler.
func (cm *CallManager) RemoveCall(uniqueId string) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.Lock()
	defer cm.Unlock()

	delete(cm.calls, uniqueId)
	// clog(constants.LevelInfo, "Call removed: %s", uniqueId)
}

// RemoveCall, bir Call nesnesini UniqueId kullanarak listeden siler.
func (cm *CallManager) RemoveAgentCallMap(agentCallUniqueId string) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.Lock()
	defer cm.Unlock()

	delete(cm.agentCalltoCall, agentCallUniqueId)
	// clog(constants.LevelInfo, "Call removed: %s", uniqueId)
}

// RemoveCall, bir Call nesnesini UniqueId kullanarak listeden siler.
func (cm *CallManager) RemoveOutBoundCall(uniqueId string) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.Lock()
	defer cm.Unlock()

	delete(cm.outChannels, uniqueId)
	// clog(constants.LevelInfo, "Call removed: %s", uniqueId)
}

// GetAllCalls, anlık olarak aktif tüm çağrıların bir listesini döndürür.
// Büyük çağrı sayıları için dikkatli kullanılmalıdır.
func (cm *CallManager) GetAllCalls() []*Call {
	cm.RLock()
	defer cm.RUnlock()

	list := make([]*Call, 0, len(cm.calls))
	for _, call := range cm.calls {
		list = append(list, call)
	}
	return list
}

// GetAllCalls, anlık olarak aktif tüm çağrıların bir listesini döndürür.
// Büyük çağrı sayıları için dikkatli kullanılmalıdır.
func (cm *CallManager) GetAllOutBoundCalls() []*ari.ChannelHandle {
	cm.RLock()
	defer cm.RUnlock()

	list := make([]*ari.ChannelHandle, 0, len(cm.outChannels))
	for _, outChannel := range cm.outChannels {
		list = append(list, outChannel)
	}
	return list
}

// GetCount, aktif çağrı sayısını döndürür.
func (cm *CallManager) GetCount() int {
	cm.RLock()
	defer cm.RUnlock()
	return len(cm.calls)
}
