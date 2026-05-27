package kernel

import "testing"

func TestPlatformExtractSessionIDCodexPrefersHyphenatedHeader(t *testing.T) {
	headers := map[string]string{
		"session-id": "session-new",
		"session_id": "session-old",
	}

	if got := PlatformCodex.ExtractSessionID(headers, nil); got != "session-new" {
		t.Fatalf("unexpected session id: %s", got)
	}
}

func TestPlatformExtractSessionIDCodexFallsBackToLegacyHeader(t *testing.T) {
	headers := map[string]string{
		"session_id": "session-old",
	}

	if got := PlatformCodex.ExtractSessionID(headers, nil); got != "session-old" {
		t.Fatalf("unexpected session id: %s", got)
	}
}
