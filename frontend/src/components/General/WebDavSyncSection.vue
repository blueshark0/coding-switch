<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseButton from '../common/BaseButton.vue'
import {
  applySnapshot,
  createLocalRollback,
  fetchRemoteSnapshotMeta,
  fetchSyncSettings,
  pullSnapshot,
  pushSnapshot,
  saveSyncSettings,
  testConnection,
  type RemoteSnapshotMeta,
  type SyncSettings,
} from '../../services/configSync'
import { getErrorMessage } from '../../utils/errors'
import { showToast } from '../../utils/toast'
import { getCurrentTheme, setTheme, type ThemeMode } from '../../utils/ThemeManager'
import { i18n, setupI18n } from '../../utils/i18n'
import type { Locale } from '../../locales'

const emit = defineEmits<{
  restored: []
}>()

const { t } = useI18n()

const DEFAULT_SETTINGS: SyncSettings = {
  endpoint: '',
  username: '',
  appPassword: '',
  remotePath: '/CodeSwitch',
  syncPassphrase: '',
}

const form = reactive<SyncSettings>({ ...DEFAULT_SETTINGS })
const savedSnapshot = ref('')
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const inspecting = ref(false)
const uploading = ref(false)
const restoring = ref(false)
const showPassword = ref(false)
const showPassphrase = ref(false)
const remoteMeta = ref<RemoteSnapshotMeta | null>(null)

const busy = computed(
  () => loading.value || saving.value || testing.value || inspecting.value || uploading.value || restoring.value,
)

const currentSnapshot = () =>
  JSON.stringify({
    endpoint: form.endpoint.trim().replace(/\/+$/, ''),
    username: form.username.trim(),
    appPassword: form.appPassword,
    remotePath: form.remotePath.trim() || '/CodeSwitch',
    syncPassphrase: form.syncPassphrase,
  })

const dirty = computed(() => currentSnapshot() !== savedSnapshot.value)
const hasConnectionFields = computed(
  () => form.endpoint.trim() !== '' && form.username.trim() !== '' && form.appPassword.trim() !== '',
)
const hasAllFields = computed(() => hasConnectionFields.value && form.syncPassphrase.trim() !== '')
const canUseRemote = computed(() => !busy.value && !dirty.value && hasConnectionFields.value)
const canSync = computed(() => !busy.value && !dirty.value && hasAllFields.value)

const remoteSummary = computed(() => {
  if (!remoteMeta.value?.exists) {
    return t('components.general.sync.meta.empty')
  }
  return t('components.general.sync.meta.summary', {
    updatedAt: formatTime(remoteMeta.value.updatedAt),
    deviceName: remoteMeta.value.deviceName || t('components.general.sync.meta.unknown'),
    appVersion: remoteMeta.value.appVersion || t('components.general.sync.meta.unknown'),
  })
})

const applyForm = (settings: SyncSettings) => {
  form.endpoint = settings.endpoint ?? ''
  form.username = settings.username ?? ''
  form.appPassword = settings.appPassword ?? ''
  form.remotePath = settings.remotePath ?? '/CodeSwitch'
  form.syncPassphrase = settings.syncPassphrase ?? ''
  savedSnapshot.value = currentSnapshot()
}

const loadSyncSettings = async () => {
  loading.value = true
  try {
    applyForm(await fetchSyncSettings())
  } catch (error) {
    showToast(getErrorMessage(error, t('components.general.sync.messages.loadFailed')), 'error')
  } finally {
    loading.value = false
  }
}

const persistSettings = async () => {
  saving.value = true
  try {
    applyForm(
      await saveSyncSettings({
        endpoint: form.endpoint,
        username: form.username,
        appPassword: form.appPassword,
        remotePath: form.remotePath,
        syncPassphrase: form.syncPassphrase,
      }),
    )
    showToast(t('components.general.sync.messages.saveSuccess'))
  } catch (error) {
    showToast(getErrorMessage(error, t('components.general.sync.messages.saveFailed')), 'error')
  } finally {
    saving.value = false
  }
}

const ensureSaved = () => {
  if (!dirty.value) return true
  showToast(t('components.general.sync.messages.saveBeforeAction'), 'error')
  return false
}

const checkConnection = async () => {
  if (!ensureSaved()) return
  testing.value = true
  try {
    await testConnection()
    showToast(t('components.general.sync.messages.connectionSuccess'))
  } catch (error) {
    showToast(getErrorMessage(error, t('components.general.sync.messages.connectionFailed')), 'error')
  } finally {
    testing.value = false
  }
}

const inspectRemoteSnapshot = async () => {
  if (!ensureSaved()) return
  inspecting.value = true
  try {
    remoteMeta.value = await fetchRemoteSnapshotMeta()
    if (!remoteMeta.value.exists) {
      showToast(t('components.general.sync.messages.remoteEmpty'))
    }
  } catch (error) {
    showToast(getErrorMessage(error, t('components.general.sync.messages.inspectFailed')), 'error')
  } finally {
    inspecting.value = false
  }
}

