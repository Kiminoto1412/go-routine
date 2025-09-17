// package main

// import (
// 	"fmt"
// 	"time"

// 	"github.com/paruj/go-routine/taskwatcher"
// )

// func main() {
// 	w := taskwatcher.NewWatcher(2) // max concurrent = 2
// 	w.Start()

// 	// Add 5 tasks
// 	for i := 1; i <= 5; i++ {
// 		task := taskwatcher.Task{
// 			ID:       fmt.Sprintf("task-%d", i),
// 			Timeout:  5 * time.Second,
// 			MaxRetry: 2,
// 		}
// 		w.AddTask(task)
// 	}

// 	// Simulate external API callbacks
// 	go func() {
// 		time.Sleep(3 * time.Second)
// 		w.MarkTaskDone("task-1")

// 		time.Sleep(2 * time.Second)
// 		w.MarkTaskDone("task-2")

// 		time.Sleep(4 * time.Second)
// 		w.MarkTaskDone("task-3")
// 	}()

// 	time.Sleep(60 * time.Second)
// 	w.Stop()
// }


package main

import (
	"fmt"
	"time"

	"github.com/paruj/go-routine/taskwatcher"
)

func main() {
	w := taskwatcher.NewWatcher(2) // max concurrent = 2
	w.Start()

	// เพิ่ม goroutine พิมพ์ heap ทุก 1 วินาที
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Println("=== Current Heap State ===")
			w.PrintHeap()
			fmt.Println("=========================")
		}
	}()

	// Add 5 tasks
	for i := 1; i <= 5; i++ {
		task := taskwatcher.Task{
			ID:       fmt.Sprintf("task-%d", i),
			Timeout:  time.Duration(3+i) * time.Second,
			MaxRetry: 2,
		}
		w.AddTask(task)
	}

	// Simulate external API callbacks
	go func() {
		time.Sleep(3 * time.Second)
		fmt.Println("✅ Marking task-1 done")
		w.MarkTaskDone("task-1")

		time.Sleep(2 * time.Second)
		fmt.Println("✅ Marking task-2 done")
		w.MarkTaskDone("task-2")

		time.Sleep(4 * time.Second)
		fmt.Println("✅ Marking task-3 done")
		w.MarkTaskDone("task-3")
	}()

	time.Sleep(40 * time.Second)
	w.Stop()
}
