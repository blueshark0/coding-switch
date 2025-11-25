package services

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	geminiSettingsDir      = ".gemini"
	geminiSettingsFileName = ".env"
	geminiBackupFileName   = "cc-studio.back.env"
	geminiAPIKeyValue      = "code-switch"
)

type GeminiSettingsService struct {
	relayAddr string
}

func NewGeminiSettingsService(relayAddr string) *GeminiSettingsService {
	return &GeminiSettingsService{relayAddr: relayAddr}
}

func (gss *GeminiSettingsService) ProxyStatus() (ClaudeProxyStatus, error) {
	status := ClaudeProxyStatus{Enabled: false, BaseURL: gss.baseURL()}
	settingsPath, _, err := gss.paths()
	if err != nil {
		return status, err
	}
	envMap, err := gss.parseEnvFile(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status, nil
		}
		return status, err
	}
	baseURL := gss.baseURL()
	enabled := strings.EqualFold(envMap["GEMINI_API_KEY"], geminiAPIKeyValue) &&
		strings.EqualFold(envMap["GOOGLE_GEMINI_BASE_URL"], baseURL)
	status.Enabled = enabled
	return status, nil
}

func (gss *GeminiSettingsService) EnableProxy() error {
	settingsPath, backupPath, err := gss.paths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}

	// 读取并备份现有配置
	var envMap map[string]string
	if _, err := os.Stat(settingsPath); err == nil {
		content, readErr := os.ReadFile(settingsPath)
		if readErr != nil {
			return readErr
		}
		// 备份原文件
		if err := os.WriteFile(backupPath, content, 0o600); err != nil {
			return err
		}
		// 解析现有配置
		envMap, err = gss.parseEnvFile(settingsPath)
		if err != nil {
			// 如果解析失败，创建新配置
			envMap = make(map[string]string)
		}
	} else {
		// 文件不存在，创建新配置
		envMap = make(map[string]string)
	}

	// 只更新代理相关的配置，保留其他配置
	envMap["GEMINI_API_KEY"] = geminiAPIKeyValue
	envMap["GOOGLE_GEMINI_BASE_URL"] = gss.baseURL()

	return gss.writeEnvFile(settingsPath, envMap)
}

func (gss *GeminiSettingsService) DisableProxy() error {
	settingsPath, backupPath, err := gss.paths()
	if err != nil {
		return err
	}
	if err := os.Remove(settingsPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Rename(backupPath, settingsPath); err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return nil
}

func (gss *GeminiSettingsService) paths() (settingsPath string, backupPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, geminiSettingsDir)
	return filepath.Join(dir, geminiSettingsFileName), filepath.Join(dir, geminiBackupFileName), nil
}

func (gss *GeminiSettingsService) baseURL() string {
	return normalizeRelayBaseURL(gss.relayAddr) + "/gemini"
}

// parseEnvFile 解析 .env 格式文件 (key=value)
func (gss *GeminiSettingsService) parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 解析 key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// 移除值两端的引号（如果有）
			value = strings.Trim(value, `"'`)
			envMap[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return envMap, nil
}

// writeEnvFile 写入 .env 格式文件
func (gss *GeminiSettingsService) writeEnvFile(path string, envMap map[string]string) error {
	var lines []string
	for key, value := range envMap {
		// 如果值包含空格或特殊字符，用引号包裹
		if strings.ContainsAny(value, " \t\n") {
			value = `"` + value + `"`
		}
		lines = append(lines, key+"="+value)
	}
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o600)
}
