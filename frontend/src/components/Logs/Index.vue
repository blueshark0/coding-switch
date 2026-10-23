<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLogsDashboard } from './useLogsDashboard'
import { providerTabs } from '../../constants/platforms'
import { formatBeijingDateTime } from '../../utils/dateTime'
import type { RequestLog } from '../../services/logs'

const { t } = useI18n()

const {
  filters,
  providerOptions,
  rangeOptions,
  httpStatusOptions,
  pagedLogs,
  stats,
  loading,
  page,
  totalPages,
  totalLogs,
  applyFilters,
  manualRefresh,
  nextPage,
  prevPage,
  exportCsv,
} = useLogsDashboard({ t })

const platformOptions = computed(() => [
  { value: '', label: t('logsPage.filters.allPlatforms') },
  ...providerTabs.map((tab) => ({ value: tab.id as string, label: tab.label })),
])

const metrics = computed(() => {
  const total = stats.value?.total_requests ?? 0
  const failed = stats.value?.failed_requests ?? 0
  const avg = stats.value?.avg_duration_sec ?? 0
  const successRate = total > 0 ? ((total - failed) / total) * 100 : null
  return [
    { key: 'requests', value: total.toLocaleString() },
    { key: 'successRate', value: successRate === null ? '—' : `${successRate.toFixed(1)}%` },
    { key: 'avgLatency', value: total > 0 ? `${avg.toFixed(2)} s` : '—' },
    { key: 'errors', value: failed.toLocaleString(), tone: failed > 0 ? 'danger' : '' },
  ]
})

const platformLabel = (platform: string) =>
  providerTabs.find((tab) => tab.id === platform)?.label ?? platform

const formatTime = (value: string) => formatBeijingDateTime(value)

const formatDuration = (sec: number) => (sec >= 10 ? `${sec.toFixed(1)}s` : `${sec.toFixed(2)}s`)

const statusOf = (log: RequestLog) => {
  if (log.http_code >= 200 && log.http_code < 300) {
    return { tone: 'success', label: t('logsPage.status.success') }
  }
  if (log.http_code === 429) {
    return { tone: 'danger', label: t('logsPage.status.rateLimited') }
  }
  if (log.http_code >= 500) {
    return { tone: 'danger', label: t('logsPage.status.serverError', { code: log.http_code }) }
  }
  return { tone: 'danger', label: t('logsPage.status.failed', { code: log.http_code }) }
}
</script>

