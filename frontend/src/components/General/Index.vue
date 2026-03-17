<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ListItem from '../Setting/ListRow.vue'
import LanguageSwitcher from '../Setting/LanguageSwitcher.vue'
import ThemeSetting from '../Setting/ThemeSetting.vue'
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
  updateSettings,
} = useAppSettingsStore()

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
    </div>
  </div>
</template>
