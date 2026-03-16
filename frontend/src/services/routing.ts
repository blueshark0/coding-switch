import { Call } from '@wailsio/runtime'

const serviceName = 'codeswitch/internal/interfaces/wails.RoutingFacade'

export type ProviderRecord = {
  id: number
  name: string
  apiUrl: string
  apiKey: string
  officialSite: string
  icon: string
  tint: string
  accent: string
  enabled: boolean
  position?: number
  supportedModels?: Record<string, boolean>
  modelMapping?: Record<string, string>
}

export type RouteProfile = {
  platform: string
  defaultProviderId?: number | null
  providers: ProviderRecord[]
}

export type AppPreferences = {
  show_heatmap: boolean
  show_home_title: boolean
}

export const fetchRouteProfile = async (platform: string): Promise<RouteProfile> => {
  return Call.ByName(`${serviceName}.GetProfile`, platform)
}

export const saveRouteProfile = async (profile: RouteProfile): Promise<RouteProfile> => {
  return Call.ByName(`${serviceName}.SaveProfile`, profile)
}

export const fetchAppPreferences = async (): Promise<AppPreferences> => {
  return Call.ByName(`${serviceName}.GetAppPreferences`)
}

export const saveAppPreferences = async (preferences: AppPreferences): Promise<AppPreferences> => {
  return Call.ByName(`${serviceName}.SaveAppPreferences`, preferences)
}
