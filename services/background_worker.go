package services

import (
	"log"
	"sync"
	"time"
)

// WorkerConfig 后台任务配置
type WorkerConfig struct {
	Name          string
	BufferSize    int
	BatchSize     int
	FlushInterval time.Duration
}

// BatchProcessor 批处理接口
type BatchProcessor[T any] interface {
	Process(batch []T) error
	ProcessSingle(item T) error
}

// BackgroundWorker 通用后台批处理工作器
type BackgroundWorker[T any] struct {
	config    WorkerConfig
	inputCh   chan T
	shutdown  chan struct{}
	wg        sync.WaitGroup
	processor BatchProcessor[T]
	shutOnce  sync.Once
	stateMu   sync.RWMutex
	stopped   bool
}

// NewBackgroundWorker 创建后台工作器
func NewBackgroundWorker[T any](config WorkerConfig, processor BatchProcessor[T]) *BackgroundWorker[T] {
	return &BackgroundWorker[T]{
		config:    config,
		inputCh:   make(chan T, config.BufferSize),
		shutdown:  make(chan struct{}),
		processor: processor,
	}
}

// Start 启动后台工作器
func (w *BackgroundWorker[T]) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop 停止后台工作器
func (w *BackgroundWorker[T]) Stop() {
	w.shutOnce.Do(func() {
		w.stateMu.Lock()
		w.stopped = true
		w.stateMu.Unlock()
		close(w.shutdown)
	})
	w.wg.Wait()
}

// Enqueue 将项目加入处理队列
// 返回 true 表示成功入队，false 表示已停止或缓冲已满（已同步处理）
func (w *BackgroundWorker[T]) Enqueue(item T) bool {
	w.stateMu.RLock()
	if w.stopped {
		w.stateMu.RUnlock()
		// 已关闭，同步处理
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

// run 后台处理循环
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
			// 停止时尽量排空缓冲，避免丢失已入队日志。
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
