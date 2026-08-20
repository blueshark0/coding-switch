import { ListByPlatform, Unbind } from '../../bindings/codeswitch/internal/interfaces/wails/sessionfacade'
import type { SessionBinding } from '../../bindings/codeswitch/internal/sessions/domain/models'

export type { SessionBinding }

export const fetchPlatformSessions = async (platform: string): Promise<SessionBinding[]> => {
  return ListByPlatform(platform)
}

export const unbindSession = async (platform: string, sessionId: string): Promise<void> => {
  await Unbind(platform, sessionId)
}
