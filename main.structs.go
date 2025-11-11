package main

import "time"

// ScheduledTask: Zamanlanmış işi ve ilişkili verileri tutar.
type ScheduledTask struct {
	CallID    string    // Hangi çağrıya ait olduğunu gösteren ID
	ExecuteAt time.Time // İşin çalışacağı zaman (Heap için öncelik)
	Action    func()    // Çalıştırılacak fonksiyon
	index     int       // Heap içindeki pozisyonu (silme/güncelleme için kritik)
}
