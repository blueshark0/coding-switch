package infrastructure

import (
	"strings"

	observabilitydomain "codeswitch/internal/observability/domain"

	"github.com/daodao97/xgo/xdb"
)

// ListSessionRequestLogs 返回本地保留期内一个会话的全部结构化请求记录，
// 供会话追踪视图切换请求发生当日与全周期；不读取或返回原始请求/响应体。
func (q *SQLiteQueries) ListSessionRequestLogs(platform string, sessionID string) ([]observabilitydomain.RequestLog, error) {
	platform = strings.TrimSpace(platform)
	sessionID = strings.TrimSpace(sessionID)
	if platform == "" || sessionID == "" {
		return []observabilitydomain.RequestLog{}, nil
	}
	records, err := requestLogModel().Selects(
		xdb.WhereEq("platform", platform),
		xdb.WhereEq("session_id", sessionID),
		xdb.OrderByDesc("id"),
	)
	if err != nil {
		if isNoSuchTableErr(err) {
			return []observabilitydomain.RequestLog{}, nil
		}
		return nil, err
	}
	return q.requestLogsFromRecords(records), nil
}
