<script setup lang="ts">
import { ref, watch } from 'vue'
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ListItem from '../Setting/ListRow.vue'
import LanguageSwitcher from '../Setting/LanguageSwitcher.vue'
import ThemeSetting from '../Setting/ThemeSetting.vue'
import WebDavSyncSection from './WebDavSyncSection.vue'
import { useAppSettingsStore } from '../../composables/useAppSettingsStore'
import { getErrorMessage } from '../../utils/errors'
import { showToast } from '../../utils/toast'

const router = useRouter()
const { t } = useI18n()
const {
  loadSettings,
  loading: settingsLoading,
  saving: saveBusy,
  showHeatmap: heatmapEnabled,
  showHomeTitle: homeTitleVisible,
  proxyEnabled,
  proxyURL: proxyUrlValue,
  updateSettings,
} = useAppSettingsStore()

const proxyInput = ref('')
const proxyError = ref('')

watch(proxyUrlValue, (val) => {
  proxyInput.value = val
}, { immediate: true })

const goBack = () => {
  router.push('/')
}

const updateVisibilitySetting = async (
  key: 'show_heatmap' | 'show_home_title',
  value: boolean,
) => {
  if (settingsLoading.value || saveBusy.value) return
  try {
    await updateSettings({ [key]: value })
  } catch (error) {
    console.error('failed to save app settings', error)
    showToast(getErrorMessage(error, t('components.general.messages.saveFailed')), 'error')
  }
}

const onHeatmapChange = (event: Event) => {
  const target = event.target as HTMLInputElement | null
  void updateVisibilitySetting('show_heatmap', Boolean(target?.checked))
}

const onHomeTitleChange = (event: Event) => {
  const target = event.target as HTMLInputElement | null
  void updateVisibilitySetting('show_home_title', Boolean(target?.checked))
}

const onProxyToggle = async (event: Event) => {
  const target = event.target as HTMLInputElement | null
  if (settingsLoading.value || saveBusy.value) return
  try {
    await updateSettings({ proxy_enabled: Boolean(target?.checked) })
  } catch (error) {
    console.error('failed to save proxy settings', error)
    showToast(getErrorMessage(error, t('components.general.messages.saveFailed')), 'error')
  }
}

const PROXY_PREFIXES = ['http://', 'https://', 'socks5://', 'socks5h://']

const onProxySave = async () => {
  const raw = proxyInput.value.trim()
  if (raw && !PROXY_PREFIXES.some(p => raw.startsWith(p))) {
    proxyError.value = 'URL must start with http://, https://, socks5://, or socks5h://'
    return
  }
  proxyError.value = ''
  if (settingsLoading.value || saveBusy.value) return
  try {
    await updateSettings({ proxy_url: raw })
  } catch (error) {
    console.error('failed to save proxy URL', error)
    showToast(getErrorMessage(error, t('components.general.messages.saveFailed')), 'error')
  }
}

const onProxyClear = async () => {
  proxyInput.value = ''
  proxyError.value = ''
  if (settingsLoading.value || saveBusy.value) return
  try {
    await updateSettings({ proxy_url: '' })
  } catch (error) {
    console.error('failed to clear proxy URL', error)
    showToast(getErrorMessage(error, t('components.general.messages.saveFailed')), 'error')
  }
}

const reloadApplicationSettings = () => {
  void loadSettings(true).catch((error) => {
    console.error('failed to reload app settings', error)
    showToast(getErrorMessage(error, t('components.general.messages.loadFailed')), 'error')
  })
}

onMounted(() => {
  void loadSettings().catch((error) => {
    console.error('failed to load app settings', error)
    showToast(getErrorMessage(error, t('components.general.messages.loadFailed')), 'error')
  })
})
</script>

