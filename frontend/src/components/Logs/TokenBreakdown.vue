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
  </td>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RequestLog } from '../../services/logs'

defineProps<{
  log: RequestLog
  formatNumber: (value?: number) => string
}>()

const { t } = useI18n()
</script>
