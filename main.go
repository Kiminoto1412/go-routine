package main

import (
	"fmt"
	"time"

	"github.com/paruj/go-routine/taskwatcher"
)

func main() {
	w := taskwatcher.NewWatcher(2) // max 2 tasks at the same time
	w.Start()

	// 模拟ส่ง 3 งานเข้าไป
	for i := 1; i <= 5; i++ {
		task := taskwatcher.Task{
			ID:       fmt.Sprintf("task-%d", i),
			Timeout:  5 * time.Second,
			MaxRetry: 3,
			OnProcess: func(taskID string) bool {
				// mock: task-2 จะ fail ตลอด
				if taskID == "task-2" {
					return false
				}
				// mock: task-1, task-3 จะเสร็จหลัง 6 วินาที
				return time.Now().Unix()%6 == 0
			},
		}
		w.AddTask(task)
	}

	time.Sleep(30 * time.Second)
	w.Stop()
}
