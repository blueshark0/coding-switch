<template>
  <div class="logs-page">
    <div class="logs-header">
      <BaseButton variant="outline" type="button" @click="backToHome">
        {{ t('components.logs.back') }}
      </BaseButton>
      <div class="tab-group" role="tablist" :aria-label="t('components.logs.tabs.ariaLabel')">
        <button
          class="tab-pill"
          :class="{ active: activeTab === 'requests' }"
          role="tab"
          :aria-selected="activeTab === 'requests'"
          type="button"
          @click="activeTab = 'requests'"
        >
          {{ t('components.logs.tabs.requests') }}
        </button>
        <button
          class="tab-pill"
          :class="{ active: activeTab === 'app' }"
          role="tab"
          :aria-selected="activeTab === 'app'"
          type="button"
          @click="activeTab = 'app'"
        >
          {{ t('components.logs.tabs.app') }}
        </button>
      </div>
    </div>

    <!-- v-if 挂载/卸载面板，保证各自的轮询定时器随切换停止 -->
    <RequestLogsPanel v-if="activeTab === 'requests'" />
    <AppLogViewer v-else />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BaseButton from '../common/BaseButton.vue'
import RequestLogsPanel from './RequestLogsPanel.vue'
import AppLogViewer from './AppLogViewer.vue'

const { t } = useI18n()
const router = useRouter()

const activeTab = ref<'requests' | 'app'>('requests')

const backToHome = () => {
  router.push('/')
}
</script>
