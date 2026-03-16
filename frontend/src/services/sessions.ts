import { Call } from '@wailsio/runtime'

export type SessionBinding = {
  platform: string
  session_id: string
  provider_name: string
  last_success_at: string
  created_at: string
}

const serviceName = 'codeswitch/internal/interfaces/wails.SessionFacade'

export const fetchPlatformSessions = async (platform: string): Promise<SessionBinding[]> => {
  return Call.ByName(`${serviceName}.ListByPlatform`, platform)
}

export const unbindSession = async (platform: string, sessionId: string): Promise<void> => {
  await Call.ByName(`${serviceName}.Unbind`, platform, sessionId)
}
