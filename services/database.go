package services

import (
	"codeswitch/internal/shared/storage"
)

const (
	// CoreDBName 是除日志/会话外其它数据的默认连接名称
	CoreDBName = storage.CoreDBName
	// RequestLogDBName 存储 request_log 表的数据库连接名
	RequestLogDBName = storage.RequestLogDBName
	// SessionDBName 存储 session_provider_binding 表的数据库连接名
	SessionDBName = storage.SessionDBName
)
