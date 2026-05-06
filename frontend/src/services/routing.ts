import {
  ExportProviderConfig,
  GetAppPreferences,
  GetProfile,
  ImportProviderConfig,
  SaveAppPreferences,
  SaveProfile,
} from '../../bindings/codeswitch/internal/interfaces/wails/routingfacade'
import type {
  AppPreferences,
  ProviderConfigBundle,
  Provider,
  RouteProfile,
} from '../../bindings/codeswitch/internal/routing/domain'

export type ProviderRecord = Provider
export type { AppPreferences, ProviderConfigBundle, RouteProfile }

export const fetchRouteProfile = async (platform: string): Promise<RouteProfile> => {
  return GetProfile(platform)
}

export const saveRouteProfile = async (profile: RouteProfile): Promise<RouteProfile> => {
  return SaveProfile(profile)
}

export const exportProviderConfig = async (): Promise<ProviderConfigBundle> => {
  return ExportProviderConfig()
}

export const importProviderConfig = async (
  bundle: ProviderConfigBundle,
): Promise<ProviderConfigBundle> => {
  return ImportProviderConfig(bundle)
}

export const fetchAppPreferences = async (): Promise<AppPreferences> => {
  return GetAppPreferences()
}

export const saveAppPreferences = async (preferences: AppPreferences): Promise<AppPreferences> => {
  return SaveAppPreferences(preferences)
}
