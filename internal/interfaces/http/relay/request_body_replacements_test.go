package relay

import (
	"testing"

	routingdomain "codeswitch/internal/routing/domain"
	"codeswitch/internal/shared/kernel"
	"github.com/tidwall/gjson"
)

func TestPrepareForwardContextAppliesRequestBodyReplacementsBeforeModelMapping(t *testing.T) {
	server := &Server{}
	provider := &routingdomain.Provider{
		Name:   "alpha",
		APIURL: "https://alpha.example",
		APIKey: "secret",
		ModelMapping: map[string]string{
			"claude-opus-4": "vendor/claude-opus-4",
		},
		RequestBodyReplacements: []routingdomain.RequestBodyReplacement{
			{Path: "model", Value: `"claude-opus-4"`},
			{Path: "metadata.user_id", Value: `"fixed-user"`},
		},
	}

	forward, err := server.prepareForwardContext(&RelayContext{
		Platform:  kernel.PlatformClaude,
		BodyBytes: []byte(`{"model":"claude-sonnet-4","metadata":{"user_id":"original"}}`),
		Query:     map[string]string{},
		Headers:   map[string]string{},
		RequestMeta: RequestMeta{
			Endpoint:          "/v1/messages",
			RequestedModel:    "claude-sonnet-4",
			BodyHasModelField: true,
		},
	}, provider)
	if err != nil {
		t.Fatalf("prepareForwardContext: %v", err)
	}

	if got := gjson.GetBytes(forward.BodyBytes, "model").String(); got != "vendor/claude-opus-4" {
		t.Fatalf("expected remapped model after request body replacement, got %q", got)
	}
	if got := gjson.GetBytes(forward.BodyBytes, "metadata.user_id").String(); got != "fixed-user" {
		t.Fatalf("expected user id replacement, got %q", got)
	}
	if forward.Model != "vendor/claude-opus-4" {
		t.Fatalf("expected forward model to use post-replacement mapping, got %q", forward.Model)
	}
}

func TestApplyRequestBodyReplacementsSkipsMissingPath(t *testing.T) {
	body, err := applyRequestBodyReplacements(
		[]byte(`{"model":"claude-sonnet-4"}`),
		[]routingdomain.RequestBodyReplacement{
			{Path: "metadata.user_id", Value: `"fixed-user"`},
			{Path: "model", Value: `"claude-opus-4"`},
		},
	)
	if err != nil {
		t.Fatalf("applyRequestBodyReplacements: %v", err)
	}
	if gjson.GetBytes(body, "metadata.user_id").Exists() {
		t.Fatalf("expected missing path not to be created: %s", body)
	}
	if got := gjson.GetBytes(body, "model").String(); got != "claude-opus-4" {
		t.Fatalf("expected existing model to be replaced, got %q", got)
	}
}

func TestApplyRequestBodyReplacementsRequiresJSONLiteral(t *testing.T) {
	_, err := applyRequestBodyReplacements(
		[]byte(`{"model":"claude-sonnet-4"}`),
		[]routingdomain.RequestBodyReplacement{
			{Path: "model", Value: `not-json`},
		},
	)
	if err == nil {
		t.Fatal("expected invalid JSON literal to fail")
	}
}

func TestApplyRequestBodyReplacementsPreservesJSONTypes(t *testing.T) {
	body, err := applyRequestBodyReplacements(
		[]byte(`{
			"stringValue":"",
			"numberValue":0,
			"boolValue":false,
			"nullValue":"value",
			"objectValue":{},
			"arrayValue":[]
		}`),
		[]routingdomain.RequestBodyReplacement{
			{Path: "stringValue", Value: `"text"`},
			{Path: "numberValue", Value: `12.5`},
			{Path: "boolValue", Value: `true`},
			{Path: "nullValue", Value: `null`},
			{Path: "objectValue", Value: `{"nested":1}`},
			{Path: "arrayValue", Value: `[1,2]`},
		},
	)
	if err != nil {
		t.Fatalf("applyRequestBodyReplacements: %v", err)
	}

	if got := gjson.GetBytes(body, "stringValue").String(); got != "text" {
		t.Fatalf("expected string replacement, got %q", got)
	}
	if got := gjson.GetBytes(body, "numberValue").Float(); got != 12.5 {
		t.Fatalf("expected number replacement, got %v", got)
	}
	if got := gjson.GetBytes(body, "boolValue").Bool(); !got {
		t.Fatal("expected bool replacement")
	}
	if got := gjson.GetBytes(body, "nullValue"); got.Type != gjson.Null {
		t.Fatalf("expected null replacement, got %s", got.Raw)
	}
	if got := gjson.GetBytes(body, "objectValue.nested").Int(); got != 1 {
		t.Fatalf("expected object replacement, got %v", got)
	}
	if got := gjson.GetBytes(body, "arrayValue.1").Int(); got != 2 {
		t.Fatalf("expected array replacement, got %v", got)
	}
}

func TestPrepareForwardContextIgnoresRequestBodyReplacementsForNonClaude(t *testing.T) {
	server := &Server{}
	provider := &routingdomain.Provider{
		Name:   "codex-provider",
		APIURL: "https://codex.example",
		APIKey: "secret",
		RequestBodyReplacements: []routingdomain.RequestBodyReplacement{
			{Path: "model", Value: `"other-model"`},
		},
	}

	forward, err := server.prepareForwardContext(&RelayContext{
		Platform:  kernel.PlatformCodex,
		BodyBytes: []byte(`{"model":"gpt-5"}`),
		Query:     map[string]string{},
		Headers:   map[string]string{},
		RequestMeta: RequestMeta{
			Endpoint:          "/responses",
			RequestedModel:    "gpt-5",
			BodyHasModelField: true,
		},
	}, provider)
	if err != nil {
		t.Fatalf("prepareForwardContext: %v", err)
	}
	if got := gjson.GetBytes(forward.BodyBytes, "model").String(); got != "gpt-5" {
		t.Fatalf("expected non-Claude body to remain unchanged, got %q", got)
	}
}
