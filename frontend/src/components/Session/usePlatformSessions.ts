import { computed, ref } from 'vue'
import {
  parseProviderTab,
  providerTabs,
  type ProviderTab,
} from '../../constants/platforms'
import {
  fetchPlatformSessions,
  unbindSession as requestUnbindSession,
  type SessionBinding,
} from '../../services/sessions'
import { parseDateTime } from '../../utils/dateTime'

type TranslateFn = (key: string, named?: Record<string, unknown>) => string
type LocaleRef = { value: string }

type UsePlatformSessionsOptions = {
  locale: LocaleRef
  syncPlatformQuery: (platform: ProviderTab) => Promise<void> | void
  t: TranslateFn
}

export type SessionProviderGroup = {
  providerName: string
  sessions: SessionBinding[]
}

export const usePlatformSessions = ({
  locale,
  syncPlatformQuery,
  t,
}: UsePlatformSessionsOptions) => {
  const tabs = providerTabs
  const selectedPlatform = ref<ProviderTab>('claude')
  const sessions = ref<SessionBinding[]>([])
  const loading = ref(false)

  const dateFormatter = computed(
    () =>
      new Intl.DateTimeFormat(locale.value || 'en', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }),
  )

  const loadSessions = async (platform = selectedPlatform.value) => {
    loading.value = true
    try {
      const loadedSessions = await fetchPlatformSessions(platform)
      sessions.value = Array.isArray(loadedSessions) ? loadedSessions : []
    } catch (error) {
      console.error('Failed to load platform sessions', error)
      sessions.value = []
    } finally {
      loading.value = false
    }
  }

  const initializeSessions = async (platformQuery: unknown) => {
    const platform = parseProviderTab(platformQuery)
    selectedPlatform.value = platform
    if (platformQuery !== platform) {
      await syncPlatformQuery(platform)
    }
    await loadSessions(platform)
  }

  const switchPlatform = async (platform: ProviderTab) => {
    if (platform === selectedPlatform.value) return
    selectedPlatform.value = platform
    await syncPlatformQuery(platform)
    await loadSessions(platform)
  }

  const refreshSessions = () => {
    void loadSessions()
  }

  const truncateSessionId = (sessionId: string) => {
    if (sessionId.length <= 12) {
      return sessionId
    }
    return `${sessionId.slice(0, 12)}...`
  }

  const formatRelativeTime = (value: unknown) => {
    const date = parseDateTime(value)
    if (!date) return t('components.sessions.dateUnknown')

    const diffMinutes = Math.floor((Date.now() - date.getTime()) / 60000)
    if (diffMinutes < 1) {
      return t('components.sessions.justNow')
    }
    if (diffMinutes < 60) {
      return t('components.sessions.minutesAgo', { count: diffMinutes })
    }
    if (diffMinutes < 24 * 60) {
      return t('components.sessions.hoursAgo', { count: Math.floor(diffMinutes / 60) })
    }
    return dateFormatter.value.format(date)
  }

  const providerGroups = computed<SessionProviderGroup[]>(() => {
    const groups = new Map<string, SessionBinding[]>()
    for (const session of sessions.value ?? []) {
      const providerName = session.provider_name?.trim() || t('components.sessions.unknownProvider')
      if (!groups.has(providerName)) {
        groups.set(providerName, [])
      }
      groups.get(providerName)?.push(session)
    }
    return Array.from(groups.entries()).map(([providerName, providerSessions]) => ({
      providerName,
      sessions: providerSessions,
    }))
  })

  const unbindSession = async (session: SessionBinding) => {
    if (!window.confirm(t('components.sessions.confirmUnbindSession'))) {
      return
    }
    try {
      await requestUnbindSession(session.platform, session.session_id)
      await loadSessions()
    } catch (error) {
      console.error('Failed to unbind session', error)
      window.alert(t('components.sessions.unbindSessionFailed'))
    }
  }

  return {
    formatRelativeTime,
    initializeSessions,
    loading,
    providerGroups,
    refreshSessions,
    selectedPlatform,
    switchPlatform,
    tabs,
    truncateSessionId,
    unbindSession,
  }
}
