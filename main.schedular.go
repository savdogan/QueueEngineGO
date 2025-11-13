package main

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

// NewScheduler: Yeni bir Scheduler başlatır.
func NewScheduler() *Scheduler {
	s := &Scheduler{
		pq:              make(TaskHeap, 0),
		cancellationMap: make(map[string][]*ScheduledTask),
		wakeUp:          make(chan struct{}, 1),
	}
	heap.Init(&s.pq)
	go s.run() // Arka planda işleri kontrol eden goroutine'i başlat
	return s
}

// --- 1. İş Tanımı ve Priority Queue Yapısı ---

// TaskHeap: time.Time'a göre sıralama yapan bir Min-Heap.
// Heap arayüzünü (Len, Less, Swap, Push, Pop) uygular.
type TaskHeap []*ScheduledTask

func (h TaskHeap) Len() int { return len(h) }

// Less: Min-Heap olduğu için, daha erken zamanda çalışacak olan daha küçüktür (yüksek önceliklidir).
func (h TaskHeap) Less(i, j int) bool { return h[i].ExecuteAt.Before(h[j].ExecuteAt) }
func (h TaskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *TaskHeap) Push(x any) {
	n := len(*h)
	task := x.(*ScheduledTask)
	task.index = n
	*h = append(*h, task)
}

func (h *TaskHeap) Pop() any {
	old := *h
	n := len(old)
	task := old[n-1]
	old[n-1] = nil // Bellek sızıntısını önler
	task.index = -1
	*h = old[0 : n-1]
	return task
}

// --- 2. Scheduler Yapısı ---

// Scheduler: İşleri yöneten ana yapı.
type Scheduler struct {
	mu sync.Mutex // Eş zamanlı erişim kontrolü
	pq TaskHeap   // Priority Queue (Heap)

	// CallID'ye ait task'ları tutan harita. İptal için HIZLI erişim sağlar.
	cancellationMap map[string][]*ScheduledTask

	// İş döngüsü kontrol kanalı. Yeni iş eklenince döngüyü yeniden başlatmak için kullanılır.
	wakeUp chan struct{}
}

// run: Arka planda çalışan ve işleri zamanında yürüten ana döngü.
func (s *Scheduler) run() {
	var timer *time.Timer
	for {
		s.mu.Lock()

		// En yakın işi kontrol et
		if s.pq.Len() == 0 {
			// İş yok. Sonsuza kadar yeni bir iş gelmesini bekleriz.
			timer = nil
			s.mu.Unlock()

			// İş gelene kadar wakeUp kanalında kilitlen.
			<-s.wakeUp
			continue
		}

		// En yakın işin zamanını al
		nextTask := s.pq[0]
		delay := time.Until(nextTask.ExecuteAt)
		s.mu.Unlock() // Lock'u bekleme süresince serbest bırak

		if delay <= 0 {
			// İşin zamanı dolmuş, hemen çalıştır.
			s.executeNextTask()
		} else {
			// Timer'ı kur veya sıfırla
			if timer == nil {
				timer = time.NewTimer(delay)
			} else {
				timer.Reset(delay)
			}

			// Timer'ın bitmesini veya yeni bir iş/iptal sinyalini bekle.
			select {
			case <-timer.C:
				s.executeNextTask() // Zamanı dolan işi çalıştır
			case <-s.wakeUp:
				// Yeni iş eklendi veya iptal oldu.
				// Döngü başa dönecek ve yeni en yakın işi hesaplayacak.
				if timer != nil {
					timer.Stop() // Eski zamanlayıcıyı durdur
				}
			}
		}
	}
}

// ScheduleTask: Yeni bir işi zamanlar.
func (s *Scheduler) ScheduleTask(callID string, delay time.Duration, action func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	executeAt := time.Now().Add(delay)
	task := &ScheduledTask{
		CallID:    callID,
		ExecuteAt: executeAt,
		Action:    action,
	}

	heap.Push(&s.pq, task)
	s.cancellationMap[callID] = append(s.cancellationMap[callID], task)

	// Heap'in başına yeni bir iş eklenmiş olabilir (yani en yakın zaman değişmiştir).
	// run goroutine'ini uyandırıp bekleme süresini güncellemesini isteriz.
	select {
	case s.wakeUp <- struct{}{}:
	default: // Eğer kanal doluysa (zaten uyandırılmayı bekliyorsa) bir şey yapma
	}
}

// CancelByCallID: Belirli bir CallID'ye ait tüm işleri iptal eder.
func (s *Scheduler) CancelByCallID(callID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks, found := s.cancellationMap[callID]
	if !found {
		fmt.Printf("[İPTAL] CallID %s zaten iptal edilmiş veya mevcut değil.\n", callID)
		return
	}

	// Bu CallID'ye ait her işi Heap'ten kaldır.
	for _, task := range tasks {
		// task.index'i kullanarak heap'ten verimli bir şekilde sileriz.
		heap.Remove(&s.pq, task.index)
	}

	delete(s.cancellationMap, callID)
	fmt.Printf("[İPTAL] CallID %s için %d adet iş başarıyla iptal edildi.\n", callID, len(tasks))

	// Yine, en yakın iş iptal edilmiş olabileceği için döngüyü uyandır.
	select {
	case s.wakeUp <- struct{}{}:
	default:
	}
}

// executeNextTask: Sıradaki işi Heap'ten çıkarır ve çalıştırır.
func (s *Scheduler) executeNextTask() {
	s.mu.Lock()

	task := heap.Pop(&s.pq).(*ScheduledTask)

	// Map'ten de bu işi temizlememiz gerekir. (Burada basitlik için tüm listeyi değil, sadece task'ın kendisini temizlemeliyiz,
	// ancak bu örnekte map'teki task listesini basitleştirmek için tam iptal sırasında temizliği yaptık)

	s.mu.Unlock()

	//fmt.Printf("[%s] *** İŞ BAŞLATILDI *** CallID: %s, Zaman: %v\n", time.Now().Format("15:04:05.000"), task.CallID, task.ExecuteAt.Format("15:04:05"))

	// İşi yeni bir goroutine içinde çalıştır (Blocking olmaması için)
	go func() {
		task.Action()
		//fmt.Printf("[%s] İş Bitti. CallID: %s\n", time.Now().Format("15:04:05.000"), task.CallID)
	}()

	// İşi haritadan temizleme adımı:
	// İşin CallID'sine bağlı birden çok task olabilir. Çalışan task'ı listeden çıkar.
	s.mu.Lock()
	if tasks, ok := s.cancellationMap[task.CallID]; ok {
		// Çalışan task'ı listeden çıkar
		for i, t := range tasks {
			if t == task {
				s.cancellationMap[task.CallID] = append(tasks[:i], tasks[i+1:]...)
				break
			}
		}
		// Eğer bu CallID'ye ait başka iş kalmadıysa, haritadan sil
		if len(s.cancellationMap[task.CallID]) == 0 {
			delete(s.cancellationMap, task.CallID)
		}
	}
	s.mu.Unlock()
}
