<template>
  <span
    v-if="iconMarkup"
    class="async-icon"
    v-html="iconMarkup"
    aria-hidden="true"
  ></span>
  <span v-else-if="fallbackText" class="async-icon-fallback">
    {{ fallbackText }}
  </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { loadLobeIcon } from '../../icons/lobeIconMap'

const props = defineProps<{
  fallbackText?: string
  icon: string
}>()

const iconMarkup = ref('')

const normalizedFallbackText = computed(() => props.fallbackText?.trim() ?? '')

watch(
  () => props.icon,
  async (nextIcon) => {
    iconMarkup.value = await loadLobeIcon(nextIcon)
  },
  { immediate: true },
)
</script>
