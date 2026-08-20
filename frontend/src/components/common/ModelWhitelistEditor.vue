<template>
  <div class="model-whitelist-editor">
    <div class="editor-header">
      <FieldHelpLabel
        :label="$t('components.provider.modelWhitelist.label')"
        :tooltip="$t('components.provider.modelWhitelist.tooltip')"
      />
    </div>

    <div v-if="modelList.length > 0" class="model-tags">
      <div
        v-for="(model, index) in modelList"
        :key="index"
        class="model-tag"
      >
        <span class="model-name" :class="{ wildcard: isWildcard(model) }">{{ model }}</span>
        <button
          type="button"
          class="tag-remove"
          :aria-label="$t('components.provider.modelWhitelist.remove')"
          @click="removeModel(index)"
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

    <div class="model-input-row">
      <BaseInput
        v-model="newModel"
        type="text"
        :placeholder="$t('components.provider.modelWhitelist.placeholder')"
        @keydown.enter.prevent="addModel"
      />
      <BaseButton
        type="button"
        variant="outline"
        @click="addModel"
      >
        {{ $t('components.provider.modelWhitelist.add') }}
      </BaseButton>
    </div>

    <EditorHelpBox :title="$t('components.provider.modelWhitelist.examples.title')">
      <ul class="help-list">
        <li>
          <code>claude-sonnet-4</code> - {{ $t('components.provider.modelWhitelist.examples.exact') }}
        </li>
        <li>
          <code>claude-*</code> - {{ $t('components.provider.modelWhitelist.examples.prefix') }}
        </li>
        <li>
          <code>anthropic/claude-*</code> - {{ $t('components.provider.modelWhitelist.examples.vendor') }}
        </li>
      </ul>
    </EditorHelpBox>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import BaseInput from './BaseInput.vue'
import BaseButton from './BaseButton.vue'
import EditorHelpBox from './EditorHelpBox.vue'
import FieldHelpLabel from './FieldHelpLabel.vue'
import { useRecordModelValue } from './useRecordModelValue'

interface Props {
  modelValue?: Record<string, boolean>
}

interface Emits {
  (e: 'update:modelValue', value: Record<string, boolean>): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { recordValue, updateRecord } = useRecordModelValue<boolean>(props, emit, {
  initializeEmpty: true,
})

const modelList = computed(() => {
  return Object.keys(recordValue.value).filter((key) => recordValue.value[key])
})

const newModel = ref('')

const isWildcard = (model: string) => model.includes('*')

const addModel = () => {
  const trimmed = newModel.value.trim()
  if (!trimmed) return

  if (recordValue.value[trimmed]) {
    newModel.value = ''
    return
  }

  updateRecord((updated) => {
    updated[trimmed] = true
  })
  newModel.value = ''
}

const removeModel = (index: number) => {
  const modelName = modelList.value[index]
  if (!modelName) return

  updateRecord((updated) => {
    delete updated[modelName]
  })
}
</script>

<style scoped>
.model-whitelist-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.model-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 10px;
  background-color: var(--background-secondary);
  border-radius: 8px;
  min-height: 44px;
}

.model-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px 4px 10px;
  background-color: var(--background);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 0.8125rem;
  line-height: 1.4;
  transition: all 0.2s;
}

.model-tag:hover {
  background-color: var(--background-hover);
}

.model-name {
  color: var(--foreground);
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
}

.model-name.wildcard {
  color: var(--accent-primary);
  font-weight: 500;
}

.tag-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  border: none;
  background: none;
  color: var(--foreground-muted);
  cursor: pointer;
  border-radius: 3px;
  transition: all 0.2s;
}

.tag-remove:hover {
  color: var(--error);
  background-color: var(--error-bg);
}

.model-input-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.model-input-row :deep(input) {
  flex: 1;
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
}
</style>
