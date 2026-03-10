package services

import (
	"testing"
)

func TestImportService_SaveProvidersPrependsImportedProviders(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	providerService := NewProviderService()
	importService := NewImportService(providerService, nil)

	existing := []Provider{
		{
			ID:      1,
			Name:    "Existing Alpha",
			APIURL:  "https://existing-alpha.example.com",
			APIKey:  "existing-alpha-key",
			Enabled: true,
		},
		{
			ID:      2,
			Name:    "Existing Beta",
			APIURL:  "https://existing-beta.example.com",
			APIKey:  "existing-beta-key",
			Enabled: true,
		},
	}
	if err := providerService.SaveProviders("claude", existing); err != nil {
		t.Fatalf("SaveProviders(existing) error = %v", err)
	}

	candidates := []providerCandidate{
		{
			Name:   "Imported Alpha",
			APIURL: "https://imported-alpha.example.com",
			APIKey: "imported-alpha-key",
		},
		{
			Name:   "Imported Beta",
			APIURL: "https://imported-beta.example.com",
			APIKey: "imported-beta-key",
		},
	}

	added, err := importService.saveProviders("claude", candidates)
	if err != nil {
		t.Fatalf("saveProviders error = %v", err)
	}
	if added != len(candidates) {
		t.Fatalf("saveProviders added = %d, want %d", added, len(candidates))
	}

	got, err := providerService.LoadProviders("claude")
	if err != nil {
		t.Fatalf("LoadProviders error = %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len(providers) = %d, want 4", len(got))
	}

	wantNames := []string{"Imported Alpha", "Imported Beta", "Existing Alpha", "Existing Beta"}
	for i, want := range wantNames {
		if got[i].Name != want {
			t.Fatalf("providers[%d].Name = %q, want %q", i, got[i].Name, want)
		}
	}

	if got[0].ID != 3 || got[1].ID != 4 {
		t.Fatalf("imported provider IDs = [%d %d], want [3 4]", got[0].ID, got[1].ID)
	}
}

func TestDiffProviderCandidatesFiltersDuplicatesAndSortsDeterministically(t *testing.T) {
	existing := []Provider{
		{
			ID:      1,
			Name:    "Existing Name",
			APIURL:  "https://existing.example.com",
			APIKey:  "existing-key",
			Enabled: true,
		},
	}

	entries := map[string]ccProviderEntry{
		"zulu": {
			Name: "Zulu",
			Settings: ccProviderSetting{
				Env: map[string]string{
					"ANTHROPIC_BASE_URL":   "https://zulu.example.com",
					"ANTHROPIC_AUTH_TOKEN": "zulu-key",
				},
			},
		},
		"alpha": {
			Name: "Alpha",
			Settings: ccProviderSetting{
				Env: map[string]string{
					"ANTHROPIC_BASE_URL":   "https://alpha.example.com",
					"ANTHROPIC_AUTH_TOKEN": "alpha-key",
				},
			},
		},
		"beta": {
			Name: "Beta",
			Settings: ccProviderSetting{
				Env: map[string]string{
					"ANTHROPIC_BASE_URL":   "https://beta.example.com",
					"ANTHROPIC_AUTH_TOKEN": "beta-key",
				},
			},
		},
		"duplicate-url": {
			Name: "Duplicate URL",
			Settings: ccProviderSetting{
				Env: map[string]string{
					"ANTHROPIC_BASE_URL":   "https://alpha.example.com",
					"ANTHROPIC_AUTH_TOKEN": "duplicate-url-key",
				},
			},
		},
		"duplicate-name": {
			Name: "Existing Name",
			Settings: ccProviderSetting{
				Env: map[string]string{
					"ANTHROPIC_BASE_URL":   "https://another.example.com",
					"ANTHROPIC_AUTH_TOKEN": "duplicate-name-key",
				},
			},
		},
		"missing-key": {
			Name: "Missing Key",
			Settings: ccProviderSetting{
				Env: map[string]string{
					"ANTHROPIC_BASE_URL": "https://missing-key.example.com",
				},
			},
		},
	}

	got := diffProviderCandidates("claude", entries, existing)
	if len(got) != 3 {
		t.Fatalf("len(candidates) = %d, want 3", len(got))
	}

	wantNames := []string{"Alpha", "Beta", "Zulu"}
	for i, want := range wantNames {
		if got[i].Name != want {
			t.Fatalf("candidates[%d].Name = %q, want %q", i, got[i].Name, want)
		}
	}
}
