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
        <span>{{ t('components.main.form.labels.icon') }}</span>
        <Listbox v-model="form.icon" v-slot="{ open: optionsOpen }">
          <div class="icon-select">
            <ListboxButton class="icon-select-button">
              <AsyncIcon :icon="form.icon" class="icon-preview" aria-hidden="true" />
              <span class="icon-select-label">{{ form.icon }}</span>
              <svg viewBox="0 0 20 20" aria-hidden="true">
                <path
                  d="M6 8l4 4 4-4"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  fill="none"
                />
              </svg>
            </ListboxButton>
            <ListboxOptions v-if="optionsOpen" class="icon-select-options">
              <ListboxOption
                v-for="iconName in iconOptions"
                :key="iconName"
                :value="iconName"
                v-slot="{ active, selected }"
              >
                <div :class="['icon-option', { active, selected }]">
                  <span class="icon-name">{{ iconName }}</span>
                </div>
              </ListboxOption>
            </ListboxOptions>
          </div>
        </Listbox>
      </div>

      <div class="form-field">
        <ModelWhitelistEditor v-model="form.supportedModels" />
      </div>

      <div class="form-field">
        <ModelMappingEditor v-model="form.modelMapping" />
      </div>

      <div class="form-field switch-field">
        <span>{{ t('components.main.form.labels.enabled') }}</span>
        <div class="switch-inline">
          <label class="mac-switch">
            <input type="checkbox" v-model="form.enabled" />
            <span></span>
          </label>
          <span class="switch-text">
            {{ form.enabled ? t('components.main.form.switch.on') : t('components.main.form.switch.off') }}
          </span>
        </div>
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
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Listbox, ListboxButton, ListboxOptions, ListboxOption } from '@headlessui/vue'
import { getIconOptions } from '../../icons/lobeIconMap'
import AsyncIcon from '../common/AsyncIcon.vue'
import BaseButton from '../common/BaseButton.vue'
import BaseInput from '../common/BaseInput.vue'
import BaseModal from '../common/BaseModal.vue'
import ModelMappingEditor from '../common/ModelMappingEditor.vue'
import ModelWhitelistEditor from '../common/ModelWhitelistEditor.vue'
import type { VendorForm } from './types'

defineProps<{
  apiUrlError: string
  editing: boolean
  form: VendorForm
  nameError: string
  open: boolean
  proxyUrlError: string
}>()

defineEmits<{
  close: []
  submit: []
}>()

const { t } = useI18n()

const iconOptions = computed(() => getIconOptions())
</script>
