<template>
  <BaseModal
    :open="open"
    :title="t('components.main.form.confirmCopyTitle')"
    variant="confirm"
    @close="$emit('close')"
  >
    <div class="confirm-body">
      <p>{{ t('components.main.form.confirmCopyMessage', { name: cardName }) }}</p>
      <div class="target-list" role="radiogroup" :aria-label="t('components.main.copyToPlatform')">
        <label v-for="tab in targetTabs" :key="tab.id" class="target-option">
          <input v-model="selectedTab" type="radio" name="copy-target-platform" :value="tab.id" />
          <span>{{ tab.label }}</span>
        </label>
      </div>
    </div>
    <footer class="form-actions confirm-actions">
      <BaseButton variant="outline" type="button" @click="$emit('close')">
        {{ t('components.main.form.actions.cancel') }}
      </BaseButton>
      <BaseButton variant="primary" type="button" @click="$emit('confirm', selectedTab)">
        {{ t('components.main.form.actions.confirm') }}
      </BaseButton>
    </footer>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { providerTabs, type ProviderTab } from '../../constants/platforms'
import BaseButton from '../common/BaseButton.vue'
import BaseModal from '../common/BaseModal.vue'

const props = defineProps<{
  cardName: string
  open: boolean
  sourceTab: ProviderTab
}>()

defineEmits<{
  close: []
  confirm: [targetTab: ProviderTab]
}>()

const { t } = useI18n()

// 仅列出源平台之外的目标，避免自我复制
const targetTabs = computed(() => providerTabs.filter((tab) => tab.id !== props.sourceTab))
const selectedTab = ref<ProviderTab>(providerTabs[0].id)

watch(
  [() => props.open, targetTabs],
  ([open, tabs]) => {
    if (!open) {
      return
    }
    if (!tabs.some((tab) => tab.id === selectedTab.value)) {
      selectedTab.value = tabs[0]?.id ?? 'claude'
    }
  },
  { immediate: true },
)
</script>

<style scoped>
.target-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 14px;
}

.target-option {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border: 1px solid var(--mac-border);
  border-radius: 10px;
  cursor: pointer;
  transition: border-color 0.15s ease;
}

.target-option:hover {
  border-color: #0a84ff;
}

.target-option:has(input:checked) {
  border-color: #0a84ff;
}
</style>
