<template>
  <div v-if="loading" class="sessions-empty">
    {{ t('components.sessions.loading') }}
  </div>
  <div v-else-if="groups.length === 0" class="sessions-empty">
    {{ t('components.sessions.noSessions') }}
  </div>
  <div v-else class="sessions-groups">
    <article v-for="group in groups" :key="group.providerName" class="sessions-group-card">
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
          <button class="session-unbind-btn" @click="$emit('unbind', session)">
            {{ t('components.sessions.unbind') }}
          </button>
        </div>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { SessionBinding } from '../../services/sessions'
import type { SessionProviderGroup } from './usePlatformSessions'

interface Props {
  formatRelativeTime: (value: unknown) => string
  groups: SessionProviderGroup[]
  loading: boolean
  truncateSessionId: (sessionId: string) => string
}

interface Emits {
  (e: 'unbind', session: SessionBinding): void
}

defineProps<Props>()
defineEmits<Emits>()

const { t } = useI18n()
</script>

<style scoped>
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
