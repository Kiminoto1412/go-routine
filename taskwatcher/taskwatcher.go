package taskwatcher

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Task struct {
	ID       string
	Created  time.Time
	Timeout  time.Duration
	Retries  int
	MaxRetry int
	Done     bool
}

type Watcher struct {
	ctx           context.Context
	cancel        context.CancelFunc
	maxConcurrent int
	tasksCh       chan Task

	activeTasks  map[string]Task
	waitingQueue []Task

	mu sync.Mutex
}

func NewWatcher(maxConcurrent int) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
		ctx:           ctx,
		cancel:        cancel,
		maxConcurrent: maxConcurrent,
		tasksCh:       make(chan Task, 100),
		activeTasks:   make(map[string]Task),
		waitingQueue:  make([]Task, 0),
	}
}

func (w *Watcher) AddTask(t Task) {
	select {
	case w.tasksCh <- t:
		fmt.Println("📥 Add task:", t.ID)
	default:
		fmt.Println("⚠️ Task queue full, drop:", t.ID)
	}
}

func (w *Watcher) Start() {
	// Goroutine #1: consumer → รับ task ใหม่
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
					w.waitingQueue = append(w.waitingQueue, t)
					fmt.Printf("⏳ Queue task: %s (waiting)\n", t.ID)
				}
				w.mu.Unlock()
			}
		}
	}()

	// Goroutine #2: runner → ตรวจ timeout + move from waiting
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.mu.Lock()

				// check timeout
				for id, t := range w.activeTasks {
					if time.Since(t.Created) > t.Timeout {
						delete(w.activeTasks, id)
						t.Retries++
						if t.MaxRetry == 0 || t.Retries <= t.MaxRetry {
							fmt.Printf("🔄 Retry task: %s (retry=%d)\n", id, t.Retries)
							t.Created = time.Now()
							w.waitingQueue = append(w.waitingQueue, t)
						} else {
							fmt.Printf("❌ Task %s failed after max retries\n", id)
						}
					}
				}
				
				w.promoteWaitingTasks()

				w.mu.Unlock()
			}
		}
	}()
}

func (w *Watcher) Stop() {
	w.cancel()
}

func (w *Watcher) promoteWaitingTasks() {
	for len(w.activeTasks) < w.maxConcurrent && len(w.waitingQueue) > 0 {
		next := w.waitingQueue[0]
		w.waitingQueue = w.waitingQueue[1:]

		next.Created = time.Now()

		w.activeTasks[next.ID] = next
		fmt.Printf("🚀 Start task from queue: %s (retry=%d)\n", next.ID, next.Retries)
	}
}

// ✅ External API/Service can call this
func (w *Watcher) MarkTaskDone(taskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.activeTasks[taskID]; exists {
		delete(w.activeTasks, taskID)
		fmt.Println("✅ Task marked done by external API:", taskID)
	} else {
		fmt.Println("⚠️ Task not found or already done:", taskID)
	}

	w.promoteWaitingTasks()
}