const uploadSnapshot = async () => {
  if (!ensureSaved()) return
  uploading.value = true
  try {
    remoteMeta.value = await pushSnapshot(currentUIPreferences())
    showToast(t('components.general.sync.messages.uploadSuccess'))
  } catch (error) {
    showToast(getErrorMessage(error, t('components.general.sync.messages.uploadFailed')), 'error')
  } finally {
    uploading.value = false
  }
}

const restoreSnapshot = async () => {
  if (!ensureSaved()) return
  if (!window.confirm(t('components.general.sync.messages.restoreConfirm'))) {
    return
  }

  restoring.value = true
  let rollbackPath = ''
  try {
    rollbackPath = await createLocalRollback(currentUIPreferences())
    const snapshot = await pullSnapshot()
    await applySnapshot(snapshot.backend)
    setTheme(normalizeTheme(snapshot.ui.theme))
    await setupI18n(normalizeLocale(snapshot.ui.locale))
    emit('restored')
    remoteMeta.value = await fetchRemoteSnapshotMeta()
    showToast(
      t('components.general.sync.messages.restoreSuccess', {
        rollback: basename(rollbackPath),
      }),
    )
  } catch (error) {
    const fallback = rollbackPath
      ? t('components.general.sync.messages.restoreFailedWithRollback', {
          rollback: basename(rollbackPath),
        })
      : t('components.general.sync.messages.restoreFailed')
    showToast(getErrorMessage(error, fallback), 'error')
  } finally {
    restoring.value = false
  }
}

const currentUIPreferences = () => ({
  theme: normalizeTheme(getCurrentTheme()),
  locale: normalizeLocale(i18n.global.locale.value as string),
})

const normalizeTheme = (value: string): ThemeMode => {
  if (value === 'light' || value === 'dark' || value === 'systemdefault') {
    return value
  }
  return 'systemdefault'
}

const normalizeLocale = (value: string): Locale => (value === 'en' ? 'en' : 'zh')

const basename = (value: string) => value.split(/[\\/]/).pop() || value

const formatTime = (value?: string) => {
  if (!value) return t('components.general.sync.meta.unknown')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

onMounted(() => {
  void loadSyncSettings()
})
</script>

<template>
  <section>
    <h2 class="mac-section-title">{{ t('components.general.title.sync') }}</h2>
    <div class="mac-panel sync-panel">
      <p class="sync-hint">{{ t('components.general.sync.hint') }}</p>

      <div class="sync-grid">
        <label class="sync-field sync-field--full">
          <span>{{ t('components.general.sync.fields.endpoint') }}</span>
          <input
            v-model="form.endpoint"
            class="mac-input"
            :disabled="busy"
            type="text"
            placeholder="https://dav.jianguoyun.com/dav"
          />
        </label>
        <label class="sync-field">
          <span>{{ t('components.general.sync.fields.username') }}</span>
          <input v-model="form.username" class="mac-input" :disabled="busy" type="text" />
        </label>
        <label class="sync-field">
          <span>{{ t('components.general.sync.fields.remotePath') }}</span>
          <input v-model="form.remotePath" class="mac-input" :disabled="busy" type="text" />
        </label>
        <label class="sync-field">
          <span>{{ t('components.general.sync.fields.appPassword') }}</span>
          <div class="sync-secret">
            <input
              v-model="form.appPassword"
              class="mac-input"
              :disabled="busy"
              :type="showPassword ? 'text' : 'password'"
            />
            <BaseButton
              class="sync-secret-toggle"
              type="button"
              variant="outline"
              :disabled="busy"
              @click="showPassword = !showPassword"
            >
              {{ showPassword ? t('components.general.sync.actions.hide') : t('components.general.sync.actions.show') }}
            </BaseButton>
          </div>
        </label>
        <label class="sync-field">
          <span>{{ t('components.general.sync.fields.syncPassphrase') }}</span>
          <div class="sync-secret">
            <input
              v-model="form.syncPassphrase"
              class="mac-input"
              :disabled="busy"
              :type="showPassphrase ? 'text' : 'password'"
            />
            <BaseButton
              class="sync-secret-toggle"
              type="button"
              variant="outline"
              :disabled="busy"
              @click="showPassphrase = !showPassphrase"
            >
              {{ showPassphrase ? t('components.general.sync.actions.hide') : t('components.general.sync.actions.show') }}
            </BaseButton>
          </div>
        </label>
      </div>

      <div class="sync-actions">
        <BaseButton
          class="sync-action-button sync-action-button--primary"
          variant="primary"
          :disabled="busy"
          @click="persistSettings"
        >
          {{ saving ? t('components.general.sync.actions.saving') : t('components.general.sync.actions.save') }}
        </BaseButton>
        <BaseButton class="sync-action-button" variant="outline" :disabled="!canUseRemote" @click="checkConnection">
          {{ testing ? t('components.general.sync.actions.testing') : t('components.general.sync.actions.test') }}
        </BaseButton>
        <BaseButton class="sync-action-button" variant="outline" :disabled="!canUseRemote" @click="inspectRemoteSnapshot">
          {{ inspecting ? t('components.general.sync.actions.loadingMeta') : t('components.general.sync.actions.viewMeta') }}
        </BaseButton>
        <BaseButton class="sync-action-button" variant="outline" :disabled="!canSync" @click="uploadSnapshot">
          {{ uploading ? t('components.general.sync.actions.uploading') : t('components.general.sync.actions.upload') }}
        </BaseButton>
        <BaseButton
          class="sync-action-button sync-action-button--danger"
          variant="danger"
          :disabled="!canSync"
          @click="restoreSnapshot"
        >
          {{ restoring ? t('components.general.sync.actions.restoring') : t('components.general.sync.actions.restore') }}
        </BaseButton>
      </div>

      <div class="sync-meta">
        <span class="sync-meta-label">{{ t('components.general.sync.meta.title') }}</span>
        <span class="sync-meta-value">{{ remoteSummary }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.sync-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sync-hint,
.sync-meta {
  padding: 0 18px;
}

.sync-hint,
.sync-meta-value {
  color: var(--mac-text-secondary);
  font-size: 0.86rem;
}

.sync-hint {
  padding-top: 14px;
}

.sync-grid {
  padding: 0 18px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 18px;
}

.sync-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.sync-field--full {
  grid-column: 1 / -1;
}

.sync-field span,
.sync-meta-label {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--mac-text);
}

.mac-input {
  width: 100%;
  border: 1px solid var(--mac-border);
  border-radius: 12px;
  padding: 10px 14px;
  font: inherit;
  background: var(--mac-surface-strong);
  color: var(--mac-text);
}

.sync-secret {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  align-items: stretch;
}

.sync-secret-toggle {
  min-width: 76px;
  min-height: 42px;
  padding: 0 16px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 95%, transparent);
  background: color-mix(in srgb, var(--mac-surface-strong) 82%, var(--mac-surface) 18%);
  color: var(--mac-text-secondary);
  box-shadow: none;
  font-size: 0.85rem;
  font-weight: 600;
  line-height: 1;
}

