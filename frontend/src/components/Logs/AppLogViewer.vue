<template>
  <div class="app-log-panel">
    <div class="app-log-toolbar">
      <label class="app-log-field">
        <span>{{ t('components.logs.appLog.levelLabel') }}</span>
        <select v-model="level" class="mac-select">
          <option value="">{{ t('components.logs.appLog.levels.all') }}</option>
          <option value="INFO">INFO</option>
          <option value="WARN">WARN</option>
          <option value="ERROR">ERROR</option>
          <option value="none">{{ t('components.logs.appLog.levels.none') }}</option>
        </select>
      </label>
      <input
        v-model="keyword"
        type="search"
        class="app-log-search"
        :placeholder="t('components.logs.appLog.keywordPlaceholder')"
      />
      <label class="app-log-follow">
        <span>{{ t('components.logs.appLog.follow') }}</span>
        <label class="mac-switch sm">
          <input v-model="following" type="checkbox" />
          <span></span>
        </label>
      </label>
      <BaseButton size="sm" :disabled="loading" @click="manualRefresh">
        {{ t('components.logs.appLog.refresh') }}
      </BaseButton>
    </div>

    <p v-if="truncated" class="app-log-hint">{{ t('components.logs.appLog.truncatedHint') }}</p>

    <div ref="listEl" class="app-log-list">
      <div v-for="(entry, index) in entries" :key="index" class="app-log-line">
        <template v-if="entry.time">
          <span class="app-log-time">{{ entry.time }}</span>
          <span class="app-log-source">{{ entry.source }}</span>
          <span v-if="entry.level" :class="['app-log-level', levelClass(entry.level)]">
            {{ entry.level }}
          </span>
        </template>
        <span class="app-log-message">{{ entry.message }}</span>
      </div>
      <p v-if="!entries.length && !loading" class="app-log-empty">
        {{ t('components.logs.appLog.empty') }}
      </p>
      <p v-if="!entries.length && loading" class="app-log-empty">
        {{ t('components.logs.loading') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseButton from '../common/BaseButton.vue'
import { useAppLog } from './useAppLog'

const { t } = useI18n()
const { entries, following, keyword, level, loading, manualRefresh, truncated } = useAppLog()

const listEl = ref<HTMLElement | null>(null)

const levelClass = (value: string) => {
  switch (value) {
    case 'ERROR':
      return 'is-error'
    case 'WARN':
      return 'is-warn'
    case 'DEBUG':
      return 'is-debug'
    default:
      return 'is-info'
  }
}

const nearBottom = () => {
  const el = listEl.value
  if (!el) {
    return true
  }
  return el.scrollHeight - el.scrollTop - el.clientHeight < 40
}

// 新内容到达时仅在用户本就位于底部附近（或列表刚重建）时吸底，避免打断上翻浏览。
watch(
  () => entries.value,
  async (next, prev) => {
    const stick = nearBottom() || (prev?.length ?? 0) === 0
    if (!stick) {
      return
    }
    await nextTick()
    if (listEl.value) {
      listEl.value.scrollTop = listEl.value.scrollHeight
    }
  },
)
</script>

<style scoped>
.app-log-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.app-log-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.app-log-field {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: var(--mac-text-secondary);
}

.app-log-field span {
  white-space: nowrap;
}

.app-log-field select {
  font-size: 0.85rem;
  padding: 6px 8px;
}

.app-log-search {
  flex: 1 1 220px;
  min-width: 160px;
  padding: 6px 10px;
  font-size: 0.85rem;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.4);
  background: transparent;
  color: var(--mac-text);
  outline: none;
}

.app-log-search:focus {
  border-color: rgba(56, 189, 248, 0.7);
}

.app-log-follow {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: var(--mac-text-secondary);
  white-space: nowrap;
}

.app-log-hint {
  margin: 0;
  font-size: 0.8rem;
  color: var(--mac-text-secondary);
}

.app-log-list {
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  padding: 12px 0;
  height: 60vh;
  overflow-y: auto;
  font-family: 'SFMono-Regular', Menlo, Consolas, monospace;
  font-size: 0.8rem;
  line-height: 1.55;
}

.app-log-line {
  display: block;
  padding: 1px 16px;
  white-space: pre-wrap;
  word-break: break-word;
}

.app-log-line:hover {
  background: rgba(148, 163, 184, 0.1);
}

.app-log-time {
  color: var(--mac-text-secondary);
  margin-right: 8px;
}

.app-log-source {
  color: #38bdf8;
  margin-right: 8px;
}

.app-log-level {
  display: inline-block;
  padding: 0 6px;
  margin-right: 8px;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  vertical-align: middle;
}

.app-log-level.is-warn {
  background: rgba(250, 204, 21, 0.25);
  color: #b45309;
}

.app-log-level.is-error {
  background: rgba(248, 113, 113, 0.25);
  color: #dc2626;
}

.app-log-level.is-info {
  background: rgba(148, 163, 184, 0.25);
  color: #475569;
}

.app-log-level.is-debug {
  background: rgba(56, 189, 248, 0.2);
  color: #0369a1;
}

.app-log-message {
  color: var(--mac-text);
}

.app-log-empty {
  text-align: center;
  padding: 20px;
  color: var(--mac-text-secondary);
}

html.dark .app-log-level.is-warn {
  color: #fcd34d;
}

html.dark .app-log-level.is-error {
  color: #fca5a5;
}

html.dark .app-log-level.is-info {
  color: #cbd5f5;
}

html.dark .app-log-level.is-debug {
  color: #7dd3fc;
}
</style>
