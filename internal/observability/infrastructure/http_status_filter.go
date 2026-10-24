package infrastructure

import (
	"strings"

	"github.com/daodao97/xgo/xdb"
)

const (
	requestLogHTTPStatusAll     = "all"
	requestLogHTTPStatusSuccess = "success"
	requestLogHTTPStatusFailed  = "failed"

	// 请求日志默认只展示 http_code = 200 的记录，需要排查失败请求时再切到 all。
	requestLogHTTPStatusSuccessCode = 200
)

// normalizeRequestLogHTTPStatus 归一化响应码过滤值；空串和非法值沿用默认 success。
func normalizeRequestLogHTTPStatus(httpStatus string) string {
	switch strings.ToLower(strings.TrimSpace(httpStatus)) {
	case requestLogHTTPStatusAll:
		return requestLogHTTPStatusAll
	case requestLogHTTPStatusFailed:
		return requestLogHTTPStatusFailed
	default:
		return requestLogHTTPStatusSuccess
	}
}

// httpStatusWhereOptions 把响应码过滤下推为 http_code 列上的 SQL 条件（xdb 查询用）。
func httpStatusWhereOptions(httpStatus string) []xdb.Option {
	switch httpStatus {
	case requestLogHTTPStatusSuccess:
		return []xdb.Option{xdb.WhereEq("http_code", requestLogHTTPStatusSuccessCode)}
	case requestLogHTTPStatusFailed:
		return []xdb.Option{xdb.WhereGte("http_code", 400)}
	default:
		return nil
	}
}

// httpStatusSQLClause 是 httpStatusWhereOptions 的原生 SQL 版本（统计聚合查询用）。
func httpStatusSQLClause(httpStatus string) (string, []any) {
	switch httpStatus {
	case requestLogHTTPStatusSuccess:
		return " AND http_code = ?", []any{requestLogHTTPStatusSuccessCode}
	case requestLogHTTPStatusFailed:
		return " AND http_code >= ?", []any{400}
	default:
		return "", nil
	}
}
