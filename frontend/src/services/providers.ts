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

import { TestProvider } from '../../bindings/codeswitch/internal/interfaces/wails/routingfacade'
import type { ProviderProbeResult } from '../../bindings/codeswitch/internal/routing/application/models'

export type { ProviderProbeResult }

// 供应商连通性探测：以后端代理/TLS 设置对 Base URL 做一次 GET（不携带凭证）。
export const testProviderConnection = async (provider: ProviderRecord): Promise<ProviderProbeResult> => {
  return TestProvider(provider)
}
