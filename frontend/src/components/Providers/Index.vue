<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { providerTabs, type ProviderTab } from '../../constants/platforms'
import { useProviderDirectory, type DirectoryEntry } from './useProviderDirectory'
import ProviderEditorModal from './ProviderEditorModal.vue'
import ProviderAdvancedModal from './ProviderAdvancedModal.vue'
import ProviderDeleteModal from './ProviderDeleteModal.vue'
import ProviderCopyModal from './ProviderCopyModal.vue'
import { getErrorMessage } from '../../utils/errors'
import { showToast } from '../../utils/toast'

const { t } = useI18n()
const route = useRoute()
const directory = useProviderDirectory(t)
const {
  filter,
  filteredEntries,
  archivedEntries,
  selectedEntry,
  selectedKey,
  entryKey,
  isDefault,
  editorState,
  advancedState,
  deleteState,
  copyState,
  probeResults,
  probing,
} = directory

const filterTabs = computed(() => [
  { id: 'all' as const, label: t('providersPage.filters.all') },
  ...providerTabs.map((tab) => ({ id: tab.id, label: tab.label })),
])

const totalCount = computed(() =>
  filteredEntries.value.length,
)

onMounted(async () => {
  await directory.loadAll()
  const rawPlatform = typeof route.query.platform === 'string' ? route.query.platform : ''
  if ((providerTabs.map((tab) => tab.id) as string[]).includes(rawPlatform)) {
    filter.value = rawPlatform as ProviderTab
  }
})

const hostOf = (apiUrl: string) => {
  try {
    return new URL(apiUrl).host
  } catch {
    return apiUrl
  }
}

const modelOf = (entry: DirectoryEntry) => {
  const models = Object.keys(entry.provider.supportedModels ?? {}).filter(
    (key) => entry.provider.supportedModels?.[key] !== false,
  )
  return models[0] ?? ''
}

const entrySubtitle = (entry: DirectoryEntry) =>
  [entry.platformLabel, modelOf(entry) || hostOf(entry.provider.apiUrl)].filter(Boolean).join(' · ')

const statusOf = (entry: DirectoryEntry) => {
  if (isDefault(entry)) return { tone: 'success', label: t('providersPage.status.active') }
  const probe = probeResults[entryKey(entry)]
  if (probe?.success) return { tone: 'success', label: t('providersPage.status.verified') }
  if (probe && !probe.success) return { tone: 'danger', label: t('providersPage.status.failed') }
  return { tone: 'warning', label: t('providersPage.status.unverified') }
}

const probeSummary = computed(() => {
  const entry = selectedEntry.value
  if (!entry) return ''
  const probe = probeResults[entryKey(entry)]
  if (!probe) return t('providersPage.detail.neverTested')
  const time = new Date(probe.at)
  const hhmm = `${String(time.getHours()).padStart(2, '0')}:${String(time.getMinutes()).padStart(2, '0')}`
  if (probe.success) {
    return t('providersPage.detail.testOk', { ms: probe.durationMs, time: hhmm })
  }
  return t('providersPage.detail.testErr', { error: probe.error ?? '', time: hhmm })
})

// ---- 行内 ⋯ 菜单 ----
const menuFor = ref('')
const toggleMenu = (key: string) => {
  menuFor.value = menuFor.value === key ? '' : key
}
const closeMenu = () => {
  menuFor.value = ''
}
onMounted(() => window.addEventListener('click', closeMenu))
onUnmounted(() => window.removeEventListener('click', closeMenu))

const copyApiKey = async (value: string) => {
  try {
    await navigator.clipboard.writeText(value)
    showToast(t('providersPage.messages.copied'))
  } catch (error) {
    showToast(getErrorMessage(error, t('providersPage.messages.copyFailed')), 'error')
  }
}

const copySourceEntry = computed(() =>
  copyState.entryKey
    ? filteredEntries.value.find((e) => entryKey(e) === copyState.entryKey) ??
      archivedEntries.value.find((e) => entryKey(e) === copyState.entryKey) ??
      null
    : null,
)

const deleteTarget = computed(() => {
  if (!deleteState.entryKey) return null
  return (
    [...filteredEntries.value, ...archivedEntries.value].find(
      (e) => entryKey(e) === deleteState.entryKey,
    ) ?? null
  )
})
</script>

