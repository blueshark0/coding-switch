import { Call } from '@wailsio/runtime'

export type ClaudeProxyStatus = {
  enabled: boolean
  base_url: string
}

type Platform = 'claude' | 'codex' | 'gemini'

const serviceName = 'codeswitch/internal/interfaces/wails.PlatformProxyFacade'

const callByPlatform = async <T = unknown>(platform: Platform, method: string, payload?: any[]): Promise<T> => {
  const args = [platform, ...(payload ?? [])]
  return Call.ByName(`${serviceName}.${method}`, ...args)
}

export const fetchProxyStatus = async (platform: Platform): Promise<ClaudeProxyStatus> => {
  return callByPlatform<ClaudeProxyStatus>(platform, 'GetStatus')
}

export const enableProxy = async (platform: Platform): Promise<void> => {
  await callByPlatform(platform, 'Enable')
}

export const disableProxy = async (platform: Platform): Promise<void> => {
  await callByPlatform(platform, 'Disable')
}
