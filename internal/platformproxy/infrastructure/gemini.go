package infrastructure

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	platformproxydomain "codeswitch/internal/platformproxy/domain"
)

const (
	geminiSettingsDir      = ".gemini"
	geminiSettingsFileName = ".env"
	geminiBackupFileName   = "cc-studio.back.env"
	geminiAPIKeyValue      = "code-switch"
)

type GeminiManager struct {
	relayAddr string
}

func NewGeminiManager(relayAddr string) *GeminiManager {
	return &GeminiManager{relayAddr: relayAddr}
}

func (m *GeminiManager) ProxyStatus() (platformproxydomain.Status, error) {
	status := platformproxydomain.Status{Enabled: false, BaseURL: m.baseURL()}
	settingsPath, _, err := m.paths()
	if err != nil {
		return status, err
	}
	envMap, err := m.parseEnvFile(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status, nil
		}
		return status, err
	}
	baseURL := m.baseURL()
	status.Enabled = strings.EqualFold(envMap["GEMINI_API_KEY"], geminiAPIKeyValue) &&
		strings.EqualFold(envMap["GOOGLE_GEMINI_BASE_URL"], baseURL)
	return status, nil
}

func (m *GeminiManager) EnableProxy() error {
	settingsPath, backupPath, err := m.paths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	var envMap map[string]string
	if _, err := os.Stat(settingsPath); err == nil {
		content, readErr := os.ReadFile(settingsPath)
		if readErr != nil {
			return readErr
		}
		if err := os.WriteFile(backupPath, content, 0o600); err != nil {
			return err
		}
		envMap, err = m.parseEnvFile(settingsPath)
		if err != nil {
			envMap = make(map[string]string)
		}
	} else {
		envMap = make(map[string]string)
	}
	envMap["GEMINI_API_KEY"] = geminiAPIKeyValue
	envMap["GOOGLE_GEMINI_BASE_URL"] = m.baseURL()
	return m.writeEnvFile(settingsPath, envMap)
}

func (m *GeminiManager) DisableProxy() error {
	settingsPath, backupPath, err := m.paths()
	if err != nil {
		return err
	}
	if err := os.Remove(settingsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(backupPath); err == nil {
		return os.Rename(backupPath, settingsPath)
	} else if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return nil
}

func (m *GeminiManager) paths() (settingsPath string, backupPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, geminiSettingsDir)
	return filepath.Join(dir, geminiSettingsFileName), filepath.Join(dir, geminiBackupFileName), nil
}

func (m *GeminiManager) baseURL() string {
	return normalizeRelayBaseURL(m.relayAddr) + "/gemini"
}

func (m *GeminiManager) parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	envMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			envMap[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return envMap, nil
}

func (m *GeminiManager) writeEnvFile(path string, envMap map[string]string) error {
	var lines []string
	for key, value := range envMap {
		if strings.ContainsAny(value, " \t\n") {
			value = `"` + value + `"`
		}
		lines = append(lines, key+"="+value)
	}
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o600)
}
