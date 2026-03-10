package services

import (
	"sync"
	"testing"
	"time"
)

type countingBatchProcessor struct {
	mu    sync.Mutex
	items []int
}

func (p *countingBatchProcessor) Process(batch []int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = append(p.items, batch...)
	return nil
}

func (p *countingBatchProcessor) ProcessSingle(item int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = append(p.items, item)
	return nil
}

func (p *countingBatchProcessor) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.items)
}

func TestBackgroundWorker_StopDrainsBufferedItems(t *testing.T) {
	processor := &countingBatchProcessor{}
	worker := NewBackgroundWorker[int](WorkerConfig{
		Name:          "test",
		BufferSize:    16,
		BatchSize:     64,
		FlushInterval: time.Hour,
	}, processor)
	worker.Start()

	const total = 8
	for i := 0; i < total; i++ {
		if !worker.Enqueue(i) {
			t.Fatalf("enqueue failed before stop at item %d", i)
		}
	}

	worker.Stop()

	if got := processor.Count(); got != total {
		t.Fatalf("processed count = %d, want %d", got, total)
	}
}

func TestBackgroundWorker_EnqueueAfterStopFallbacksToSync(t *testing.T) {
	processor := &countingBatchProcessor{}
	worker := NewBackgroundWorker[int](WorkerConfig{
		Name:          "test",
		BufferSize:    4,
		BatchSize:     2,
		FlushInterval: time.Second,
	}, processor)
	worker.Start()
	worker.Stop()

	if ok := worker.Enqueue(42); ok {
		t.Fatalf("enqueue should return false after stop")
	}
	if got := processor.Count(); got != 1 {
		t.Fatalf("processed count = %d, want 1", got)
	}
}

func TestBackgroundWorker_ConcurrentStopAndEnqueue_NoDrop(t *testing.T) {
	processor := &countingBatchProcessor{}
	worker := NewBackgroundWorker[int](WorkerConfig{
		Name:          "test",
		BufferSize:    4096,
		BatchSize:     128,
		FlushInterval: 20 * time.Millisecond,
	}, processor)
	worker.Start()

	const goroutines = 20
	const perGoroutine = 100
	const total = goroutines * perGoroutine

	start := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			<-start
			offset := base * perGoroutine
			for i := 0; i < perGoroutine; i++ {
				worker.Enqueue(offset + i)
			}
		}(g)
	}

	close(start)
	time.Sleep(1 * time.Millisecond)
	go worker.Stop()
	wg.Wait()
	worker.Stop()

	if got := processor.Count(); got != total {
		t.Fatalf("processed count = %d, want %d", got, total)
	}
}
