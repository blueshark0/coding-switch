package services

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// SetupLogger 初始化日志系统
// 配置自动轮转的文件日志，同时输出到控制台和日志文件
func SetupLogger() io.WriteCloser {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("无法获取用户主目录，使用当前目录: %v", err)
		home = "."
	}

	// 确保日志目录存在
	logDir := filepath.Join(home, ".code-switch")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("无法创建日志目录 %s: %v", logDir, err)
		return nil
	}

	// 配置 Lumberjack 日志轮转
	logger := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"), // 日志文件路径
		MaxSize:    10,                               // 单个文件最大 10MB
		MaxBackups: 5,                                // 保留最多 5 个备份文件
		MaxAge:     14,                               // 保留最多 14 天
		Compress:   true,                             // 启用 gzip 压缩旧日志
	}

	// 创建 MultiWriter，同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, logger)
	log.SetOutput(multiWriter)

	// 设置日志格式：包含日期、时间、文件名和行号
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Printf("日志系统初始化成功，日志文件: %s", logger.Filename)

	return logger
}
