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
		close(w.shutdown)
		close(w.inputCh)
	})
	w.wg.Wait()
}

// Enqueue 将项目加入处理队列
// 返回 true 表示成功入队，false 表示缓冲已满（已同步处理）
func (w *BackgroundWorker[T]) Enqueue(item T) bool {
	select {
	case <-w.shutdown:
		// 已关闭，同步处理
		_ = w.processor.ProcessSingle(item)
		return false
	default:
	}
	select {
	case w.inputCh <- item:
		return true
	default:
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
		case item, ok := <-w.inputCh:
			if !ok {
				flush()
				return
			}
			batch = append(batch, item)
			if len(batch) >= w.config.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-w.shutdown:
			flush()
			return
		}
	}
}
