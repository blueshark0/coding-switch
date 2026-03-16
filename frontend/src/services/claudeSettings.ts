import { Disable, Enable, GetStatus } from '../../bindings/codeswitch/internal/interfaces/wails/platformproxyfacade'
import type { Status as ClaudeProxyStatus } from '../../bindings/codeswitch/internal/platformproxy/domain/models'

type Platform = 'claude' | 'codex' | 'gemini'

export type { ClaudeProxyStatus }

export const fetchProxyStatus = async (platform: Platform): Promise<ClaudeProxyStatus> => {
  return GetStatus(platform)
}

export const enableProxy = async (platform: Platform): Promise<void> => {
  await Enable(platform)
}

export const disableProxy = async (platform: Platform): Promise<void> => {
  await Disable(platform)
}
