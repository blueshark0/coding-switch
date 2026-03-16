<template>
  <div class="main-shell">
    <div class="global-actions">
      <p class="global-eyebrow">{{ t('components.sessions.hero.eyebrow') }}</p>
      <button class="ghost-icon" :data-tooltip="t('components.sessions.actions.back')" @click="goHome">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            d="M15 18l-6-6 6-6"
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
        :data-tooltip="t('components.sessions.actions.refresh')"
        :disabled="loading"
        @click="refreshSessions"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true" :class="{ spin: loading }">
          <path
            d="M20.5 8a8.5 8.5 0 10-2.38 7.41"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
          <path
            d="M20.5 4v4h-4"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </div>

    <div class="contrib-page sessions-page">
      <section class="contrib-hero">
        <h1>{{ t('components.sessions.hero.title') }}</h1>
        <p class="lead">{{ t('components.sessions.hero.lead') }}</p>
      </section>

      <section class="automation-section">
        <div class="section-header">
          <div class="tab-group" role="tablist" :aria-label="t('components.sessions.platforms.ariaLabel')">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              class="tab-pill"
              :class="{ active: selectedPlatform === tab.id }"
              role="tab"
              :aria-selected="selectedPlatform === tab.id"
              @click="switchPlatform(tab.id)"
            >
              {{ tab.label }}
            </button>
          </div>
          <div class="section-controls">
            <BaseButton variant="outline" size="sm" :disabled="loading" @click="refreshSessions">
              {{ t('components.sessions.actions.refresh') }}
            </BaseButton>
          </div>
        </div>
        <SessionGroups
          :format-relative-time="formatRelativeTime"
          :groups="providerGroups"
          :loading="loading"
          :truncate-session-id="truncateSessionId"
          @unbind="unbindSession"
        />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '../common/BaseButton.vue'
import SessionGroups from './SessionGroups.vue'
import { usePlatformSessions } from './usePlatformSessions'
import type { ProviderTab } from '../../constants/platforms'

const { t, locale } = useI18n()
const router = useRouter()
const route = useRoute()

const syncPlatformQuery = async (platform: ProviderTab) => {
  await router.replace({ path: '/sessions', query: { platform } })
}

const {
  formatRelativeTime,
  initializeSessions,
  loading,
  providerGroups,
  refreshSessions,
  selectedPlatform,
  switchPlatform,
  tabs,
  truncateSessionId,
  unbindSession,
} = usePlatformSessions({
  locale,
  syncPlatformQuery,
  t,
})

const goHome = () => {
  void router.push('/')
}

onMounted(() => {
  void initializeSessions(route.query.platform)
})
</script>

<style scoped>
.sessions-page {
  gap: 18px;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
