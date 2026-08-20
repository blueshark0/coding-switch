<template>
  <div class="field-help-label">
    <span>{{ label }}</span>
    <button type="button" class="help-icon" :data-tooltip="tooltip">
      <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
        <path
          d="M8 1a7 7 0 100 14A7 7 0 008 1zm0 13A6 6 0 118 2a6 6 0 010 12zm0-9.5a.75.75 0 01.75.75v4a.75.75 0 01-1.5 0v-4A.75.75 0 018 4.5zm0 7.5a1 1 0 100-2 1 1 0 000 2z"
          fill="currentColor"
        />
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  tooltip: string
}>()
</script>

<style scoped>
.field-help-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  font-size: 0.875rem;
  color: var(--foreground);
}

.help-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  border: none;
  background: none;
  color: var(--foreground-muted);
  cursor: help;
  border-radius: 4px;
  transition: all 0.2s;
}

.help-icon:hover {
  color: var(--foreground);
  background-color: var(--background-hover);
}
</style>
