<template>
  <article
    :class="['automation-card', { dragging: isDragging }]"
    draggable="true"
    @dragstart="handleDragStart"
    @dragover.prevent
    @dragend="$emit('drag-end')"
    @drop="$emit('drop', card.id)"
  >
    <div class="card-leading">
      <div class="card-icon" :style="{ backgroundColor: card.tint, color: card.accent }">
        <AsyncIcon :icon="card.icon" :fallback-text="fallbackInitials" />
      </div>

      <div class="card-text">
        <div class="card-title-row">
          <p class="card-title">{{ card.name }}</p>
          <span
            v-if="card.officialSite"
            class="card-site"
            role="button"
            tabindex="0"
            @click.stop="openOfficialSite(card.officialSite)"
            @keydown.enter.stop.prevent="openOfficialSite(card.officialSite)"
            @keydown.space.stop.prevent="openOfficialSite(card.officialSite)"
          >
            {{ officialSiteLabel }}
          </span>
        </div>

        <p class="card-metrics">
          <template v-if="providerStats.state !== 'ready'">
            {{ providerStats.message }}
          </template>
          <template v-else>
            <span
              v-if="providerStats.successRateLabel"
              class="card-success-rate"
              :class="providerStats.successRateClass"
            >
              {{ providerStats.successRateLabel }}
            </span>
            <span class="card-metric-separator" aria-hidden="true">·</span>
            <span>{{ providerStats.requests }}</span>
            <span class="card-metric-separator" aria-hidden="true">·</span>
            <span>{{ providerStats.tokens }}</span>
            <span class="card-metric-separator" aria-hidden="true">·</span>
            <span>{{ providerStats.cost }}</span>
          </template>
        </p>
      </div>
    </div>

    <div class="card-actions">
      <label class="mac-switch sm">
        <input
          type="checkbox"
          :checked="card.enabled"
          @change="handleEnabledChange"
        />
        <span></span>
      </label>

      <button
        class="ghost-icon pin-icon"
        :class="{ 'is-top': isTopProvider }"
        :title="isTopProvider ? t('components.main.pinnedToTop') : t('components.main.pinToTop')"
        :aria-label="isTopProvider ? t('components.main.pinnedToTop') : t('components.main.pinToTop')"
        type="button"
        @click="$emit('pin', card.id)"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            v-if="isTopProvider"
            d="M9 3.75h6v5.1l2.6 2.6v1.8H13.5v7l-1.5-1.5v-5.5H6.4v-1.8L9 8.85z"
            fill="currentColor"
          />
          <path
            v-else
            d="M9 3.75h6v5.1l2.6 2.6v1.8H13.5v7l-1.5-1.5v-5.5H6.4v-1.8L9 8.85z"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>

      <button
        class="ghost-icon star-icon"
        :class="{ 'is-default': isDefaultProvider }"
        :title="isDefaultProvider ? t('components.main.removeDefault') : t('components.main.setDefault')"
        type="button"
        @click="$emit('toggle-default', card)"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            v-if="isDefaultProvider"
            d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.563.563 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z"
            fill="currentColor"
          />
          <path
            v-else
            d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.563.563 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>

      <button class="ghost-icon" type="button" @click="$emit('configure', card)">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            d="M11.983 2.25a1.125 1.125 0 011.077.81l.563 2.101a7.482 7.482 0 012.326 1.343l2.08-.621a1.125 1.125 0 011.356.651l1.313 3.207a1.125 1.125 0 01-.442 1.339l-1.86 1.205a7.418 7.418 0 010 2.686l1.86 1.205a1.125 1.125 0 01.442 1.339l-1.313 3.207a1.125 1.125 0 01-1.356.651l-2.08-.621a7.482 7.482 0 01-2.326 1.343l-.563 2.101a1.125 1.125 0 01-1.077.81h-2.634a1.125 1.125 0 01-1.077-.81l-.563-2.101a7.482 7.482 0 01-2.326-1.343l-2.08.621a1.125 1.125 0 01-1.356-.651l-1.313-3.207a1.125 1.125 0 01.442-1.339l1.86-1.205a7.418 7.418 0 010-2.686l-1.86-1.205a1.125 1.125 0 01-.442-1.339l1.313-3.207a1.125 1.125 0 011.356-.651l2.08.621a7.482 7.482 0 012.326-1.343l.563-2.101a1.125 1.125 0 011.077-.81h2.634z"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
          <path d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      </button>

      <button class="ghost-icon" type="button" @click="$emit('remove', card)">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            d="M9 3h6m-7 4h8m-6 0v11m4-11v11M5 7h14l-.867 12.138A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.862L5 7z"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Browser } from '@wailsio/runtime'
