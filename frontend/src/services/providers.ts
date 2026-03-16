import { fetchRouteProfile, saveRouteProfile, type ProviderRecord } from './routing'

export type { ProviderRecord } from './routing'

export const loadProviders = async (platform: string): Promise<ProviderRecord[]> => {
  const profile = await fetchRouteProfile(platform)
  return profile?.providers ?? []
}

export const saveProviders = async (platform: string, providers: ProviderRecord[]): Promise<void> => {
  const current = await fetchRouteProfile(platform)
  const normalizedProviders = providers.map((provider, index) => ({
    ...provider,
    position: index + 1,
  }))
  await saveRouteProfile({
    ...current,
    providers: normalizedProviders,
  })
}
