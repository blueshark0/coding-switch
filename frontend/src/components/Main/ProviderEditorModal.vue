<template>
  <BaseModal
    :open="open"
    :title="editing ? t('components.main.form.editTitle') : t('components.main.form.createTitle')"
    @close="$emit('close')"
  >
    <form class="vendor-form" @submit.prevent="$emit('submit')">
      <label class="form-field">
        <span class="label-row">
          {{ t('components.main.form.labels.name') }}
          <span v-if="nameError" class="field-error">
            {{ nameError }}
          </span>
        </span>
        <BaseInput
          v-model="form.name"
          type="text"
          :placeholder="t('components.main.form.placeholders.name')"
          required
          :disabled="editing"
          :class="{ 'has-error': !!nameError }"
        />
      </label>

      <label class="form-field">
        <span class="label-row">
          {{ t('components.main.form.labels.apiUrl') }}
          <span v-if="apiUrlError" class="field-error">
            {{ apiUrlError }}
          </span>
        </span>
        <BaseInput
          v-model="form.apiUrl"
          type="text"
          :placeholder="t('components.main.form.placeholders.apiUrl')"
          required
          :class="{ 'has-error': !!apiUrlError }"
        />
      </label>

      <label class="form-field">
        <span>{{ t('components.main.form.labels.officialSite') }}</span>
        <BaseInput
          v-model="form.officialSite"
          type="text"
          :placeholder="t('components.main.form.placeholders.officialSite')"
        />
      </label>

      <label class="form-field">
        <span>{{ t('components.main.form.labels.apiKey') }}</span>
        <BaseInput
          v-model="form.apiKey"
          type="text"
          :placeholder="t('components.main.form.placeholders.apiKey')"
        />
      </label>

      <section class="advanced-section">
        <button
          class="advanced-toggle"
          type="button"
          :aria-expanded="advancedOpen"
          @click="advancedOpen = !advancedOpen"
        >
          <svg
            class="advanced-chevron"
            :class="{ 'is-expanded': advancedOpen }"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              d="M8.25 4.5l7.5 7.5-7.5 7.5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          {{ t('components.main.form.advancedSettings') }}
        </button>

        <div v-show="advancedOpen" class="advanced-body">
          <div class="form-field">
            <span>{{ t('components.main.form.labels.proxyMode') }}</span>
            <div class="segmented" role="radiogroup" :aria-label="t('components.main.form.labels.proxyMode')">
              <button
                v-for="mode in proxyModeOptions"
                :key="mode.value"
                type="button"
                role="radio"
                :aria-checked="form.proxyMode === mode.value"
                :class="['segmented-item', { active: form.proxyMode === mode.value }]"
                @click="form.proxyMode = mode.value"
              >
                {{ t(mode.labelKey) }}
              </button>
            </div>
          </div>

          <label v-if="form.proxyMode === 'custom'" class="form-field">
            <span class="label-row">
              {{ t('components.main.form.labels.proxyUrl') }}
              <span v-if="proxyUrlError" class="field-error">
                {{ proxyUrlError }}
              </span>
            </span>
            <BaseInput
              v-model="form.proxyUrl"
              type="text"
              :placeholder="t('components.main.form.placeholders.proxyUrl')"
              :class="{ 'has-error': !!proxyUrlError }"
            />
          </label>

          <div class="form-field">
            <label class="insecure-tls-option">
              <input type="checkbox" v-model="form.insecureSkipVerify" />
              <span>{{ t('components.main.form.labels.insecureSkipVerify') }}</span>
            </label>
            <p v-if="form.insecureSkipVerify" class="insecure-tls-warning">
              {{ t('components.main.form.insecureSkipVerifyWarning') }}
            </p>
          </div>

          <div class="form-field">
            <ModelWhitelistEditor v-model="form.supportedModels" />
          </div>

          <div class="form-field">
            <ModelMappingEditor v-model="form.modelMapping" />
          </div>

          <div v-if="platform === 'claude'" class="form-field">
            <RequestBodyReplacementEditor v-model="form.requestBodyReplacements" />
          </div>
        </div>
      </section>

      <footer class="form-actions">
        <BaseButton variant="outline" type="button" @click="$emit('close')">
          {{ t('components.main.form.actions.cancel') }}
        </BaseButton>
        <BaseButton type="submit">
          {{ t('components.main.form.actions.save') }}
        </BaseButton>
      </footer>
    </form>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseButton from '../common/BaseButton.vue'
import BaseInput from '../common/BaseInput.vue'
import BaseModal from '../common/BaseModal.vue'
import ModelMappingEditor from '../common/ModelMappingEditor.vue'
import ModelWhitelistEditor from '../common/ModelWhitelistEditor.vue'
import RequestBodyReplacementEditor from '../common/RequestBodyReplacementEditor.vue'
import type { ProviderTab } from '../../constants/platforms'
import type { VendorForm } from './types'

const props = defineProps<{
  apiUrlError: string
  editing: boolean
  form: VendorForm
  nameError: string
  open: boolean
  platform: ProviderTab
  proxyUrlError: string
}>()

defineEmits<{
  close: []
  submit: []
}>()

const { t } = useI18n()

// 每次打开弹窗都收起高级区；代理地址校验失败时自动展开，避免报错被折叠隐藏。
const advancedOpen = ref(false)

watch(
  () => props.open,
  (open) => {
    if (open) {
      advancedOpen.value = false
    }
  },
)

watch(
  () => props.proxyUrlError,
  (error) => {
    if (error) {
      advancedOpen.value = true
    }
  },
)

const proxyModeOptions = [
  { value: '', labelKey: 'components.main.form.proxyMode.inherit' },
  { value: 'custom', labelKey: 'components.main.form.proxyMode.custom' },
  { value: 'direct', labelKey: 'components.main.form.proxyMode.direct' },
] as const
</script>
