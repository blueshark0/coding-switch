<template>
  <div class="main-shell">
    <div class="global-actions home-actions">
      <p class="global-eyebrow">{{ t('components.main.hero.eyebrow') }}</p>
      <button
        class="ghost-icon"
        :data-tooltip="t('components.main.controls.theme')"
        @click="toggleTheme"
      >
        <svg v-if="themeIcon === 'sun'" viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="12" cy="12" r="4" stroke="currentColor" stroke-width="1.5" fill="none" />
          <path
            d="M12 3v2m0 14v2m9-9h-2M5 12H3m14.95 6.95-1.41-1.41M7.46 7.46 6.05 6.05m12.9 0-1.41 1.41M7.46 16.54l-1.41 1.41"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
          />
        </svg>
        <svg v-else viewBox="0 0 24 24" aria-hidden="true">
          <path
            d="M21 12.79A9 9 0 1111.21 3a7 7 0 109.79 9.79z"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <button
        class="ghost-icon"
        :data-tooltip="t('components.main.controls.settings')"
        @click="goToSettings"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            d="M12 15a3 3 0 100-6 3 3 0 000 6z"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            fill="none"
          />
          <path
            d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 01-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09a1.65 1.65 0 00-1-1.51 1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09a1.65 1.65 0 001.51-1 1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            fill="none"
          />
        </svg>
      </button>
    </div>
    <div class="contrib-page">
      <section class="contrib-hero">
        <h1 v-if="showHomeTitle">{{ t('components.main.hero.title') }}</h1>
        <!-- <p class="lead">
          {{ t('components.main.hero.lead') }}
        </p> -->
      </section>

      <HeatmapWall v-if="showHeatmap" :usage-heatmap="usageHeatmap" />

      <section class="automation-section">
      <div class="section-header">
        <div class="tab-group" role="tablist" :aria-label="t('components.main.tabs.ariaLabel')">
          <button
            v-for="(tab, idx) in tabs"
            :key="tab.id"
            class="tab-pill"
            :class="{ active: selectedIndex === idx }"
            role="tab"
            :aria-selected="selectedIndex === idx"
            type="button"
            @click="onTabChange(idx)"
          >
            {{ tab.label }}
          </button>
        </div>
        <div class="section-controls">
          <div class="relay-toggle" :aria-label="currentProxyLabel">
            <div class="relay-switch">
              <label class="mac-switch sm">
                <input
                  type="checkbox"
                  :checked="activeProxyState"
                  :disabled="activeProxyBusy"
                  @change="onProxyToggle"
                />
                <span></span>
              </label>
              <span class="relay-tooltip-content">{{ currentProxyLabel }} · {{ t('components.main.relayToggle.tooltip') }}</span>
            </div>
          </div>
          <button
            class="ghost-icon"
            :data-tooltip="t('components.main.logs.view')"
            @click="goToLogs"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M5 7h14M5 12h14M5 17h9"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                fill="none"
              />
            </svg>
          </button>
          <button
            class="ghost-icon"
            :data-tooltip="t('components.main.controls.sessions')"
            @click="goToSessions"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M9.5 14.5l5-5"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                fill="none"
              />
              <rect
                x="3.5"
                y="10"
                width="8"
                height="8"
                rx="2.5"
                stroke="currentColor"
                stroke-width="1.5"
                fill="none"
              />
              <rect
                x="12.5"
                y="4"
                width="8"
                height="8"
                rx="2.5"
                stroke="currentColor"
                stroke-width="1.5"
                fill="none"
              />
            </svg>
          </button>
          <button
            class="ghost-icon"
            :data-tooltip="t('components.main.tabs.addCard')"
            @click="openCreateModal"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M12 5v14M5 12h14"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
                fill="none"
              />
            </svg>
          </button>
        </div>
      </div>
      <div class="automation-list" @dragover.prevent>
        <ProviderCard
          v-for="card in activeCards"
          :key="card.id"
          :card="card"
          :is-dragging="draggingId === card.id"
          :is-top-provider="isTopProvider(card.id)"
          :is-default-provider="isDefaultProvider(card.name)"
          :provider-stats="providerStatDisplay(card.name)"
          @configure="configure"
          @drag-end="onDragEnd"
          @dragstart="onDragStart"
          @drop="onDrop"
          @pin="pinProvider"
          @remove="requestRemove"
          @toggle-default="toggleDefaultProvider"
        />
      </div>
      </section>

      <ProviderEditorModal
        :api-url-error="modalState.errors.apiUrl"
        :editing="Boolean(modalState.editingId)"
        :form="modalState.form"
        :name-error="modalState.errors.name"
        :open="modalState.open"
        :proxy-url-error="modalState.errors.proxyUrl"
        @close="closeModal"
        @submit="submitModal"
      />
      <ProviderDeleteModal
        :name="confirmState.card?.name ?? ''"
        :open="confirmState.open"
        @close="closeConfirm"
        @confirm="confirmRemove"
      />

      <footer v-if="appVersion" class="main-version">
        {{ t('components.main.versionLabel', { version: appVersion }) }}
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HeatmapWall from './HeatmapWall.vue'
import ProviderDeleteModal from './ProviderDeleteModal.vue'
import ProviderEditorModal from './ProviderEditorModal.vue'
import ProviderCard from './ProviderCard.vue'
import { setTheme } from '../../utils/ThemeManager'
import { useProviderWorkbench } from './useProviderWorkbench'
import { useRouter } from 'vue-router'
import { useDarkMode } from '../../composables/useDarkMode'

const { t, locale } = useI18n()
const router = useRouter()
const { isDarkMode } = useDarkMode()
const themeIcon = computed(() => (isDarkMode.value ? 'moon' : 'sun'))
const {
  activeTab,
  activeCards,
  activeProxyBusy,
  activeProxyState,
  appVersion,
  closeConfirm,
  closeModal,
  confirmRemove,
  confirmState,
  configure,
  currentProxyLabel,
  draggingId,
  isDefaultProvider,
  isTopProvider,
  modalState,
  onDragEnd,
  onDragStart,
  onDrop,
  onProxyToggle,
  onTabChange,
  openCreateModal,
  pinProvider,
  providerStatDisplay,
  requestRemove,
  selectedIndex,
  showHeatmap,
  showHomeTitle,
  submitModal,
  tabs,
  toggleDefaultProvider,
  usageHeatmap,
} = useProviderWorkbench({
  locale,
  t,
})

const goToLogs = () => {
  router.push('/logs')
}

const goToSessions = () => {
  router.push({ path: '/sessions', query: { platform: activeTab.value } })
}

const goToSettings = () => {
  router.push('/settings')
}

const toggleTheme = () => {
  const next = isDarkMode.value ? 'light' : 'dark'
  setTheme(next)
}
</script>

<style scoped>
.main-version {
  margin: 32px auto 12px;
  text-align: center;
  color: var(--mac-text-secondary);
  font-size: 0.85rem;
}
</style>