.sync-secret-toggle:hover,
.sync-secret-toggle:focus-visible {
  background: color-mix(in srgb, var(--mac-surface-strong) 62%, var(--mac-accent) 8%);
  color: var(--mac-text);
}

.sync-secret-toggle:disabled {
  opacity: 0.58;
  cursor: not-allowed;
}

.sync-actions {
  padding: 0 18px 6px;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.sync-action-button {
  min-width: 112px;
  min-height: 40px;
  padding: 0 18px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 95%, transparent);
  background: color-mix(in srgb, var(--mac-surface-strong) 85%, var(--mac-surface) 15%);
  color: var(--mac-text);
  box-shadow: none;
  font-size: 0.88rem;
  font-weight: 600;
  letter-spacing: 0;
}

.sync-action-button:hover,
.sync-action-button:focus-visible {
  background: color-mix(in srgb, var(--mac-surface-strong) 70%, var(--mac-accent) 9%);
}

.sync-action-button.sync-action-button--primary {
  border-color: color-mix(in srgb, var(--mac-accent) 32%, transparent);
  background: color-mix(in srgb, var(--mac-accent) 92%, white 8%);
  color: #fff;
  box-shadow: 0 10px 18px color-mix(in srgb, var(--mac-accent) 24%, transparent);
}

.sync-action-button.sync-action-button--primary:hover,
.sync-action-button.sync-action-button--primary:focus-visible {
  background: color-mix(in srgb, var(--mac-accent) 96%, white 4%);
}

.sync-action-button.sync-action-button--danger {
  border-color: color-mix(in srgb, #ff3b30 35%, transparent);
  background: color-mix(in srgb, #ff3b30 92%, white 8%);
  color: #fff;
  box-shadow: 0 10px 18px color-mix(in srgb, #ff3b30 20%, transparent);
}

.sync-action-button.sync-action-button--danger:hover,
.sync-action-button.sync-action-button--danger:focus-visible {
  background: color-mix(in srgb, #ff3b30 96%, white 4%);
}

.sync-action-button:disabled {
  opacity: 0.56;
  cursor: not-allowed;
  box-shadow: none;
}

.sync-meta {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  padding-bottom: 18px;
}

@media (max-width: 960px) {
  .sync-grid {
    grid-template-columns: 1fr;
  }

  .sync-secret {
    grid-template-columns: 1fr;
  }

  .sync-secret-toggle,
  .sync-action-button {
    width: 100%;
  }
}
</style>
