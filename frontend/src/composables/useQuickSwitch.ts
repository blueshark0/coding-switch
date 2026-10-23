import { ref } from 'vue'

// 快速切换面板的全局开关：工作台入口与全局 ⌘K/Ctrl+K 共用。
const quickSwitchOpen = ref(false)
const quickSwitchPlatform = ref('')

export const useQuickSwitch = () => {
  const openQuickSwitch = (platform = '') => {
    quickSwitchPlatform.value = platform
    quickSwitchOpen.value = true
  }

  const closeQuickSwitch = () => {
    quickSwitchOpen.value = false
  }

  return {
    quickSwitchOpen,
    quickSwitchPlatform,
    openQuickSwitch,
    closeQuickSwitch,
  }
}