<template>
  <div class="cs-page">
    <div class="cs-page-head">
      <div>
        <h1 class="cs-page-title">{{ t('logsPage.title') }}</h1>
        <p class="cs-page-sub">{{ t('logsPage.subtitle') }}</p>
      </div>
    </div>

    <div class="logs-metrics">
      <div v-for="metric in metrics" :key="metric.key" class="cs-card logs-metric">
        <p class="logs-metric-label">{{ t(`logsPage.metrics.${metric.key}`) }}</p>
        <p class="logs-metric-value cs-mono" :class="{ 'logs-metric-value--danger': metric.tone === 'danger' }">
          {{ metric.value }}
        </p>
      </div>
    </div>

    <div class="cs-card cs-card-pad logs-panel">
      <div class="logs-panel-head">
        <div>
          <h2 class="cs-card-title">{{ t('logsPage.recent.title') }}</h2>
          <p class="cs-section-sub">{{ t('logsPage.recent.subtitle') }}</p>
        </div>
        <button type="button" class="cs-btn" :disabled="pagedLogs.length === 0" @click="exportCsv">
          {{ t('logsPage.actions.export') }}
        </button>
      </div>

      <div class="cs-filterbar cs-filterbar--inset">
        <select v-model="filters.platform" class="cs-select logs-filter" @change="applyFilters">
          <option v-for="opt in platformOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <select v-model="filters.provider" class="cs-select logs-filter" @change="applyFilters">
          <option value="">{{ t('logsPage.filters.allProviders') }}</option>
          <option v-for="provider in providerOptions" :key="provider" :value="provider">{{ provider }}</option>
        </select>
        <select v-model="filters.httpStatus" class="cs-select logs-filter" @change="applyFilters">
          <option v-for="opt in httpStatusOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <select v-model="filters.rangeKey" class="cs-select logs-filter" @change="applyFilters">
          <option v-for="opt in rangeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <button type="button" class="cs-btn logs-refresh" @click="manualRefresh">
          {{ t('logsPage.actions.refresh') }}
        </button>
      </div>

      <div v-if="pagedLogs.length === 0 && !loading" class="cs-empty">
        {{ t('logsPage.empty') }}
      </div>

      <div v-else class="logs-rows">
        <div class="logs-row logs-row--head">
          <span>{{ t('logsPage.columns.time') }}</span>
          <span>{{ t('logsPage.columns.client') }}</span>
          <span>{{ t('logsPage.columns.provider') }}</span>
          <span>{{ t('logsPage.columns.model') }}</span>
          <span class="logs-cell-num">{{ t('logsPage.columns.httpCode') }}</span>
          <span class="logs-cell-num">{{ t('logsPage.columns.duration') }}</span>
          <span class="logs-cell-status">{{ t('logsPage.columns.status') }}</span>
        </div>
        <div
          v-for="(log, index) in pagedLogs"
          :key="log.id"
          class="logs-row"
          :class="{ 'logs-row--alt': index % 2 === 0, 'logs-row--error': log.http_code >= 400 }"
        >
          <span class="logs-cell-time cs-mono" :title="formatTime(log.created_at)">{{ formatTime(log.created_at) }}</span>
          <span class="logs-cell-ellipsis">{{ platformLabel(log.platform) }}</span>
          <span class="logs-cell-ellipsis" :title="log.provider">{{ log.provider }}</span>
          <span class="logs-cell-ellipsis" :title="log.model">{{ log.model }}</span>
          <span class="logs-cell-num cs-mono">{{ log.http_code }}</span>
          <span class="logs-cell-num cs-mono">{{ formatDuration(log.duration_sec) }}</span>
          <span class="logs-cell-status">
            <span class="cs-pill" :class="`cs-pill--${statusOf(log).tone}`">{{ statusOf(log).label }}</span>
          </span>
        </div>
      </div>

      <div class="logs-pager">
        <span class="logs-pager-info">{{ t('logsPage.pager.total', { total: totalLogs }) }}</span>
        <button type="button" class="cs-btn" :disabled="page <= 1" @click="prevPage">
          {{ t('logsPage.pager.prev') }}
        </button>
        <span class="logs-pager-page cs-mono">{{ page }} / {{ totalPages }}</span>
        <button type="button" class="cs-btn" :disabled="page >= totalPages" @click="nextPage">
          {{ t('logsPage.pager.next') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.logs-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 18px;
}

.logs-metric {
  padding: 13px 15px;
  min-height: 94px;
}

.logs-metric-label {
  margin: 0;
  font-size: 11px;
  color: var(--cs-text-secondary);
}

.logs-metric-value {
  margin: 10px 0 0;
  font-size: 23px;
  font-weight: 700;
  color: var(--cs-text);
}

.logs-metric-value--danger {
  color: var(--cs-danger);
}

.logs-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.logs-panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.logs-filter {
  width: auto;
  min-width: 128px;
  min-height: 32px;
  border-color: transparent;
  background-color: transparent;
}

.logs-refresh {
  margin-left: auto;
  height: 30px;
}

.logs-rows {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.logs-row {
  display: grid;
  grid-template-columns: 160px 96px minmax(80px, 1fr) minmax(110px, 1.2fr) 64px 72px 76px;
  align-items: center;
  gap: 12px;
  min-height: 40px;
  padding: 0 12px;
  border-radius: 6px;
  font-size: 12px;
  color: var(--cs-text);
}

.logs-row--head {
  min-height: 30px;
  font-size: 11px;
  color: var(--cs-text-secondary);
}

.logs-row--alt:not(.logs-row--head) {
  background: var(--cs-field);
}

.logs-cell-time {
  white-space: nowrap;
}

.logs-cell-ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.logs-cell-num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.logs-cell-status {
  display: flex;
  justify-content: flex-end;
}

.logs-row--error .logs-cell-num {
  color: var(--cs-danger);
}

.logs-pager {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.logs-pager-info {
  margin-right: auto;
  font-size: 11px;
  color: var(--cs-text-secondary);
}

.logs-pager-page {
  font-size: 11px;
  color: var(--cs-text-secondary);
}

@media (max-width: 1100px) {
  .logs-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
