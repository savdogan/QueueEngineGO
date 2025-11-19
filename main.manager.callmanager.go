package main

import "github.com/CyCoreSystems/ari/v6"

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) AddCall(uniqueId string, call *Call) *Call {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cm.calls[uniqueId] = call
	return call
}

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) AddOutBoundCall(uniqueId string, channelHandle *ari.ChannelHandle) *ari.ChannelHandle {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cm.outChannels[uniqueId] = channelHandle
	return channelHandle
}

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) GetCall(uniqueId string) (*Call, bool) {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	call, ok := cm.calls[uniqueId]
	return call, ok
}

// GetCall, UniqueId kullanarak bir Call nesnesini döndürür.
// Eşzamanlı okuma için uygundur.
func (cm *CallManager) GetOutBoundCall(uniqueId string) (*ari.ChannelHandle, bool) {
	// Okuma işlemi olduğu için okuma kilidi (RLock) kullanıyoruz
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	outboundChannel, ok := cm.outChannels[uniqueId]
	return outboundChannel, ok
}

// RemoveCall, bir Call nesnesini UniqueId kullanarak listeden siler.
func (cm *CallManager) RemoveCall(uniqueId string) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.calls, uniqueId)
	// CustomLog(constants.LevelInfo, "Call removed: %s", uniqueId)
}

// RemoveCall, bir Call nesnesini UniqueId kullanarak listeden siler.
func (cm *CallManager) RemoveOutBoundCall(uniqueId string) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.outChannels, uniqueId)
	// CustomLog(constants.LevelInfo, "Call removed: %s", uniqueId)
}

// GetAllCalls, anlık olarak aktif tüm çağrıların bir listesini döndürür.
// Büyük çağrı sayıları için dikkatli kullanılmalıdır.
func (cm *CallManager) GetAllCalls() []*Call {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	list := make([]*Call, 0, len(cm.calls))
	for _, call := range cm.calls {
		list = append(list, call)
	}
	return list
}

// GetAllCalls, anlık olarak aktif tüm çağrıların bir listesini döndürür.
// Büyük çağrı sayıları için dikkatli kullanılmalıdır.
func (cm *CallManager) GetAllOutBoundCalls() []*ari.ChannelHandle {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	list := make([]*ari.ChannelHandle, 0, len(cm.outChannels))
	for _, outChannel := range cm.outChannels {
		list = append(list, outChannel)
	}
	return list
}

// GetCount, aktif çağrı sayısını döndürür.
func (cm *CallManager) GetCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.calls)
}
