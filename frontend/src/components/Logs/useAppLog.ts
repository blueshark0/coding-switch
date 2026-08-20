import { onMounted, onUnmounted, ref, watch } from 'vue'
import { fetchAppLogTail, type AppLogEntry, type AppLogLevel } from '../../services/appLog'

const FOLLOW_INTERVAL_MS = 3000
const KEYWORD_DEBOUNCE_MS = 300
// 前端保留的最大条目数，跟随模式长时间运行时防止内存无限增长。
const MAX_KEPT_ENTRIES = 2000

export const useAppLog = () => {
  const entries = ref<AppLogEntry[]>([])
  const level = ref<AppLogLevel>('')
  const keyword = ref('')
  const following = ref(true)
  const loading = ref(false)
  const truncated = ref(false)
  let nextOffset = 0
  let loadPromise: Promise<void> | null = null
  let followTimer: number | undefined
  let keywordTimer: number | undefined

  const load = (options: { reset?: boolean } = {}) => {
    if (loadPromise) {
      return loadPromise
    }
    loadPromise = (async () => {
      loading.value = true
      try {
        const chunk = await fetchAppLogTail({
          offset: options.reset ? 0 : nextOffset,
          level: level.value,
          keyword: keyword.value,
        })
        if (options.reset || chunk.reset) {
          entries.value = []
          truncated.value = chunk.truncated
        }
        const fresh = chunk.entries ?? []
        if (fresh.length) {
          entries.value = [...entries.value, ...fresh].slice(-MAX_KEPT_ENTRIES)
        }
        nextOffset = chunk.next_offset
      } catch (error) {
        console.error('failed to load app log', error)
      } finally {
        loading.value = false
        loadPromise = null
      }
    })()
    return loadPromise
  }

  // 过滤条件变化/手动刷新：等在途请求结束后从尾部全量重拉。
  const reload = async () => {
    if (loadPromise) {
      try {
        await loadPromise
      } catch {
        // load 内部已处理错误
      }
    }
    await load({ reset: true })
  }

  const stopFollow = () => {
    if (followTimer) {
      clearInterval(followTimer)
      followTimer = undefined
    }
  }

  const startFollow = () => {
    stopFollow()
    followTimer = window.setInterval(() => {
      if (!document.hidden) {
        void load()
      }
    }, FOLLOW_INTERVAL_MS)
  }

  watch(following, (enabled) => {
    if (enabled) {
      void load()
      startFollow()
    } else {
      stopFollow()
    }
  })

  watch(level, () => {
    void reload()
  })

  watch(keyword, () => {
    if (keywordTimer) {
      clearTimeout(keywordTimer)
    }
    keywordTimer = window.setTimeout(() => {
      keywordTimer = undefined
      void reload()
    }, KEYWORD_DEBOUNCE_MS)
  })

  onMounted(() => {
    void load({ reset: true })
    if (following.value) {
      startFollow()
    }
  })

  onUnmounted(() => {
    stopFollow()
    if (keywordTimer) {
      clearTimeout(keywordTimer)
      keywordTimer = undefined
    }
  })

  return {
    entries,
    following,
    keyword,
    level,
    loading,
    manualRefresh: reload,
    truncated,
  }
}