import type { AutomationCard } from '../../data/cards'
import AsyncIcon from '../common/AsyncIcon.vue'
import type { ProviderStatDisplay } from './types'

const props = defineProps<{
  card: AutomationCard
  isDragging: boolean
  isTopProvider: boolean
  isDefaultProvider: boolean
  providerStats: ProviderStatDisplay
}>()

const emit = defineEmits<{
  configure: [card: AutomationCard]
  'drag-end': []
  dragstart: [id: number, event: DragEvent]
  drop: [id: number]
  pin: [cardId: number]
  remove: [card: AutomationCard]
  'toggle-default': [card: AutomationCard]
  'toggle-enabled': [card: AutomationCard, enabled: boolean]
}>()

const { t } = useI18n()

const fallbackInitials = computed(() => {
  if (!props.card.name) return 'AI'
  return props.card.name
    .split(/\s+/)
    .filter(Boolean)
    .map((word) => word[0])
    .join('')
    .slice(0, 2)
    .toUpperCase()
})

const normalizeUrlWithScheme = (value: string) => {
  if (!value) return ''
  try {
    return new URL(value).toString()
  } catch {
    return `https://${value}`
  }
}

const officialSiteLabel = computed(() => {
  if (!props.card.officialSite) return ''
  try {
    const url = new URL(normalizeUrlWithScheme(props.card.officialSite))
    return url.hostname.replace(/^www\./, '')
  } catch {
    return props.card.officialSite
  }
})

const openOfficialSite = (site: string) => {
  const target = normalizeUrlWithScheme(site)
  if (!target) return

  Browser.OpenURL(target).catch(() => {
    console.error('failed to open link', target)
  })
}

const handleDragStart = (event: DragEvent) => {
  event.dataTransfer?.setData('text/plain', String(props.card.id))
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
  emit('dragstart', props.card.id, event)
}

const handleEnabledChange = (event: Event) => {
  const target = event.target as HTMLInputElement | null
  emit('toggle-enabled', props.card, Boolean(target?.checked))
}
</script>

<style scoped>
.star-icon {
  position: relative;
  transition: all 0.2s ease;
}

.pin-icon {
  position: relative;
  transition: all 0.2s ease;
}

.pin-icon svg {
  width: 18px;
  height: 18px;
  transition: all 0.2s ease;
}

.pin-icon:not(.is-top) {
  opacity: 0.55;
}

.pin-icon:not(.is-top):hover {
  opacity: 0.85;
  transform: scale(1.08);
}

.pin-icon.is-top {
  color: #0f766e;
}

.pin-icon.is-top:hover {
  transform: scale(1.08);
}

:global(.dark) .pin-icon.is-top {
  color: #2dd4bf;
}

.star-icon svg {
  width: 18px;
  height: 18px;
  transition: all 0.2s ease;
}

.star-icon:not(.is-default) {
  opacity: 0.5;
}

.star-icon:not(.is-default):hover {
  opacity: 0.8;
  transform: scale(1.1);
}

.star-icon.is-default {
  color: #f59e0b;
}

.star-icon.is-default:hover {
  transform: scale(1.15) rotate(15deg);
}

:global(.dark) .star-icon.is-default {
  color: #fbbf24;
}
</style>
