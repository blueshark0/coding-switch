package services

import (
	"log"

	"github.com/daodao97/xgo/xdb"
)

// RequestLogWriter 请求日志批量写入处理器
type RequestLogWriter struct{}

// Process 批量处理请求日志
func (w *RequestLogWriter) Process(batch []*RequestLog) error {
	records := make([]xdb.Record, 0, len(batch))
	for _, logEntry := range batch {
		if logEntry != nil {
			records = append(records, recordFromRequestLog(logEntry))
		}
	}
	if len(records) == 0 {
		return nil
	}
	_, err := requestLogModel().InsertBatch(records)
	return err
}

// ProcessSingle 处理单条请求日志
func (w *RequestLogWriter) ProcessSingle(item *RequestLog) error {
	if item == nil {
		return nil
	}
	_, err := requestLogModel().Insert(recordFromRequestLog(item))
	if err != nil {
		log.Printf("写入 request_log 失败: %v\n", err)
	}
	return err
}
