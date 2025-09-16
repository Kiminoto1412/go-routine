package taskwatcher

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Task struct {
	ID        string
	Created   time.Time
	Timeout   time.Duration
	Retries   int
	MaxRetry  int
	Done      bool
	OnProcess func(taskID string) bool // callback เพื่อตรวจว่าเสร็จหรือยัง
}

type Watcher struct {
	ctx           context.Context
	cancel        context.CancelFunc
	maxConcurrent int
	tasksCh       chan Task
	activeTasks   map[string]Task
	mu            sync.Mutex
}

func NewWatcher(maxConcurrent int) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
		ctx:           ctx,
		cancel:        cancel,
		maxConcurrent: maxConcurrent,
		tasksCh:       make(chan Task, 100),
		activeTasks:   make(map[string]Task),
	}
}

// AddTask: เพิ่มงานเข้าคิว
func (w *Watcher) AddTask(t Task) {
	select {
	case w.tasksCh <- t:
		fmt.Println("📥 Add task:", t.ID)
	default:
		fmt.Println("⚠️ Task queue full, drop:", t.ID)
	}
}

// Start: มีแค่ 2 goroutine
func (w *Watcher) Start() {
	// Goroutine #1: consumer
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case t := <-w.tasksCh:
				w.mu.Lock()
				if len(w.activeTasks) < w.maxConcurrent {
					t.Created = time.Now()
					w.activeTasks[t.ID] = t
					fmt.Printf("🚀 Start task: %s (retry=%d)\n", t.ID, t.Retries)
				} else {
					// ถ้าเต็ม ใส่คืน queue
					go func(tt Task) { w.tasksCh <- tt }(t)
				}
				w.mu.Unlock()
			}
		}
	}()

	// Goroutine #2: runner
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.mu.Lock()
				for id, t := range w.activeTasks {
					// ✅ เช็คว่าเสร็จหรือยัง (callback)
					if t.OnProcess != nil && t.OnProcess(t.ID) {
						fmt.Println("✅ Task done:", id)
						delete(w.activeTasks, id)
						continue
					}

					// ⏳ Timeout เช็ค
					if time.Since(t.Created) > t.Timeout {
						delete(w.activeTasks, id)
						t.Retries++
						if t.MaxRetry == 0 || t.Retries <= t.MaxRetry {
							fmt.Printf("🔄 Task timeout, retry %s (retry=%d)\n", id, t.Retries)
							go func(tt Task) { w.tasksCh <- tt }(t)
						} else {
							fmt.Printf("❌ Task %s failed after max retries\n", id)
						}
					}
				}
				w.mu.Unlock()
			}
		}
	}()
}

func (w *Watcher) Stop() {
	w.cancel()
}
