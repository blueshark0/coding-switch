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

        <div v-if="loading" class="sessions-empty">
          {{ t('components.sessions.loading') }}
        </div>
        <div v-else-if="providerGroups.length === 0" class="sessions-empty">
          {{ t('components.sessions.noSessions') }}
        </div>
        <div v-else class="sessions-groups">
          <article v-for="group in providerGroups" :key="group.providerName" class="sessions-group-card">
            <header class="sessions-group-head">
              <h2>{{ group.providerName }}</h2>
              <span>{{ t('components.sessions.groupCount', { count: group.sessions.length }) }}</span>
            </header>
            <div class="sessions-list">
              <div
                v-for="session in group.sessions"
                :key="`${session.platform}-${session.session_id}`"
                class="session-row"
              >
                <div class="session-main">
                  <div class="session-id" :title="session.session_id">
                    {{ truncateSessionId(session.session_id) }}
                  </div>
                  <div class="session-time-row">
                    <span>{{ t('components.sessions.bindTime') }}: {{ formatRelativeTime(session.created_at) }}</span>
                    <span>{{ t('components.sessions.lastSuccess') }}: {{ formatRelativeTime(session.last_success_at) }}</span>
                  </div>
                </div>
                <button class="session-unbind-btn" @click="unbindSession(session)">
                  {{ t('components.sessions.unbind') }}
                </button>
              </div>
            </div>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import BaseButton from '../common/BaseButton.vue'
import { fetchPlatformSessions, unbindSession as requestUnbindSession, type SessionBinding } from '../../services/sessions'

const { t, locale } = useI18n()
const router = useRouter()
const route = useRoute()

const tabs = [
  { id: 'claude', label: 'Claude Code' },
  { id: 'codex', label: 'Codex' },
  { id: 'gemini', label: 'Gemini' },
] as const

type ProviderTab = (typeof tabs)[number]['id']

const selectedPlatform = ref<ProviderTab>('claude')
const sessions = ref<SessionBinding[]>([])
const loading = ref(false)

const dateFormatter = computed(
  () =>
    new Intl.DateTimeFormat(locale.value || 'en', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }),
)

const parsePlatform = (value: unknown): ProviderTab => {
  const raw = typeof value === 'string' ? value : ''
  if (raw === 'claude' || raw === 'codex' || raw === 'gemini') {
    return raw
  }
  return 'claude'
}

const parseDate = (value: unknown): Date | null => {
  if (!value) return null
  const parsed = new Date(value as string)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }
  return parsed
}

const loadSessions = async () => {
  loading.value = true
  try {
    sessions.value = await fetchPlatformSessions(selectedPlatform.value)
  } catch (error) {
    console.error('Failed to load platform sessions', error)
    sessions.value = []
  } finally {
    loading.value = false
  }
}

const switchPlatform = async (platform: ProviderTab) => {
  if (platform === selectedPlatform.value) return
  selectedPlatform.value = platform
  await router.replace({ path: '/sessions', query: { platform } })
  await loadSessions()
}

const refreshSessions = () => {
  void loadSessions()
}

const truncateSessionId = (sessionId: string) => {
  if (sessionId.length <= 12) {
    return sessionId
  }
  return `${sessionId.slice(0, 12)}...`
}

const formatRelativeTime = (value: unknown) => {
  const date = parseDate(value)
  if (!date) return t('components.sessions.dateUnknown')

  const now = Date.now()
  const diffMs = now - date.getTime()
  const diffMinutes = Math.floor(diffMs / 60000)
  if (diffMinutes < 1) {
    return t('components.sessions.justNow')
  }
  if (diffMinutes < 60) {
    return t('components.sessions.minutesAgo', { count: diffMinutes })
  }
  if (diffMinutes < 24 * 60) {
    const diffHours = Math.floor(diffMinutes / 60)
    return t('components.sessions.hoursAgo', { count: diffHours })
  }
  return dateFormatter.value.format(date)
}

const providerGroups = computed(() => {
  const groups = new Map<string, SessionBinding[]>()
  for (const session of sessions.value) {
    const providerName = session.provider_name?.trim() || t('components.sessions.unknownProvider')
    if (!groups.has(providerName)) {
      groups.set(providerName, [])
    }
    groups.get(providerName)?.push(session)
  }
  return Array.from(groups.entries()).map(([providerName, providerSessions]) => ({
    providerName,
    sessions: providerSessions,
  }))
})

const unbindSession = async (session: SessionBinding) => {
  if (!confirm(t('components.sessions.confirmUnbindSession'))) {
    return
  }
  try {
    await requestUnbindSession(session.platform, session.session_id)
    await loadSessions()
  } catch (error) {
    console.error('Failed to unbind session', error)
    alert(t('components.sessions.unbindSessionFailed'))
  }
}

const goHome = () => {
  router.push('/')
}

onMounted(async () => {
  const platform = parsePlatform(route.query.platform)
  selectedPlatform.value = platform
  if (route.query.platform !== platform) {
    await router.replace({ path: '/sessions', query: { platform } })
  }
  await loadSessions()
})
</script>

<style scoped>
.sessions-page {
  gap: 18px;
}

.sessions-empty {
  border: 1px dashed var(--mac-border);
  border-radius: 16px;
  padding: 28px 20px;
  text-align: center;
  color: var(--mac-text-secondary);
  background: color-mix(in srgb, var(--mac-surface-strong) 90%, transparent);
}

.sessions-groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.sessions-group-card {
  border: 1px solid var(--mac-border);
  border-radius: 18px;
  background: var(--mac-surface);
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sessions-group-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.sessions-group-head h2 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
}

.sessions-group-head span {
  font-size: 0.82rem;
  color: var(--mac-text-secondary);
}

.sessions-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.session-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 85%, transparent);
  border-radius: 14px;
  padding: 12px 14px;
}

.session-main {
  min-width: 0;
  flex: 1;
}

.session-id {
  font-family: 'SFMono-Regular', Menlo, Consolas, monospace;
  font-size: 0.86rem;
  margin-bottom: 6px;
  color: var(--mac-text);
}

.session-time-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  font-size: 0.78rem;
  color: var(--mac-text-secondary);
}

.session-unbind-btn {
  border: 1px solid color-mix(in srgb, #dc2626 35%, transparent);
  color: #dc2626;
  background: transparent;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  align-self: flex-start;
  font-size: 0.8rem;
  font-weight: 600;
  line-height: 1;
  height: 30px;
  min-width: 64px;
  padding: 0 10px;
  cursor: pointer;
  transition: background 0.15s ease;
  flex: 0 0 auto;
  white-space: nowrap;
}

.session-unbind-btn:hover {
  background: color-mix(in srgb, #dc2626 10%, transparent);
}

html.dark .session-unbind-btn {
  color: #f87171;
  border-color: color-mix(in srgb, #f87171 45%, transparent);
}

html.dark .session-unbind-btn:hover {
  background: color-mix(in srgb, #f87171 16%, transparent);
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

@media (max-width: 720px) {
  .session-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .session-unbind-btn {
    align-self: flex-end;
  }
}
</style>
