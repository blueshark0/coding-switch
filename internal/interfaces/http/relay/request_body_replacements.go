package relay

import (
	"encoding/json"
	"fmt"
	"strings"

	routingdomain "codeswitch/internal/routing/domain"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func applyRequestBodyReplacements(
	bodyBytes []byte,
	replacements []routingdomain.RequestBodyReplacement,
) ([]byte, error) {
	current := bodyBytes
	for _, replacement := range replacements {
		path := strings.TrimSpace(replacement.Path)
		if path == "" {
			continue
		}
		if !gjson.GetBytes(current, path).Exists() {
			continue
		}
		rawValue := []byte(strings.TrimSpace(replacement.Value))
		if !json.Valid(rawValue) {
			return bodyBytes, fmt.Errorf("请求体替换值不是合法 JSON: path=%s", path)
		}
		modified, err := sjson.SetRawBytes(current, path, rawValue)
		if err != nil {
			return bodyBytes, fmt.Errorf("替换请求体字段失败: path=%s: %w", path, err)
		}
		current = modified
	}
	return current, nil
}