<template>
  <div class="cs-page" @click="closeMenu">
    <div class="cs-page-head">
      <div>
        <h1 class="cs-page-title">{{ t('providersPage.title') }}</h1>
        <p class="cs-page-sub">{{ t('providersPage.subtitle') }}</p>
      </div>
      <div class="cs-page-actions">
        <button
          type="button"
          class="cs-btn"
          :disabled="!selectedEntry"
          @click="selectedEntry && directory.requestCopy(selectedEntry)"
        >
          ⇄&nbsp;&nbsp;{{ t('providersPage.actions.copyAcross') }}
        </button>
        <button type="button" class="cs-btn cs-btn--primary" @click="directory.openCreate()">
          ＋&nbsp;{{ t('providersPage.actions.create') }}
        </button>
      </div>
    </div>

    <div class="cs-filterbar">
      <button
        v-for="tab in filterTabs"
        :key="tab.id"
        type="button"
        class="cs-tab"
        :class="{ active: filter === tab.id }"
        @click="filter = tab.id"
      >
        {{ tab.label }}
      </button>
      <span class="cs-filter-count">{{ t('providersPage.filters.count', { count: totalCount }) }}</span>
    </div>

    <div class="prov-grid">
      <div class="cs-card cs-card-pad prov-list">
        <h2 class="cs-card-title">{{ t('providersPage.list.title') }}</h2>
        <p class="cs-section-sub">{{ t('providersPage.list.subtitle') }}</p>

        <div v-if="filteredEntries.length === 0" class="cs-empty">
          {{ t('providersPage.list.empty') }}
        </div>

        <button
          v-for="entry in filteredEntries"
          :key="entryKey(entry)"
          type="button"
          class="prov-row"
          :class="{ selected: selectedEntry && entryKey(selectedEntry) === entryKey(entry) }"
          @click.stop="selectedKey = entryKey(entry)"
        >
          <span class="prov-avatar cs-mono">{{ entry.provider.name.slice(0, 1).toUpperCase() || '?' }}</span>
          <span class="prov-main">
            <span class="prov-name">{{ entry.provider.name }}</span>
            <span class="prov-sub cs-mono">{{ entrySubtitle(entry) }}</span>
          </span>
          <span class="cs-pill" :class="`cs-pill--${statusOf(entry).tone}`">
            {{ statusOf(entry).label }}
          </span>
          <span class="cs-menu-anchor" @click.stop>
            <button type="button" class="prov-kebab" @click="toggleMenu(entryKey(entry))">⋯</button>
            <div v-if="menuFor === entryKey(entry)" class="cs-menu">
              <button type="button" class="cs-menu-item" @click="directory.openEdit(entry); closeMenu()">
                {{ t('providersPage.menu.edit') }}
              </button>
              <button type="button" class="cs-menu-item" @click="directory.openAdvanced(entry); closeMenu()">
                {{ t('providersPage.menu.advanced') }}
              </button>
              <button type="button" class="cs-menu-item" @click="directory.requestCopy(entry); closeMenu()">
                {{ t('providersPage.menu.copyAcross') }}
              </button>
              <button type="button" class="cs-menu-item" @click="directory.toggleArchive(entry); closeMenu()">
                {{ t('providersPage.menu.archive') }}
              </button>
              <button type="button" class="cs-menu-item cs-menu-item--danger" @click="directory.requestDelete(entry); closeMenu()">
                {{ t('providersPage.menu.delete') }}
              </button>
            </div>
          </span>
        </button>
      </div>

      <div class="cs-card cs-card-pad prov-detail" v-if="selectedEntry">
        <h2 class="cs-card-title">{{ t('providersPage.detail.title') }}</h2>
        <p class="cs-section-sub">{{ selectedEntry.platformLabel }}</p>

        <p class="prov-detail-name">{{ selectedEntry.provider.name }}</p>
        <span class="cs-pill" :class="`cs-pill--${statusOf(selectedEntry).tone}`">
          {{ statusOf(selectedEntry).label }}
        </span>

        <p class="prov-detail-label">{{ t('providersPage.detail.baseUrl') }}</p>
        <p class="prov-detail-value cs-mono">{{ selectedEntry.provider.apiUrl }}</p>

        <hr class="cs-divider" />

        <p class="prov-detail-label">{{ t('providersPage.detail.apiKey') }}</p>
        <div class="prov-key-row">
          <span class="prov-key-value cs-mono">{{ selectedEntry.provider.apiKey || '—' }}</span>
          <button
            type="button"
            class="cs-btn"
            :disabled="!selectedEntry.provider.apiKey"
            @click="copyApiKey(selectedEntry.provider.apiKey)"
          >
            {{ t('providersPage.detail.copy') }}
          </button>
        </div>

        <hr class="cs-divider" />

        <p class="prov-detail-label">{{ t('providersPage.detail.lastTest') }}</p>
        <p class="prov-detail-probe">{{ probeSummary }}</p>

        <div class="prov-detail-actions">
          <button type="button" class="cs-btn" @click="directory.openEdit(selectedEntry)">
            {{ t('providersPage.actions.edit') }}
          </button>
          <button
            type="button"
            class="cs-btn"
            :disabled="probing === entryKey(selectedEntry)"
            @click="directory.testConnection(selectedEntry)"
          >
            {{ probing === entryKey(selectedEntry) ? t('providersPage.actions.testing') : t('providersPage.actions.test') }}
          </button>
          <button
            type="button"
            class="cs-btn cs-btn--primary"
            :disabled="isDefault(selectedEntry)"
            @click="directory.setDefault(selectedEntry)"
          >
            {{ isDefault(selectedEntry) ? t('providersPage.actions.current') : t('providersPage.actions.setCurrent') }}
          </button>
        </div>

        <p class="prov-detail-note">{{ t('providersPage.detail.privacyNote') }}</p>
      </div>
    </div>

    <div v-if="archivedEntries.length > 0" class="cs-card cs-card-pad">
      <h2 class="cs-card-title">{{ t('providersPage.archived.title', { count: archivedEntries.length }) }}</h2>
      <div
        v-for="entry in archivedEntries"
        :key="entryKey(entry)"
        class="prov-archived-row"
      >
        <span class="prov-name">{{ entry.provider.name }}</span>
        <span class="prov-sub">{{ entry.platformLabel }}</span>
        <span class="prov-archived-actions">
          <button type="button" class="cs-btn" @click="directory.toggleArchive(entry)">
            {{ t('providersPage.menu.restore') }}
          </button>
          <button type="button" class="cs-btn" @click="directory.requestDelete(entry)">
            {{ t('providersPage.menu.delete') }}
          </button>
        </span>
      </div>
    </div>

    <ProviderEditorModal
      :state="editorState"
      @close="editorState.open = false"
      @submit="directory.submitEditor"
    />
    <ProviderAdvancedModal
      :state="advancedState"
      @close="advancedState.open = false"
      @submit="directory.submitAdvanced"
    />
    <ProviderDeleteModal
      :open="deleteState.open"
      :name="deleteTarget?.provider.name ?? ''"
      :busy="deleteState.busy"
      @close="deleteState.open = false"
      @confirm="directory.confirmDelete"
    />
    <ProviderCopyModal
      :open="copyState.open"
      :source="copySourceEntry"
      @close="copyState.open = false"
      @confirm="directory.confirmCopy"
    />
  </div>
