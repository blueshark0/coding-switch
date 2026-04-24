package domain

import (
	"strings"
	"testing"

	"codeswitch/internal/shared/kernel"
)

func TestRouteProfileValidateRejectsDuplicateProviderNames(t *testing.T) {
	profile := RouteProfile{
		Platform:          kernel.PlatformClaude,
		DefaultProviderID: intPtr(1),
		Providers: []Provider{
			{ID: 1, Name: "Alpha", Position: 1, APIURL: "https://alpha.example", APIKey: "secret"},
			{ID: 2, Name: " alpha ", Position: 2, APIURL: "https://beta.example", APIKey: "secret"},
		},
	}

	err := profile.Normalize().Validate()
	if err == nil {
		t.Fatal("expected duplicate provider name validation error")
	}
	if !strings.Contains(err.Error(), "duplicate provider name") {
		t.Fatalf("expected duplicate provider name error, got %v", err)
	}
}

func intPtr(value int) *int {
	return &value
}
