import { Call } from '@wailsio/runtime'

export type AppSettings = {
  show_heatmap: boolean
  show_home_title: boolean
  default_claude_provider: string     // Claude 默认供应商名称
  default_codex_provider: string      // Codex 默认供应商名称
}

const DEFAULT_SETTINGS: AppSettings = {
  show_heatmap: true,
  show_home_title: true,
  default_claude_provider: '',
  default_codex_provider: '',
}

export const fetchAppSettings = async (): Promise<AppSettings> => {
  const data = await Call.ByName('codeswitch/services.AppSettingsService.GetAppSettings')
  return data ?? DEFAULT_SETTINGS
}

export const saveAppSettings = async (settings: AppSettings): Promise<AppSettings> => {
  return Call.ByName('codeswitch/services.AppSettingsService.SaveAppSettings', settings)
}
