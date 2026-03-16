<template>
  <section
    class="contrib-wall"
    :aria-label="t('components.main.heatmap.ariaLabel')"
  >
    <div class="contrib-legend">
      <span>{{ t('components.main.heatmap.legendLow') }}</span>
      <span
        v-for="level in 5"
        :key="level"
        :class="['legend-box', intensityClass(level - 1)]"
      />
      <span>{{ t('components.main.heatmap.legendHigh') }}</span>
    </div>

    <div class="contrib-grid">
      <div
        v-for="(week, weekIndex) in usageHeatmap"
        :key="weekIndex"
        class="contrib-column"
      >
        <div
          v-for="(day, dayIndex) in week"
          :key="dayIndex"
          class="contrib-cell"
          :class="intensityClass(day.intensity)"
          @mouseenter="showUsageTooltip(day, $event)"
          @mousemove="showUsageTooltip(day, $event)"
          @mouseleave="hideUsageTooltip"
        />
      </div>
    </div>

    <div
      v-if="usageTooltip.visible"
      ref="tooltipRef"
      class="contrib-tooltip"
      :class="usageTooltip.placement"
      :style="{ left: `${usageTooltip.left}px`, top: `${usageTooltip.top}px` }"
    >
      <p class="tooltip-heading">{{ formattedTooltipLabel }}</p>
      <ul class="tooltip-metrics">
        <li v-for="metric in usageTooltipMetrics" :key="metric.key">
          <span class="metric-label">{{ metric.label }}</span>
          <span class="metric-value">{{ metric.value }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageHeatmapDay, UsageHeatmapWeek } from '../../data/usageHeatmap'

type TooltipPlacement = 'above' | 'below'

type TooltipState = {
  visible: boolean
  label: string
  dateKey: string
  left: number
  top: number
  placement: TooltipPlacement
  requests: number
  inputTokens: number
  outputTokens: number
  reasoningTokens: number
  cost: number
}

const TOOLTIP_DEFAULT_WIDTH = 220
const TOOLTIP_DEFAULT_HEIGHT = 120
const TOOLTIP_VERTICAL_OFFSET = 12
const TOOLTIP_HORIZONTAL_MARGIN = 20
const TOOLTIP_VERTICAL_MARGIN = 24

defineProps<{
  usageHeatmap: UsageHeatmapWeek[]
}>()

const { t, locale } = useI18n()
const tooltipRef = ref<HTMLElement | null>(null)

const usageTooltip = reactive<TooltipState>({
  visible: false,
  label: '',
  dateKey: '',
  left: 0,
  top: 0,
  placement: 'above',
  requests: 0,
  inputTokens: 0,
  outputTokens: 0,
  reasoningTokens: 0,
  cost: 0,
})

const intensityClass = (value: number) => `gh-level-${value}`

const formatMetric = (value: number) => value.toLocaleString()

const tooltipDateFormatter = computed(
  () =>
    new Intl.DateTimeFormat(locale.value || 'en', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }),
)

const currencyFormatter = computed(
  () =>
    new Intl.NumberFormat(locale.value || 'en', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }),
)

const formattedTooltipLabel = computed(() => {
  if (!usageTooltip.dateKey) return usageTooltip.label

  const date = new Date(usageTooltip.dateKey)
  if (Number.isNaN(date.getTime())) {
    return usageTooltip.label
  }

  return tooltipDateFormatter.value.format(date)
})

const usageTooltipMetrics = computed(() => [
  {
    key: 'cost',
    label: t('components.main.heatmap.metrics.cost'),
    value: currencyFormatter.value.format(Math.max(usageTooltip.cost, 0)),
  },
  {
    key: 'requests',
    label: t('components.main.heatmap.metrics.requests'),
    value: formatMetric(usageTooltip.requests),
  },
  {
    key: 'inputTokens',
    label: t('components.main.heatmap.metrics.inputTokens'),
    value: formatMetric(usageTooltip.inputTokens),
  },
  {
    key: 'outputTokens',
    label: t('components.main.heatmap.metrics.outputTokens'),
    value: formatMetric(usageTooltip.outputTokens),
  },
  {
    key: 'reasoningTokens',
    label: t('components.main.heatmap.metrics.reasoningTokens'),
    value: formatMetric(usageTooltip.reasoningTokens),
  },
])

const clamp = (value: number, min: number, max: number) => {
  if (max <= min) return min
  return Math.min(Math.max(value, min), max)
}

const getTooltipSize = () => {
  const rect = tooltipRef.value?.getBoundingClientRect()
  return {
    width: rect?.width ?? TOOLTIP_DEFAULT_WIDTH,
    height: rect?.height ?? TOOLTIP_DEFAULT_HEIGHT,
  }
}

const getViewportSize = () => {
  if (typeof window !== 'undefined') {
    return { width: window.innerWidth, height: window.innerHeight }
  }

  if (typeof document !== 'undefined' && document.documentElement) {
    return {
      width: document.documentElement.clientWidth,
      height: document.documentElement.clientHeight,
    }
  }

  return { width: 0, height: 0 }
}

const showUsageTooltip = (day: UsageHeatmapDay, event: MouseEvent) => {
  const target = event.currentTarget as HTMLElement | null
  const cellRect = target?.getBoundingClientRect()
  if (!cellRect) return

  usageTooltip.label = day.label
  usageTooltip.dateKey = day.dateKey
  usageTooltip.requests = day.requests
  usageTooltip.inputTokens = day.inputTokens
  usageTooltip.outputTokens = day.outputTokens
  usageTooltip.reasoningTokens = day.reasoningTokens
  usageTooltip.cost = day.cost

  const { width: tooltipWidth, height: tooltipHeight } = getTooltipSize()
  const { width: viewportWidth, height: viewportHeight } = getViewportSize()
  const centerX = cellRect.left + cellRect.width / 2
  const halfWidth = tooltipWidth / 2
  const minLeft = TOOLTIP_HORIZONTAL_MARGIN + halfWidth
  const maxLeft =
    viewportWidth > 0
      ? viewportWidth - halfWidth - TOOLTIP_HORIZONTAL_MARGIN
      : centerX

  usageTooltip.left = clamp(centerX, minLeft, maxLeft)

  const anchorTop = cellRect.top
  const anchorBottom = cellRect.bottom
  const canShowAbove =
    anchorTop - tooltipHeight - TOOLTIP_VERTICAL_OFFSET >= TOOLTIP_VERTICAL_MARGIN
  const viewportBottomLimit =
    viewportHeight > 0
      ? viewportHeight - tooltipHeight - TOOLTIP_VERTICAL_MARGIN
      : anchorBottom
  const shouldPlaceBelow = !canShowAbove

  usageTooltip.placement = shouldPlaceBelow ? 'below' : 'above'

  const desiredTop = shouldPlaceBelow
    ? anchorBottom + TOOLTIP_VERTICAL_OFFSET
    : anchorTop - tooltipHeight - TOOLTIP_VERTICAL_OFFSET

  usageTooltip.top = clamp(
    desiredTop,
    TOOLTIP_VERTICAL_MARGIN,
    viewportBottomLimit,
  )
  usageTooltip.visible = true
}

const hideUsageTooltip = () => {
  usageTooltip.visible = false
}
</script>
