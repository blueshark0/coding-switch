package relay

import (
	"net/http"
	"testing"
)

func TestNewInsecureProxyClientDisablesTLSVerification(t *testing.T) {
	client, err := newInsecureProxyClient("http://127.0.0.1:7890")
	if err != nil {
		t.Fatalf("new insecure proxy client: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}
	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected TLS verification to be disabled for proxied requests")
	}
}

func TestRelayShouldBypassProxy(t *testing.T) {
	tests := []struct {
		name  string
		host  string
		rules []string
		want  bool
	}{
		{name: "wildcard", host: "api.example.com", rules: []string{"*"}, want: true},
		{name: "exact", host: "localhost", rules: []string{"localhost"}, want: true},
		{name: "suffix", host: "api.example.com", rules: []string{".example.com"}, want: true},
		{name: "port stripped", host: "127.0.0.1", rules: []string{"127.0.0.1:8080"}, want: true},
		{name: "miss", host: "api.example.com", rules: []string{"internal.local"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relayShouldBypassProxy(tt.host, tt.rules); got != tt.want {
				t.Fatalf("relayShouldBypassProxy() = %v, want %v", got, tt.want)
			}
		})
	}
}
