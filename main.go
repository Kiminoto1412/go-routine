package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/paruj/go-routine/watcher"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("🚀 Starting POC for both flowchart patterns...")
	fmt.Println("Choose pattern to run:")
	fmt.Println("1. Task Scheduler Pattern (Flowchart 1)")
	fmt.Println("2. Kafka Producer Pattern (Flowchart 2)")
	fmt.Println("3. Both patterns simultaneously")

	// For demo, let's run both patterns
	runTaskSchedulerPattern()
	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")
	runKafkaProducerPattern()
}

// Pattern 1: Task Scheduler (Flowchart 1)
func runTaskSchedulerPattern() {
	fmt.Println("📋 PATTERN 1: Task Scheduler")

	w := watcher.NewWatcher()

	// Background worker
	w.Go("task-worker", func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Println("🔧 Task worker: processing...")
				time.Sleep(2 * time.Second)
			}
		}
	})

	// Simulate task that might complete
	taskFn := func() {
		fmt.Println("📝 Starting main task...")
		time.Sleep(1 * time.Second)

		// Randomly decide if task completes
		if rand.Intn(3) == 0 {
			fmt.Println("✅ Task completed successfully!")
			w.SetProcessFlag(watcher.FlagDone)
		} else {
			fmt.Println("⏳ Task still processing...")
			w.SetProcessFlag(watcher.FlagProcessing)
		}
	}

	// Start task scheduler with 3-second countdown (for demo)
	w.StartTaskScheduler(1, taskFn) // 1 minute in real scenario, but using seconds for demo

	// Wait a bit to see the pattern
	time.Sleep(10 * time.Second)
	w.Stop()
}

// Pattern 2: Kafka Producer with Retry (Flowchart 2)
func runKafkaProducerPattern() {
	fmt.Println("📨 PATTERN 2: Kafka Producer with Retry")

	w := watcher.NewWatcher()

	// Background worker
	w.Go("gold-sync", func(ctx context.Context) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Println("📊 Kafka worker: monitoring... (still working)")
			}
		}
	})

	// Define the message to be sent
	message := []byte(`{"task_id": "kafka-task-123", "action": "process_data", "data": {"user_id": 456, "file": "document.pdf"}}`)

	// Simulate Kafka producer
	producerFn := func(msg []byte) {
		fmt.Printf("📤 Producing Kafka message: %s\n", string(msg))
		time.Sleep(2 * time.Second)

		// Randomly decide if message is sent successfully
		if rand.Intn(4) == 0 {
			fmt.Println("✅ Kafka message sent successfully!")
			w.SignalDone() // Signal completion
		} else {
			fmt.Println("❌ Kafka message failed, will retry...")
		}
	}

	// Start Kafka producer with 2-second retry interval (for demo)
	w.StartKafkaProducer(1, message, producerFn) // 1 minute in real scenario, but using seconds for demo

	// Wait a bit to see the pattern
	time.Sleep(15 * time.Second)
	w.Stop()
}
