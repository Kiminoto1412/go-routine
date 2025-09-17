package taskwatcher

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"
)

// Task struct
type Task struct {
	ID       string
	Created  time.Time
	Timeout  time.Duration
	Retries  int
	MaxRetry int
	Done     bool
	expireAt time.Time // เวลา timeout ของ task
}

// TaskHeap: priority queue (min-heap) -------------------
type TaskHeap struct {
	tasks []Task
}

func (h TaskHeap) Len() int { return len(h.tasks) }

func (h TaskHeap) Less(i, j int) bool {
	return h.tasks[i].expireAt.Before(h.tasks[j].expireAt)
}

func (h TaskHeap) Swap(i, j int) { h.tasks[i], h.tasks[j] = h.tasks[j], h.tasks[i] }

func (h *TaskHeap) Push(x interface{}) {
	h.tasks = append(h.tasks, x.(Task))
}

func (h *TaskHeap) Pop() interface{} {
	old := h.tasks
	n := len(old)
	x := old[n-1]
	h.tasks = old[0 : n-1]
	return x
}

func (h TaskHeap) Peek() (Task, bool) {
	if len(h.tasks) == 0 {
		return Task{}, false
	}
	return h.tasks[0], true
}

// Watcher --------------------------------------------
type Watcher struct {
	ctx           context.Context
	cancel        context.CancelFunc
	maxConcurrent int
	tasksCh       chan Task
	notifyCh      chan struct{} // ปลุก runner เวลามีงานใหม่

	activeTasks  map[string]Task
	waitingQueue []Task
	taskHeap     *TaskHeap

	mu sync.Mutex
}

func NewWatcher(maxConcurrent int) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	h := &TaskHeap{tasks: []Task{}}
	heap.Init(h)

	return &Watcher{
		ctx:           ctx,
		cancel:        cancel,
		maxConcurrent: maxConcurrent,
		tasksCh:       make(chan Task, 100),
		notifyCh:      make(chan struct{}, 1),
		activeTasks:   make(map[string]Task),
		waitingQueue:  make([]Task, 0),
		taskHeap:      h,
	}
}

// AddTask ------------------------------------------------
func (w *Watcher) AddTask(t Task) {
	select {
	case w.tasksCh <- t:
		fmt.Println("📥 Add task:", t.ID)
	default:
		fmt.Println("⚠️ Task queue full, drop:", t.ID)
	}
}

// Start --------------------------------------------------
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
					t.expireAt = t.Created.Add(t.Timeout)
					w.activeTasks[t.ID] = t
					heap.Push(w.taskHeap, t)
					fmt.Printf("🚀 Start task: %s (retry=%d, expireAt=%v)\n",
						t.ID, t.Retries, t.expireAt.Format("15:04:05"))
					w.signalRunner()
				} else {
					w.waitingQueue = append(w.waitingQueue, t)
					fmt.Printf("⏳ Queue task: %s (waiting)\n", t.ID)
				}
				w.mu.Unlock()
			}
		}
	}()

	// Goroutine #2: runner (ใช้ heap)
	go func() {
		for {
			w.mu.Lock()
			if w.taskHeap.Len() == 0 {
				w.mu.Unlock()
				select {
				case <-w.ctx.Done():
					return
				case <-w.notifyCh: // รอจนมีงานใหม่
					continue
				}
			}
			next, _ := w.taskHeap.Peek()
			wait := time.Until(next.expireAt)
			w.mu.Unlock()

			select {
			case <-w.ctx.Done():
				return
			case <-time.After(wait):
				// หมดเวลา → handle
				w.mu.Lock()
				expired := heap.Pop(w.taskHeap).(Task)
				if _, exists := w.activeTasks[expired.ID]; exists && !expired.Done {
					delete(w.activeTasks, expired.ID)
					expired.Retries++
					if expired.MaxRetry == 0 || expired.Retries <= expired.MaxRetry {
						fmt.Printf("🔄 Retry task: %s (retry=%d)\n", expired.ID, expired.Retries)
						expired.Created = time.Now()
						expired.expireAt = expired.Created.Add(expired.Timeout)
						w.waitingQueue = append(w.waitingQueue, expired)
					} else {
						fmt.Printf("❌ Task %s failed after max retries\n", expired.ID)
					}
				}
				w.promoteWaitingTasks()
				w.mu.Unlock()
			case <-w.notifyCh:
				// แค่ wake-up → loop จะ recalibrate timeout ใหม่
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
		next.expireAt = next.Created.Add(next.Timeout)
		w.activeTasks[next.ID] = next
		heap.Push(w.taskHeap, next)
		fmt.Printf("🚀 Start task from queue: %s (retry=%d, expireAt=%v)\n",
			next.ID, next.Retries, next.expireAt.Format("15:04:05"))
		w.signalRunner()
	}
}

func (w *Watcher) MarkTaskDone(taskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if t, exists := w.activeTasks[taskID]; exists {
		t.Done = true
		w.activeTasks[taskID] = t // Update the task in the map
		delete(w.activeTasks, taskID)
		fmt.Println("✅ Task marked done by external API:", taskID)
	} else {
		fmt.Println("⚠️ Task not found or already done:", taskID)
	}
	w.promoteWaitingTasks()
}

func (w *Watcher) signalRunner() {
	select {
	case w.notifyCh <- struct{}{}:
	default:
	}
}

func (w *Watcher) PrintHeap() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.taskHeap.tasks) == 0 {
		fmt.Println("Heap is empty")
		return
	}

	for i, t := range w.taskHeap.tasks {
		fmt.Printf("[%d] ID=%s expireAt=%s done=%v retries=%d\n",
			i, t.ID, t.expireAt.Format("15:04:05"), t.Done, t.Retries)
	}
}

