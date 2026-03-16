package infrastructure

import (
	"log"

	observabilitydomain "codeswitch/internal/observability/domain"

	"github.com/daodao97/xgo/xdb"
)

type RequestLogWriter struct{}

func (w *RequestLogWriter) Process(batch []*observabilitydomain.RequestLog) error {
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

func (w *RequestLogWriter) ProcessSingle(item *observabilitydomain.RequestLog) error {
	if item == nil {
		return nil
	}
	_, err := requestLogModel().Insert(recordFromRequestLog(item))
	if err != nil {
		log.Printf("写入 request_log 失败: %v\n", err)
	}
	return err
}