</template>

<style scoped>
.prov-grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(320px, 1fr);
  gap: 18px;
  align-items: start;
}

.prov-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.prov-list .cs-section-sub {
  margin-bottom: 8px;
}

.prov-row {
  display: flex;
  align-items: center;
  gap: 13px;
  min-height: 78px;
  padding: 12px 11px;
  border-radius: var(--cs-r-row);
  border: 1px solid var(--cs-border);
  background: var(--cs-surface);
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.12s ease, background 0.12s ease;
}

.prov-row.selected {
  background: var(--cs-brand-tint);
  border-color: var(--cs-brand-tint-border);
}

.prov-avatar {
  width: 43px;
  height: 43px;
  border-radius: 10px;
  background: var(--cs-field);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 17px;
  font-weight: 700;
  color: var(--cs-brand);
  flex-shrink: 0;
}

.prov-row.selected .prov-avatar {
  background: var(--cs-brand-tint);
}

.prov-main {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.prov-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--cs-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.prov-sub {
  font-size: 10px;
  color: var(--cs-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.prov-kebab {
  border: none;
  background: transparent;
  color: var(--cs-text-secondary);
  font-size: 18px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
}

.prov-kebab:hover {
  background: var(--cs-row-hover);
}

.prov-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
  position: sticky;
  top: 0;
}

.prov-detail-name {
  margin: 10px 0 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--cs-text);
}

.prov-detail .cs-pill {
  align-self: flex-start;
}

.prov-detail-label {
  margin: 6px 0 0;
  font-size: 10px;
  color: var(--cs-text-secondary);
}

.prov-detail-value {
  margin: 0;
  font-size: 11px;
  color: var(--cs-text);
  word-break: break-all;
  user-select: text;
}

.prov-key-row {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--cs-field);
  border: 1px solid var(--cs-border);
  border-radius: var(--cs-r-field);
  padding: 2px 2px 2px 10px;
}

.prov-key-value {
  flex: 1;
  font-size: 10px;
  color: var(--cs-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  user-select: text;
}

.prov-detail-probe {
  margin: 0;
  font-size: 12px;
  color: var(--cs-text);
}

.prov-detail-actions {
  display: flex;
  gap: 9px;
  margin-top: 6px;
  flex-wrap: wrap;
}

.prov-detail-note {
  margin: 4px 0 0;
  font-size: 10px;
  color: var(--cs-text-secondary);
}

.prov-archived-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 4px;
  border-top: 1px solid var(--cs-border);
}

.prov-archived-row:first-of-type {
  border-top: none;
}

.prov-archived-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

@media (max-width: 1100px) {
  .prov-grid {
    grid-template-columns: 1fr;
  }

  .prov-detail {
    position: static;
  }
}
</style>
