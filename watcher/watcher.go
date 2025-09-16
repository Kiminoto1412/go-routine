package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ProcessFlag int

const (
	FlagProcessing ProcessFlag = iota
	FlagDone
)

// Watcher เก็บ context และ channel สำหรับส่งผลลัพธ์
type Watcher struct {
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	doneCh      chan struct{}
	processFlag ProcessFlag
	mu          sync.RWMutex
}

// NewWatcher สร้าง watcher พร้อม channel สำหรับส่งผลลัพธ์
func NewWatcher() *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
		ctx:         ctx,
		cancel:      cancel,
		doneCh:      make(chan struct{}, 1),
		processFlag: FlagProcessing,
	}
}

// Go รัน goroutine watcher
func (w *Watcher) Go(name string, fn func(ctx context.Context)) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		fn(w.ctx)
		fmt.Println("🛑 Goroutine", name, "exited")
	}()
}

// Stop ส่ง cancel และรอ goroutine ออกให้หมด
func (w *Watcher) Stop() {
	w.cancel()
	w.wg.Wait()
	fmt.Println("✅ All watchers stopped")
}

// SetProcessFlag ตั้งค่า process flag
func (w *Watcher) SetProcessFlag(flag ProcessFlag) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.processFlag = flag
	fmt.Printf("📊 Process flag set to: %v\n", flag)
}

// GetProcessFlag อ่านค่า process flag
func (w *Watcher) GetProcessFlag() ProcessFlag {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.processFlag
}

// SignalDone ส่งสัญญาณ done
func (w *Watcher) SignalDone() {
	select {
	case w.doneCh <- struct{}{}:
		fmt.Println("📢 Done signal sent")
	default:
		// ถ้ามีสัญญาณรออยู่แล้ว ไม่ต้องส่งซ้ำ
	}
}

// StartTaskScheduler สำหรับ Pattern 1: Task Scheduler
func (w *Watcher) StartTaskScheduler(countdownMinutes int, taskFn func()) {
	fmt.Printf("⏰ Starting task scheduler with %d minutes countdown\n", countdownMinutes)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(time.Duration(countdownMinutes) * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				fmt.Println("⏹ Task scheduler stopped by context")
				return
			case <-ticker.C:
				fmt.Printf("⏰ Countdown finished after %d minutes\n", countdownMinutes)

				if w.GetProcessFlag() == FlagDone {
					fmt.Println("✅ Process flag is Done → spinning down")
					w.Stop()
					return
				} else {
					fmt.Println("❌ Process flag is not Done → doing nothing")
					// Do nothing, continue waiting
				}
			}
		}
	}()

	// Start the task directly
	if taskFn != nil {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			taskFn()
		}()
	}
}

// StartKafkaProducer สำหรับ Pattern 2: Kafka Producer with Retry
func (w *Watcher) StartKafkaProducer(countdownMinutes int, message []byte, producerFn func(message []byte)) {
	fmt.Printf("📨 Starting Kafka producer with %d minutes retry interval\n", countdownMinutes)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(time.Duration(countdownMinutes) * time.Minute)
		defer ticker.Stop()

		// Start first attempt immediately
		if producerFn != nil && message != nil {
			fmt.Println("📨 Producing Kafka message (first attempt)")
			w.wg.Add(1)
			go func() {
				defer w.wg.Done()
				producerFn(message)
			}()
		}

		for {
			select {
			case <-w.ctx.Done():
				fmt.Println("⏹ Kafka producer stopped by context")
				return
			case <-w.doneCh:
				fmt.Println("✅ Received done signal → ending")
				return
			case <-ticker.C:
				fmt.Printf("⏰ Retry interval finished after %d minutes\n", countdownMinutes)
				if producerFn != nil && message != nil {
					fmt.Println("📨 Producing Kafka message (retry)")
					w.wg.Add(1)
					go func() {
						defer w.wg.Done()
						producerFn(message)
					}()
				}
			}
		}
	}()
}
