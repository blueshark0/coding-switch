package infrastructure

import (
	"strings"

	"github.com/daodao97/xgo/xdb"
)

const (
	requestLogHTTPStatusAll     = "all"
	requestLogHTTPStatusSuccess = "success"

	// 请求日志默认只展示 http_code = 200 的记录，需要排查失败请求时再切到 all。
	requestLogHTTPStatusSuccessCode = 200
)

// normalizeRequestLogHTTPStatus 归一化响应码过滤值：只有显式 "all" 才展示全部，
// 其余（含空串、非法值）一律按默认的 success 处理。
func normalizeRequestLogHTTPStatus(httpStatus string) string {
	if strings.ToLower(strings.TrimSpace(httpStatus)) == requestLogHTTPStatusAll {
		return requestLogHTTPStatusAll
	}
	return requestLogHTTPStatusSuccess
}

// httpStatusWhereOptions 把响应码过滤下推为 http_code 列上的 SQL 条件（xdb 查询用）。
func httpStatusWhereOptions(httpStatus string) []xdb.Option {
	if httpStatus == requestLogHTTPStatusSuccess {
		return []xdb.Option{xdb.WhereEq("http_code", requestLogHTTPStatusSuccessCode)}
	}
	return nil
}

// httpStatusSQLClause 是 httpStatusWhereOptions 的原生 SQL 版本（统计聚合查询用）。
func httpStatusSQLClause(httpStatus string) (string, []any) {
	if httpStatus == requestLogHTTPStatusSuccess {
		return " AND http_code = ?", []any{requestLogHTTPStatusSuccessCode}
	}
	return "", nil
}
