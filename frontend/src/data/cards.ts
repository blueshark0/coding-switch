export type AutomationCard = {
  id: number
  name: string
  apiUrl: string
  apiKey: string
  officialSite: string
  icon: string
  tint: string
  accent: string
  supportedModels?: Record<string, boolean>
  modelMapping?: Record<string, string>
  proxyMode?: string
  proxyUrl?: string
}

export const automationCardGroups: Record<'claude' | 'codex' | 'gemini', AutomationCard[]> = {
  claude: [
    {
      id: 100,
      name: '0011',
      apiUrl: 'https://0011.ai',
      apiKey: '',
      officialSite: 'https://0011.ai',
      icon: 'aicoding',
      tint: 'rgba(10, 132, 255, 0.14)',
      accent: '#0aff5cff',
    },
    {
      id: 101,
      name: 'AICoding.sh',
      apiUrl: 'https://api.aicoding.sh',
      apiKey: '',
      officialSite: 'https://aicoding.sh',
      icon: 'aicoding',
      tint: 'rgba(10, 132, 255, 0.14)',
      accent: '#0a84ff',
    },
    {
      id: 102,
      name: 'Kimi',
      apiUrl: 'https://api.moonshot.cn/anthropic',
      apiKey: '',
      officialSite: 'https://kimi.moonshot.cn',
      icon: 'kimi',
      tint: 'rgba(16, 185, 129, 0.16)',
      accent: '#10b981',
    },
    {
      id: 103,
      name: 'Deepseek',
      apiUrl: 'https://api.deepseek.com/anthropic',
      apiKey: '',
      officialSite: 'https://www.deepseek.com',
      icon: 'deepseek',
      tint: 'rgba(251, 146, 60, 0.18)',
      accent: '#f97316',
    },
  ],
  codex: [
    {
      id: 201,
      name: 'AICoding.sh',
      apiUrl: 'https://api.aicoding.sh',
      apiKey: '',
      officialSite: 'https://www.aicoding.sh',
      icon: 'aicoding',
      tint: 'rgba(236, 72, 153, 0.16)',
      accent: '#ec4899',
    },
  ],
  gemini: [
    {
      id: 301,
      name: 'Google AI Studio',
      apiUrl: 'https://generativelanguage.googleapis.com',
      apiKey: '',
      officialSite: 'https://ai.google.dev',
      icon: 'gemini',
      tint: 'rgba(99, 102, 241, 0.16)',
      accent: '#6366f1',
      supportedModels: {
        'gemini-1.5-pro': true,
        'gemini-1.5-flash': true,
        'gemini-2.0-flash': true,
        'gemini-2.0-pro': true,
      },
    },
  ],
}

export function createAutomationCards(data: AutomationCard[] = []): AutomationCard[] {
  return data.map((item) => ({
    ...item,
    modelMapping: { ...(item.modelMapping ?? {}) },
    officialSite: item.officialSite ?? '',
    supportedModels: { ...(item.supportedModels ?? {}) },
    proxyMode: item.proxyMode ?? '',
    proxyUrl: item.proxyUrl ?? '',
  }))
}
