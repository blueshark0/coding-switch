import type { AutomationCard } from '../../data/cards'

export type { TranslateFn, LocaleRef } from '../../types/i18n'
export type { RequestBodyReplacement } from '../../data/cards'

// 编辑表单是卡片去掉前端展示专属字段后的子集，随 ProviderRecord 自动跟进。
export type VendorForm = Omit<AutomationCard, 'id' | 'tint' | 'accent'>

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
