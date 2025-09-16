package main

import (
	"fmt"
	"time"

	"github.com/paruj/go-routine/taskwatcher"
)

func main() {
	w := taskwatcher.NewWatcher(2) // max 2 at once
	w.Start()

	// Add 5 tasks
	for i := 1; i <= 5; i++ {
		task := taskwatcher.Task{
			ID:       fmt.Sprintf("task-%d", i),
			Timeout:  10 * time.Second,
			MaxRetry: 2,
		}
		w.AddTask(task)
	}

	// Simulate external API callbacks
	go func() {
		time.Sleep(3 * time.Second)
		w.MarkTaskDone("task-1")

		time.Sleep(2 * time.Second)
		w.MarkTaskDone("task-2")

		time.Sleep(4 * time.Second)
		w.MarkTaskDone("task-3")
	}()

	time.Sleep(30 * time.Second)
	w.Stop()
}
