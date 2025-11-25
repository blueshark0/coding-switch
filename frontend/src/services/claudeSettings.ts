import { Call } from '@wailsio/runtime'
import type { ClaudeProxyStatus } from '../../bindings/codeswitch/services/models'

type Platform = 'claude' | 'codex' | 'gemini'

const serviceNames: Record<Platform, string> = {
  claude: 'codeswitch/services.ClaudeSettingsService',
  codex: 'codeswitch/services.CodexSettingsService',
  gemini: 'codeswitch/services.GeminiSettingsService',
}

const callByPlatform = async <T = unknown>(platform: Platform, method: string, payload?: any[]): Promise<T> => {
  const service = serviceNames[platform]
  const args = payload ?? []
  return Call.ByName(`${service}.${method}`, ...args)
}

export const fetchProxyStatus = async (platform: Platform): Promise<ClaudeProxyStatus> => {
  return callByPlatform<ClaudeProxyStatus>(platform, 'ProxyStatus')
}

export const enableProxy = async (platform: Platform): Promise<void> => {
  await callByPlatform(platform, 'EnableProxy')
}

export const disableProxy = async (platform: Platform): Promise<void> => {
  await callByPlatform(platform, 'DisableProxy')
}
