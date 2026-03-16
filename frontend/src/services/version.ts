import { Call } from '@wailsio/runtime'

export const fetchCurrentVersion = async (): Promise<string> => {
  const version = await Call.ByName('codeswitch.VersionService.CurrentVersion')
  return version ?? ''
}
