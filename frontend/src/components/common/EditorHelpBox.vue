<template>
  <div class="help-text">
    <p v-if="title" class="help-example">
      <strong>{{ title }}</strong>
    </p>
    <div class="help-content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title?: string
}>()
</script>

<style scoped>
.help-text {
  padding: 12px;
  background-color: var(--background-secondary);
  border-radius: 8px;
  font-size: 0.8125rem;
  color: var(--foreground-muted);
}

.help-example {
  margin-bottom: 8px;
  color: var(--foreground);
}

.help-content :deep(.help-list) {
  margin: 0;
  padding-left: 20px;
  list-style: disc;
}

.help-content :deep(.help-list li) {
  margin-bottom: 6px;
  line-height: 1.5;
}

.help-content :deep(.help-list code) {
  padding: 2px 6px;
  background-color: var(--background);
  border: 1px solid var(--border);
  border-radius: 4px;
  font-family: 'SF Mono', 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: 0.75rem;
  color: var(--accent-primary);
}

.help-content :deep(.help-desc) {
  font-size: 0.75rem;
  color: var(--foreground-muted);
  font-style: italic;
}
</style>
