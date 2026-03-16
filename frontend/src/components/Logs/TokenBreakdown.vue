<template>
  <td class="token-cell">
    <div>
      <span class="token-label">{{ t('components.logs.tokenLabels.input') }}</span>
      <span class="token-value">{{ formatNumber(log.input_tokens) }}</span>
    </div>
    <div>
      <span class="token-label">{{ t('components.logs.tokenLabels.output') }}</span>
      <span class="token-value">{{ formatNumber(log.output_tokens) }}</span>
    </div>
    <div>
      <span class="token-label">{{ t('components.logs.tokenLabels.reasoning') }}</span>
      <span class="token-value">{{ formatNumber(log.reasoning_tokens) }}</span>
    </div>
    <div>
      <span class="token-label">{{ t('components.logs.tokenLabels.cacheWrite') }}</span>
      <span class="token-value">{{ formatNumber(log.cache_create_tokens) }}</span>
    </div>
    <div>
      <span class="token-label">{{ t('components.logs.tokenLabels.cacheRead') }}</span>
      <span class="token-value">{{ formatNumber(log.cache_read_tokens) }}</span>
    </div>
    <div class="token-cost-row">
      <span class="token-label">{{ t('components.logs.tokenLabels.cost') }}</span>
      <span :class="['token-value', costValueClass(log)]">
        {{ log.has_pricing ? formatCurrency(log.total_cost) : '—' }}
      </span>
    </div>
  </td>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RequestLog } from '../../services/logs'

defineProps<{
  log: RequestLog
  formatCurrency: (value?: number) => string
  formatNumber: (value?: number) => string
}>()

const { t } = useI18n()

const costValueClass = (log: RequestLog) => {
  if (!log.has_pricing) return ''
  if (log.total_cost < 0.1) return 'token-cost-low'
  if (log.total_cost > 0.5) return 'token-cost-high'
  return 'token-cost-medium'
}
</script>
