export type TranslateFn = (key: string, named?: Record<string, unknown>) => string
export type LocaleRef = { value: string }

export type VendorForm = {
  name: string
  apiUrl: string
  apiKey: string
  officialSite: string
  icon: string
  supportedModels?: Record<string, boolean>
  modelMapping?: Record<string, string>
  proxyMode?: string
  proxyUrl?: string
}

export type ProviderStatDisplay =
  | { state: 'loading' | 'empty'; message: string }
  | {
      state: 'ready'
      requests: string
      tokens: string
      cost: string
      successRateLabel: string
      successRateClass: string
    }
