package main

// NewCallManager, CallManager'ın güvenli bir örneğini oluşturur ve başlatır.
func NewCallManager() *CallManager {
	return &CallManager{
		calls: make(map[string]*Call),
	}
}

// --- İŞLEM METOTLARI ---

// AddCall, yeni bir Call nesnesini yöneticinin listesine ekler.
func (cm *CallManager) AddCall(call *Call) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.calls[call.UniqueId] = call
	// CustomLog(constants.LevelInfo, "Call added: %s", call.UniqueId)
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

// RemoveCall, bir Call nesnesini UniqueId kullanarak listeden siler.
func (cm *CallManager) RemoveCall(uniqueId string) {
	// Yazma işlemi olduğu için tam kilit (Lock) kullanıyoruz
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.calls, uniqueId)
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

// GetCount, aktif çağrı sayısını döndürür.
func (cm *CallManager) GetCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.calls)
}
