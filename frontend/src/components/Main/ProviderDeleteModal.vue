<template>
  <BaseModal
    :open="open"
    :title="t('components.main.form.confirmDeleteTitle')"
    variant="confirm"
    @close="$emit('close')"
  >
    <div class="confirm-body">
      <p>
        {{ t('components.main.form.confirmDeleteMessage', { name }) }}
      </p>
    </div>
    <footer class="form-actions confirm-actions">
      <BaseButton variant="outline" type="button" @click="$emit('close')">
        {{ t('components.main.form.actions.cancel') }}
      </BaseButton>
      <BaseButton variant="danger" type="button" @click="$emit('confirm')">
        {{ t('components.main.form.actions.delete') }}
      </BaseButton>
    </footer>
  </BaseModal>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseButton from '../common/BaseButton.vue'
import BaseModal from '../common/BaseModal.vue'

defineProps<{
  name: string
  open: boolean
}>()

defineEmits<{
  close: []
  confirm: []
}>()

const { t } = useI18n()
</script>
