package worker

import (
	"log"
	"sync"
	"time"
)

// Config defines queue sizing and flush behavior for a background worker.
type Config struct {
	Name          string
	BufferSize    int
	BatchSize     int
	FlushInterval time.Duration
}

type BatchProcessor[T any] interface {
	Process(batch []T) error
	ProcessSingle(item T) error
}

type BackgroundWorker[T any] struct {
	config    Config
	inputCh   chan T
	shutdown  chan struct{}
	wg        sync.WaitGroup
	processor BatchProcessor[T]
	shutOnce  sync.Once
	stateMu   sync.RWMutex
	stopped   bool
}

func New[T any](config Config, processor BatchProcessor[T]) *BackgroundWorker[T] {
	return &BackgroundWorker[T]{
		config:    config,
		inputCh:   make(chan T, config.BufferSize),
		shutdown:  make(chan struct{}),
		processor: processor,
	}
}

func (w *BackgroundWorker[T]) Start() {
	w.wg.Add(1)
	go w.run()
}

func (w *BackgroundWorker[T]) Stop() {
	w.shutOnce.Do(func() {
		w.stateMu.Lock()
		w.stopped = true
		w.stateMu.Unlock()
		close(w.shutdown)
	})
	w.wg.Wait()
}

// Enqueue returns true when the item is queued and false when it is processed synchronously.
func (w *BackgroundWorker[T]) Enqueue(item T) bool {
	w.stateMu.RLock()
	if w.stopped {
		w.stateMu.RUnlock()
		_ = w.processor.ProcessSingle(item)
		return false
	}
	select {
	case w.inputCh <- item:
		w.stateMu.RUnlock()
		return true
	default:
		w.stateMu.RUnlock()
		log.Printf("%s 缓冲已满，回退为同步处理\n", w.config.Name)
		_ = w.processor.ProcessSingle(item)
		return false
	}
}

func (w *BackgroundWorker[T]) run() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]T, 0, w.config.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := w.processor.Process(batch); err != nil {
			log.Printf("%s 批量处理失败: %v，尝试逐条处理\n", w.config.Name, err)
			for _, item := range batch {
				_ = w.processor.ProcessSingle(item)
			}
		}
		batch = batch[:0]
	}

	for {
		select {
		case item := <-w.inputCh:
			batch = append(batch, item)
			if len(batch) >= w.config.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-w.shutdown:
			for {
				select {
				case item := <-w.inputCh:
					batch = append(batch, item)
					if len(batch) >= w.config.BatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}
