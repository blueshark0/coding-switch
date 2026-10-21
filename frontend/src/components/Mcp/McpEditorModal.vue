<template>
  <BaseModal :open="open" :title="title" @close="$emit('close')">
    <div class="mcp-editor">
      <label class="form-field">
        <span>{{ t('components.mcp.form.name') }}</span>
        <BaseInput v-model="form.name" :placeholder="t('components.mcp.form.namePlaceholder')" />
      </label>

      <label class="form-field">
        <span>{{ t('components.mcp.form.type') }}</span>
        <select v-model="form.type" class="mac-select">
          <option value="stdio">{{ t('components.mcp.form.typeStdio') }}</option>
          <option value="http">{{ t('components.mcp.form.typeHttp') }}</option>
        </select>
      </label>

      <template v-if="form.type === 'stdio'">
        <label class="form-field">
          <span>{{ t('components.mcp.form.command') }}</span>
          <BaseInput v-model="form.command" placeholder="npx / uvx / /usr/local/bin/xxx" />
        </label>
        <label class="form-field">
          <span>{{ t('components.mcp.form.args') }}</span>
          <BaseInput v-model="form.argsText" :placeholder="t('components.mcp.form.argsPlaceholder')" />
        </label>
        <label class="form-field">
          <span>{{ t('components.mcp.form.env') }}</span>
          <BaseTextarea v-model="form.envText" :placeholder="t('components.mcp.form.kvPlaceholder')" rows="3" />
        </label>
      </template>

      <template v-else>
        <label class="form-field">
          <span>{{ t('components.mcp.form.url') }}</span>
          <BaseInput v-model="form.url" placeholder="https://example.com/mcp" />
        </label>
        <label class="form-field">
          <span>{{ t('components.mcp.form.headers') }}</span>
          <BaseTextarea v-model="form.headersText" :placeholder="t('components.mcp.form.kvPlaceholder')" rows="3" />
        </label>
      </template>

      <label class="form-field">
        <span>{{ t('components.mcp.form.timeout') }}</span>
        <BaseInput v-model="form.timeoutText" type="number" min="1" max="300" />
      </label>

      <p v-if="error" class="form-error">{{ error }}</p>
      <p v-if="testResult" class="test-result" :class="{ ok: testResult.success }">
        <template v-if="testResult.success">
          {{ t('components.mcp.form.testOk', { count: testResult.toolCount, ms: testResult.durationMs }) }}
        </template>
        <template v-else>
          {{ t('components.mcp.form.testErr', { error: testResult.error }) }}
        </template>
      </p>

      <div class="form-actions">
        <BaseButton variant="outline" size="sm" :disabled="testing || saving" @click="$emit('test')">
          {{ testing ? t('components.mcp.form.testing') : t('components.mcp.form.test') }}
        </BaseButton>
        <span class="form-actions-spacer"></span>
        <BaseButton variant="outline" size="sm" :disabled="saving" @click="$emit('close')">
          {{ t('components.mcp.form.cancel') }}
        </BaseButton>
        <BaseButton variant="primary" size="sm" :disabled="saving || testing" @click="$emit('submit')">
          {{ saving ? t('components.mcp.form.saving') : t('components.mcp.form.save') }}
        </BaseButton>
      </div>
    </div>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseModal from '../common/BaseModal.vue'
import BaseButton from '../common/BaseButton.vue'
import BaseInput from '../common/BaseInput.vue'
import BaseTextarea from '../common/BaseTextarea.vue'
import type { McpServerForm } from './useMcpServers'
import type { TestResult } from '../../services/mcp'

const props = defineProps<{
  open: boolean
  editing: boolean
  form: McpServerForm
  error: string
  saving: boolean
  testing: boolean
  testResult: TestResult | null
}>()

defineEmits<{
  (e: 'close'): void
  (e: 'submit'): void
  (e: 'test'): void
}>()

const { t } = useI18n()
const title = computed(() =>
  props.editing ? t('components.mcp.form.titleEdit') : t('components.mcp.form.titleNew'),
)
</script>

<style scoped>
.mcp-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 0.85rem;
  color: var(--mac-text-secondary);
}

.form-error {
  color: var(--mac-danger, #d0342c);
  font-size: 0.85rem;
  margin: 0;
}

.test-result {
  margin: 0;
  font-size: 0.85rem;
  color: var(--mac-danger, #d0342c);
  word-break: break-all;
}

.test-result.ok {
  color: var(--mac-success, #1a7f37);
}

.form-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.form-actions-spacer {
  flex: 1;
}
</style>
