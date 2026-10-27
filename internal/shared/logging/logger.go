package logging

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"codeswitch/internal/shared/storage"

	"github.com/daodao97/xgo/xlog"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	appLogQueueSize     = 512
	appLogBufferSize    = 64 * 1024
	appLogFlushInterval = time.Second
)

// Setup configures the global logger to write to stdout and a rotating file.
func Setup() io.WriteCloser {
	logDir, err := storage.AppDataDir()
	if err != nil {
		log.Printf("无法获取日志目录，使用当前目录: %v", err)
		logDir = "."
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		log.Printf("无法创建日志目录 %s: %v", logDir, err)
		return nil
	}

	logger := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     14,
		Compress:   true,
	}
	bufferedLogger := newAsyncWriter(logger, appLogQueueSize, appLogBufferSize, appLogFlushInterval)

	// xgo 的 xlog 默认是 DEBUG -> stdout，可能打印 SQL、完整请求体和凭证。
	// 应用已有自己的日志管道，第三方日志统一写入 io.Discard；级别设为 Error
	// 让 DEBUG/INFO/WARN 在 handler 处直接禁用，避免无谓的格式化开销。
	// 注意 SetLogger 内部会调用 slog.SetDefault，必须先于下面的 log.SetOutput，
	// 否则标准 log 包会被重定向而不再写入 app.log。
	xlog.SetLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))
	log.SetOutput(io.MultiWriter(os.Stdout, bufferedLogger))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("日志系统初始化成功，日志文件: %s", logger.Filename)
	return bufferedLogger
}
