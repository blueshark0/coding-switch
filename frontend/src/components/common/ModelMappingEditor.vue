<template>
  <div class="model-mapping-editor">
    <div class="editor-header">
      <FieldHelpLabel
        :label="$t('components.provider.modelMapping.label')"
        :tooltip="$t('components.provider.modelMapping.tooltip')"
      />
    </div>

    <div v-if="mappingList.length > 0" class="mapping-list">
      <div
        v-for="(mapping, index) in mappingList"
        :key="index"
        class="mapping-row"
      >
        <div class="mapping-content">
          <code class="mapping-key" :class="{ wildcard: isWildcard(mapping.key) }">
            {{ mapping.key }}
          </code>
          <svg class="mapping-arrow" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
            <path
              d="M6 4l4 4-4 4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          <code class="mapping-value" :class="{ wildcard: isWildcard(mapping.value) }">
            {{ mapping.value }}
          </code>
        </div>
        <button
          type="button"
          class="mapping-remove"
          :aria-label="$t('components.provider.modelMapping.remove')"
          @click="removeMapping(index)"
        >
          <svg viewBox="0 0 12 12" width="10" height="10" aria-hidden="true">
            <path
              d="M3 3l6 6M9 3l-6 6"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
            />
          </svg>
        </button>
      </div>
    </div>

    <div class="mapping-input-row">
      <BaseInput
        v-model="newKey"
        type="text"
        :placeholder="$t('components.provider.modelMapping.keyPlaceholder')"
        @keydown.enter.prevent="focusValueInput"
      />
      <svg class="input-arrow" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
        <path
          d="M6 4l4 4-4 4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
      <BaseInput
        ref="valueInputRef"
        v-model="newValue"
        type="text"
        :placeholder="$t('components.provider.modelMapping.valuePlaceholder')"
        @keydown.enter.prevent="addMapping"
      />
      <BaseButton
        type="button"
        variant="outline"
        @click="addMapping"
      >
        {{ $t('components.provider.modelMapping.add') }}
      </BaseButton>
    </div>

    <EditorHelpBox :title="$t('components.provider.modelMapping.examples.title')">
      <ul class="help-list">
        <li>
          <code>claude-sonnet-4</code> → <code>anthropic/claude-sonnet-4</code><br />
          <span class="help-desc">{{ $t('components.provider.modelMapping.examples.exact') }}</span>
        </li>
        <li>
          <code>claude-*</code> → <code>anthropic/claude-*</code><br />
          <span class="help-desc">{{ $t('components.provider.modelMapping.examples.wildcard') }}</span>
        </li>
        <li>
          <code>gpt-*</code> → <code>openai/gpt-*</code><br />
          <span class="help-desc">{{ $t('components.provider.modelMapping.examples.prefix') }}</span>
        </li>
      </ul>
    </EditorHelpBox>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import BaseInput from './BaseInput.vue'
import BaseButton from './BaseButton.vue'
import EditorHelpBox from './EditorHelpBox.vue'
import FieldHelpLabel from './FieldHelpLabel.vue'
import { useRecordModelValue } from './useRecordModelValue'

interface Props {
  modelValue?: Record<string, string>
}

interface Emits {
  (e: 'update:modelValue', value: Record<string, string>): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { recordValue, updateRecord } = useRecordModelValue<string>(props, emit)

const mappingList = computed(() => {
  return Object.entries(recordValue.value).map(([key, value]) => ({ key, value }))
})

const newKey = ref('')
const newValue = ref('')
const valueInputRef = ref<{ focus: () => void } | null>(null)

const isWildcard = (text: string) => text.includes('*')

const focusValueInput = () => {
  valueInputRef.value?.focus()
}

const addMapping = () => {
  const key = newKey.value.trim()
  const value = newValue.value.trim()

  if (!key || !value) return

  updateRecord((updated) => {
    updated[key] = value
  })
  newKey.value = ''
  newValue.value = ''
}

const removeMapping = (index: number) => {
  const mapping = mappingList.value[index]
  if (!mapping) return

  updateRecord((updated) => {
    delete updated[mapping.key]
  })
}
</script>

<style scoped>
.model-mapping-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.mapping-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  background-color: var(--background-secondary);
  border-radius: 8px;
}

.mapping-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  background-color: var(--background);
  border: 1px solid var(--border);
  border-radius: 6px;
  transition: all 0.2s;
}

.mapping-row:hover {
  background-color: var(--background-hover);
}

.mapping-content {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.mapping-key,
.mapping-value {
  padding: 3px 7px;
  background-color: var(--background-secondary);
  border: 1px solid var(--border);
  border-radius: 4px;
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: 0.75rem;
  color: var(--foreground);
  word-break: break-all;
}

.mapping-key.wildcard,
.mapping-value.wildcard {
  color: var(--accent-primary);
  font-weight: 500;
}

.mapping-arrow {
  flex-shrink: 0;
  color: var(--foreground-muted);
}

.mapping-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  border: none;
  background: none;
  color: var(--foreground-muted);
  cursor: pointer;
  border-radius: 3px;
  flex-shrink: 0;
  transition: all 0.2s;
}

.mapping-remove:hover {
  color: var(--error);
  background-color: var(--error-bg);
}

.mapping-input-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.mapping-input-row :deep(input) {
  flex: 1;
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
}

.input-arrow {
  flex-shrink: 0;
  color: var(--foreground-muted);
}
</style>
