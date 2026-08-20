<template>
  <div class="request-body-replacement-editor">
    <div class="editor-header">
      <FieldHelpLabel
        :label="$t('components.provider.requestBodyReplacements.label')"
        :tooltip="$t('components.provider.requestBodyReplacements.tooltip')"
      />
    </div>

    <div v-if="replacementList.length > 0" class="replacement-list">
      <div
        v-for="(replacement, index) in replacementList"
        :key="index"
        class="replacement-row"
      >
        <div class="replacement-content">
          <code class="replacement-path">{{ replacement.path }}</code>
          <svg class="replacement-arrow" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
            <path
              d="M6 4l4 4-4 4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          <code class="replacement-value">{{ replacement.value }}</code>
        </div>
        <button
          type="button"
          class="replacement-remove"
          :aria-label="$t('components.provider.requestBodyReplacements.remove')"
          @click="removeReplacement(index)"
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

    <div class="replacement-inputs">
      <BaseInput
        v-model="newPath"
        type="text"
        :placeholder="$t('components.provider.requestBodyReplacements.pathPlaceholder')"
        @keydown.enter.prevent="focusValueInput"
      />
      <BaseTextarea
        ref="valueInputRef"
        v-model="newValue"
        rows="3"
        :placeholder="$t('components.provider.requestBodyReplacements.valuePlaceholder')"
      />
      <BaseButton type="button" variant="outline" @click="addReplacement">
        {{ $t('components.provider.requestBodyReplacements.add') }}
      </BaseButton>
    </div>

    <EditorHelpBox :title="$t('components.provider.requestBodyReplacements.examples.title')">
      <ul class="help-list">
        <li>
          <code>metadata.user_id</code> → <code>"fixed-user"</code><br />
          <span class="help-desc">{{ $t('components.provider.requestBodyReplacements.examples.string') }}</span>
        </li>
        <li>
          <code>temperature</code> → <code>0.2</code><br />
          <span class="help-desc">{{ $t('components.provider.requestBodyReplacements.examples.number') }}</span>
        </li>
        <li>
          <code>thinking</code> → <code>{"type":"disabled"}</code><br />
          <span class="help-desc">{{ $t('components.provider.requestBodyReplacements.examples.object') }}</span>
        </li>
      </ul>
    </EditorHelpBox>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import BaseButton from './BaseButton.vue'
import BaseInput from './BaseInput.vue'
import BaseTextarea from './BaseTextarea.vue'
import EditorHelpBox from './EditorHelpBox.vue'
import FieldHelpLabel from './FieldHelpLabel.vue'

export type RequestBodyReplacement = {
  path: string
  value: string
}

interface Props {
  modelValue?: RequestBodyReplacement[]
}

interface Emits {
  (e: 'update:modelValue', value: RequestBodyReplacement[]): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const replacementList = computed(() => props.modelValue ?? [])
const newPath = ref('')
const newValue = ref('')
const valueInputRef = ref<{ focus: () => void } | null>(null)

const focusValueInput = () => {
  valueInputRef.value?.focus()
}

const addReplacement = () => {
  const path = newPath.value.trim()
  const value = newValue.value.trim()
  if (!path || !value) return

  emit('update:modelValue', [...replacementList.value, { path, value }])
  newPath.value = ''
  newValue.value = ''
}

const removeReplacement = (index: number) => {
  emit('update:modelValue', replacementList.value.filter((_, itemIndex) => itemIndex !== index))
}
</script>

<style scoped>
.request-body-replacement-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.replacement-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  background-color: var(--background-secondary);
  border-radius: 8px;
}

.replacement-row {
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

.replacement-row:hover {
  background-color: var(--background-hover);
}

.replacement-content {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.replacement-path,
.replacement-value {
  padding: 3px 7px;
  background-color: var(--background-secondary);
  border: 1px solid var(--border);
  border-radius: 4px;
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: 0.75rem;
  color: var(--foreground);
  word-break: break-all;
}

.replacement-path {
  color: var(--accent-primary);
  font-weight: 500;
}

.replacement-value {
  max-width: 60%;
}

.replacement-arrow {
  flex-shrink: 0;
  color: var(--foreground-muted);
}

.replacement-remove {
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

.replacement-remove:hover {
  color: var(--error);
  background-color: var(--error-bg);
}

.replacement-inputs {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 8px;
}

.replacement-inputs :deep(input),
.replacement-inputs :deep(textarea) {
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
}

.replacement-inputs :deep(textarea) {
  min-height: 72px;
  resize: vertical;
}
</style>