<template>
  <div class="main-shell general-shell">
    <div class="global-actions">
      <p class="global-eyebrow">{{ $t('components.general.title.application') }}</p>
      <button class="ghost-icon" :aria-label="$t('components.general.buttons.back')" @click="goBack">
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
    </div>

    <div class="general-page">
      <section>
        <h2 class="mac-section-title">{{ $t('components.general.title.application') }}</h2>
        <div class="mac-panel">
          <ListItem :label="$t('components.general.label.heatmap')">
            <label class="mac-switch">
              <input
                type="checkbox"
                :disabled="settingsLoading || saveBusy"
                :checked="heatmapEnabled"
                @change="onHeatmapChange"
              />
              <span></span>
            </label>
          </ListItem>
          <ListItem :label="$t('components.general.label.homeTitle')">
            <label class="mac-switch">
              <input
                type="checkbox"
                :disabled="settingsLoading || saveBusy"
                :checked="homeTitleVisible"
                @change="onHomeTitleChange"
              />
              <span></span>
            </label>
          </ListItem>
        </div>
      </section>

      <section>
        <h2 class="mac-section-title">{{ $t('components.general.title.network') }}</h2>
        <div class="mac-panel">
          <ListItem :label="$t('components.general.label.proxy')">
            <label class="mac-switch">
              <input
                type="checkbox"
                :disabled="settingsLoading || saveBusy"
                :checked="proxyEnabled"
                @change="onProxyToggle"
              />
              <span></span>
            </label>
          </ListItem>
          <div v-if="proxyEnabled" class="proxy-url-row">
            <div class="proxy-input-group">
              <input
                type="text"
                class="mac-input proxy-input"
                :placeholder="$t('components.general.placeholder.proxy')"
                :disabled="settingsLoading || saveBusy"
                v-model="proxyInput"
                @keydown.enter="onProxySave"
              />
              <button
                v-if="proxyInput"
                class="proxy-clear-btn"
                :disabled="settingsLoading || saveBusy"
                @click="onProxyClear"
              >&times;</button>
              <button
                class="proxy-save-btn"
                :disabled="settingsLoading || saveBusy || proxyInput === proxyUrlValue"
                @click="onProxySave"
              >{{ $t('components.general.buttons.save') }}</button>
            </div>
            <div v-if="proxyError" class="proxy-error">{{ proxyError }}</div>
          </div>
        </div>
      </section>

      <section>
        <h2 class="mac-section-title">{{ $t('components.general.title.exterior') }}</h2>
        <div class="mac-panel">
          <ListItem :label="$t('components.general.label.language')">
            <LanguageSwitcher />
          </ListItem>
          <ListItem :label="$t('components.general.label.theme')">
            <ThemeSetting />
          </ListItem>
        </div>
      </section>

      <WebDavSyncSection @restored="reloadApplicationSettings" />
    </div>
  </div>
</template>

<style scoped>
.proxy-url-row {
  padding: 6px 18px 14px;
}

.proxy-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.proxy-input {
  flex: 1;
  padding: 6px 10px;
  font-size: 0.85rem;
  border: 1px solid var(--mac-border, #d1d5db);
  border-radius: 6px;
  background: var(--mac-bg, #fff);
  color: var(--mac-text, #1d1d1f);
  outline: none;
  transition: border-color 0.15s;
}

.proxy-input:focus {
  border-color: var(--mac-accent, #007aff);
}

.proxy-input:disabled {
  opacity: 0.5;
}

.proxy-clear-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 50%;
  background: var(--mac-border, #d1d5db);
  color: var(--mac-text-secondary, #86868b);
  font-size: 0.9rem;
  cursor: pointer;
  line-height: 1;
  flex-shrink: 0;
}

.proxy-clear-btn:hover {
  background: var(--mac-text-secondary, #86868b);
  color: #fff;
}

.proxy-clear-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.proxy-save-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 5px 14px;
  min-height: 28px;
  font-size: 0.8rem;
  font-weight: 600;
  line-height: 1;
  border: none;
  border-radius: 6px;
  background: var(--mac-accent, #007aff);
  color: #fff;
  cursor: pointer;
  flex-shrink: 0;
}

.proxy-save-btn:hover:not(:disabled) {
  opacity: 0.85;
}

.proxy-save-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.proxy-error {
  margin-top: 4px;
  font-size: 0.78rem;
  color: #ff3b30;
}
</style>
