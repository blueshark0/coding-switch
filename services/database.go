package services

import "github.com/daodao97/xgo/xdb"

const (
	// CoreDBName 是除日志/会话外其它数据的默认连接名称
	CoreDBName = "default"
	// RequestLogDBName 存储 request_log 表的数据库连接名
	RequestLogDBName = "request_log"
	// SessionDBName 存储 session_provider_binding 表的数据库连接名
	SessionDBName = "session"
)

// requestLogModel 返回绑定到 request_log 数据库连接的 Model
func requestLogModel() xdb.Model {
	return xdb.New("request_log", xdb.WithConn(RequestLogDBName))
}
