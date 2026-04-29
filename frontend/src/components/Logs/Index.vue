<template>
  <div class="logs-page">
    <div class="logs-header">
      <BaseButton variant="outline" type="button" @click="backToHome">
        {{ t('components.logs.back') }}
      </BaseButton>
      <div class="refresh-indicator">
        <span>{{ t('components.logs.nextRefresh', { seconds: countdown }) }}</span>
        <BaseButton size="sm" :disabled="loading" @click="manualRefresh">
          {{ t('components.logs.refresh') }}
        </BaseButton>
      </div>
    </div>

    <section class="logs-summary" v-if="statsCards.length">
      <article v-for="card in statsCards" :key="card.key" class="summary-card">
        <div class="summary-card__label">{{ card.label }}</div>
        <div class="summary-card__value">{{ card.value }}</div>
        <div class="summary-card__hint">{{ card.hint }}</div>
      </article>
    </section>

    <section class="logs-chart">
      <Line :data="chartData" :options="chartOptions" />
    </section>

    <form class="logs-filter-row" @submit.prevent="applyFilters">
      <div class="filter-fields">
        <label class="filter-field">
          <span>{{ t('components.logs.filters.platform') }}</span>
          <select v-model="filters.platform" class="mac-select">
            <option value="">{{ t('components.logs.filters.allPlatforms') }}</option>
            <option value="claude">Claude</option>
            <option value="codex">Codex</option>
            <option value="gemini">Gemini</option>
          </select>
        </label>
        <label class="filter-field">
          <span>{{ t('components.logs.filters.provider') }}</span>
          <select v-model="filters.provider" class="mac-select">
            <option value="">{{ t('components.logs.filters.allProviders') }}</option>
            <option v-for="provider in providerOptions" :key="provider" :value="provider">
              {{ provider }}
            </option>
          </select>
        </label>
        <label class="filter-field">
          <span>{{ t('components.logs.filters.range') }}</span>
          <select v-model="filters.rangeKey" class="mac-select">
            <option v-for="option in rangeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>
        <label class="filter-field">
          <span>{{ t('components.logs.filters.costTier') }}</span>
          <select v-model="filters.costTier" class="mac-select">
            <option v-for="option in costTierOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>
      </div>
      <div class="filter-actions">
        <BaseButton type="submit" :disabled="loading">
          {{ t('components.logs.query') }}
        </BaseButton>
      </div>
    </form>

    <section class="logs-table-wrapper">
      <table class="logs-table">
        <thead>
          <tr>
            <th class="col-time">{{ t('components.logs.table.time') }}</th>
            <th class="col-platform">{{ t('components.logs.table.platform') }}</th>
            <th class="col-provider">{{ t('components.logs.table.provider') }}</th>
            <th class="col-model">{{ t('components.logs.table.model') }}</th>
            <th class="col-http">{{ t('components.logs.table.httpCode') }}</th>
            <th class="col-stream">{{ t('components.logs.table.stream') }}</th>
            <th class="col-duration">{{ t('components.logs.table.duration') }}</th>
            <th class="col-tokens">{{ t('components.logs.table.tokens') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in pagedLogs" :key="item.id">
            <td>{{ formatTime(item.created_at) }}</td>
            <td>{{ item.platform || '—' }}</td>
            <td>{{ item.provider || '—' }}</td>
            <td>{{ item.model || '—' }}</td>
            <td :class="['code', httpCodeClass(item.http_code)]">{{ item.http_code }}</td>
            <td><span :class="['stream-tag', item.is_stream ? 'on' : 'off']">{{ formatStream(item.is_stream) }}</span></td>
            <td><span :class="['duration-tag', durationColor(item.duration_sec)]">{{ formatDuration(item.duration_sec) }}</span></td>
            <TokenBreakdown
              :log="item"
              :format-number="formatNumber"
              :format-currency="formatCurrency"
            />
          </tr>
          <tr v-if="!pagedLogs.length && !loading">
            <td colspan="8" class="empty">{{ t('components.logs.empty') }}</td>
          </tr>
        </tbody>
      </table>
      <p v-if="loading" class="empty">{{ t('components.logs.loading') }}</p>
    </section>

    <div class="logs-pagination">
      <label class="page-size-field">
        <span>{{ t('components.logs.pagination.pageSize') }}</span>
        <select v-model.number="pageSize" class="mac-select" :disabled="loading" @change="updatePageSize">
          <option v-for="option in pageSizeOptions" :key="option.value" :value="option.value">
            {{ option.value }}
          </option>
        </select>
      </label>
      <span>{{ t('components.logs.pagination.total', { total: formatNumber(totalLogs) }) }}</span>
      <span>{{ page }} / {{ totalPages }}</span>
      <div class="pagination-actions">
        <BaseButton variant="outline" size="sm" :disabled="page === 1 || loading" @click="prevPage">
          ‹
        </BaseButton>
        <BaseButton variant="outline" size="sm" :disabled="page >= totalPages || loading" @click="nextPage">
          ›
        </BaseButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BaseButton from '../common/BaseButton.vue'
import TokenBreakdown from './TokenBreakdown.vue'
import {
  Chart,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import { useDarkMode } from '../../composables/useDarkMode'
import { useLogsDashboard } from './useLogsDashboard'

Chart.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)

const { t } = useI18n()
const router = useRouter()
const { getCssVarValue, isDarkMode } = useDarkMode()
const {
  applyFilters,
  chartData,
  chartOptions,
  costTierOptions,
  countdown,
  durationColor,
  filters,
  formatDuration,
  formatCurrency,
  formatNumber,
  formatStream,
  formatTime,
  httpCodeClass,
  loading,
  manualRefresh,
  nextPage,
  page,
  pageSize,
  pageSizeOptions,
  pagedLogs,
  prevPage,
  providerOptions,
  rangeOptions,
  statsCards,
  totalLogs,
  totalPages,
  updatePageSize,
} = useLogsDashboard({
  getCssVarValue,
  isDarkMode,
  t,
})

const backToHome = () => {
  router.push('/')
}
</script>

<style scoped>
.logs-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.summary-meta {
  grid-column: 1 / -1;
  font-size: 0.85rem;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #64748b;
}

.summary-card {
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 16px;
  padding: 1rem 1.25rem;
  background: radial-gradient(circle at top, rgba(148, 163, 184, 0.1), rgba(15, 23, 42, 0));
  backdrop-filter: blur(6px);
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.summary-card__label {
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #475569;
}

.summary-card__value {
  font-size: 1.85rem;
  font-weight: 600;
  color: #0f172a;
}

.summary-card__hint {
  font-size: 0.85rem;
  color: #94a3b8;
}

html.dark .summary-card {
  border-color: rgba(255, 255, 255, 0.12);
  background: radial-gradient(circle at top, rgba(148, 163, 184, 0.2), rgba(15, 23, 42, 0.35));
}

html.dark .summary-card__label {
  color: rgba(248, 250, 252, 0.75);
}

html.dark .summary-card__value {
  color: rgba(248, 250, 252, 0.95);
}

html.dark .summary-card__hint {
  color: rgba(186, 194, 210, 0.8);
}

@media (max-width: 768px) {
  .logs-summary {
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  }
}
</style>
