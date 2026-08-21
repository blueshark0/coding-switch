<template>
  <section v-if="cards.length" class="archived-section">
    <button class="archived-toggle" type="button" @click="expanded = !expanded">
      <svg
        class="archived-chevron"
        :class="{ 'is-expanded': expanded }"
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
      {{ t('components.main.archivedSectionTitle') }} ({{ cards.length }})
    </button>

    <div v-show="expanded" class="archived-list">
      <div v-for="card in cards" :key="card.id" class="archived-row">
        <span class="archived-name">{{ card.name }}</span>

        <div class="card-actions">
          <button
            class="ghost-icon"
            :title="t('components.main.restore')"
            :aria-label="t('components.main.restore')"
            type="button"
            @click="$emit('restore', card)"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M9 15L3 9m0 0l6-6M3 9h12a6 6 0 010 12h-3"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
          <button
            class="ghost-icon"
            :title="t('components.main.form.actions.delete')"
            :aria-label="t('components.main.form.actions.delete')"
            type="button"
            @click="$emit('remove', card)"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M9 3h6m-7 4h8m-6 0v11m4-11v11M5 7h14l-.867 12.138A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.862L5 7z"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AutomationCard } from '../../data/cards'

defineProps<{
  cards: AutomationCard[]
}>()

defineEmits<{
  remove: [card: AutomationCard]
  restore: [card: AutomationCard]
}>()

const { t } = useI18n()
const expanded = ref(false)
</script>

<style scoped>
.archived-section {
  margin-top: 8px;
}

.archived-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  border: none;
  background: transparent;
  padding: 6px 4px;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--mac-text-secondary);
  cursor: pointer;
}

.archived-toggle:hover {
  color: var(--mac-text);
}

.archived-chevron {
  width: 14px;
  height: 14px;
  transition: transform 0.2s ease;
}

.archived-chevron.is-expanded {
  transform: rotate(90deg);
}

.archived-list {
  margin-top: 6px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.archived-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border: 1px dashed var(--mac-border);
  border-radius: 14px;
  background: var(--mac-surface);
  opacity: 0.85;
}

.archived-name {
  flex: 1;
  text-align: left;
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--mac-text-secondary);
}
</style>
