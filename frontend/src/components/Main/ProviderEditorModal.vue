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

      <div class="form-field">
        <span>{{ t('components.main.form.labels.proxyMode') }}</span>
        <div class="proxy-mode-options">
          <label class="proxy-mode-option">
            <input type="radio" v-model="form.proxyMode" value="" />
            <span>{{ t('components.main.form.proxyMode.inherit') }}</span>
          </label>
          <label class="proxy-mode-option">
            <input type="radio" v-model="form.proxyMode" value="custom" />
            <span>{{ t('components.main.form.proxyMode.custom') }}</span>
          </label>
          <label class="proxy-mode-option">
            <input type="radio" v-model="form.proxyMode" value="direct" />
            <span>{{ t('components.main.form.proxyMode.direct') }}</span>
          </label>
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
import { useI18n } from 'vue-i18n'
import BaseButton from '../common/BaseButton.vue'
import BaseInput from '../common/BaseInput.vue'
import BaseModal from '../common/BaseModal.vue'
import ModelMappingEditor from '../common/ModelMappingEditor.vue'
import ModelWhitelistEditor from '../common/ModelWhitelistEditor.vue'
import RequestBodyReplacementEditor from '../common/RequestBodyReplacementEditor.vue'
import type { ProviderTab } from '../../constants/platforms'
import type { VendorForm } from './types'

defineProps<{
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
</script>
