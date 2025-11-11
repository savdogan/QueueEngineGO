package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== Go Gecikmeli İş Scheduler Başlatıldı ===")
	scheduler := NewScheduler()

	// 1. İş: 4 saniye sonra çalışacak (Call 1)
	scheduler.ScheduleTask("Call-123", 4*time.Second, func() {
		fmt.Println("\n--- Call-123: 4 saniye sonra planlanan iş çalıştı. ---")
	})

	// 1. İş: 4 saniye sonra çalışacak (Call 1)
	scheduler.ScheduleTask("Call-123", 1*time.Second, func() {
		fmt.Println("\n--- Call-123: 1 saniye sonra planlanan iş çalıştı. ---")
	})

	// 2. İş: 10 saniye sonra çalışacak (Call 456)
	scheduler.ScheduleTask("Call-456", 10*time.Second, func() {
		fmt.Println("\n--- Call-456: 10 saniye sonra planlanan iş çalıştı. ---")
	})

	// 3. İş: 7 saniye sonra çalışacak (Call 123'e ait 2. iş)
	scheduler.ScheduleTask("Call-123", 7*time.Second, func() {
		fmt.Println("\n--- Call-123: 7 saniye sonra planlanan iş çalıştı. ---")
	})

	fmt.Println("\n3 saniye bekliyoruz ve Call-123'ü iptal ediyoruz (Bu, 4s ve 7s işlerini siler).")
	time.Sleep(8 * time.Second)

	// Call-123'e ait tüm işleri iptal et
	scheduler.CancelByCallID("Call-123")

	fmt.Println("10 saniyelik işin çalışmasını bekliyoruz.")

	// Programın hemen bitmemesi için bekleyin (11 saniye, 10 saniyelik işin çalışması için)
	time.Sleep(9 * time.Second)

	fmt.Println("\n=== Scheduler Kapatılıyor. ===")
}
