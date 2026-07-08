import { TailAppLog } from '../../bindings/codeswitch/internal/interfaces/wails/applogfacade'
import type {
  AppLogChunk,
  AppLogEntry,
} from '../../bindings/codeswitch/internal/observability/domain/models'

export type { AppLogChunk, AppLogEntry }

export const APP_LOG_LEVELS = ['', 'INFO', 'WARN', 'ERROR', 'none'] as const
export type AppLogLevel = (typeof APP_LOG_LEVELS)[number]

export const DEFAULT_APP_LOG_MAX_LINES = 500

type AppLogTailQuery = {
  offset?: number
  maxLines?: number
  level?: string
  keyword?: string
}

export const fetchAppLogTail = async (query: AppLogTailQuery = {}): Promise<AppLogChunk> => {
  const offset = Math.max(0, Math.floor(query.offset ?? 0))
  const maxLines = Math.max(0, Math.floor(query.maxLines ?? DEFAULT_APP_LOG_MAX_LINES))
  return TailAppLog(offset, maxLines, query.level ?? '', query.keyword ?? '')
}
