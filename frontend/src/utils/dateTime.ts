const BACKEND_DATE_PATTERN =
  /^(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2}) ([+-]\d{4}) UTC$/

export const padDatePart = (num: number) => num.toString().padStart(2, '0')

export const parseDateTime = (value?: unknown): Date | null => {
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? null : value
  }

  if (typeof value !== 'string') {
    return null
  }

  const raw = value.trim()
  if (!raw) {
    return null
  }

  const normalized = raw.replace(' ', 'T')
  const attempts = [raw, normalized, `${normalized}Z`]

  for (const candidate of attempts) {
    const parsed = new Date(candidate)
    if (!Number.isNaN(parsed.getTime())) {
      return parsed
    }
  }

  const match = raw.match(BACKEND_DATE_PATTERN)
  if (!match) {
    return null
  }

  const [, day, time, zone] = match
  const zoneWithColon = `${zone.slice(0, 3)}:${zone.slice(3)}`
  const parsed = new Date(`${day}T${time}${zoneWithColon}`)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

export const formatDateTime = (date: Date) => {
  return `${date.getFullYear()}-${padDatePart(date.getMonth() + 1)}-${padDatePart(date.getDate())} ${padDatePart(date.getHours())}:${padDatePart(date.getMinutes())}:${padDatePart(date.getSeconds())}`
}

export const formatHourBucketLabel = (value?: string) => {
  if (!value) return ''

  const parsed = parseDateTime(value)
  if (parsed) {
    return `${padDatePart(parsed.getHours())}:00`
  }

  const match = value.match(/(\d{2}):(\d{2})/)
  if (match) {
    return `${match[1]}:${match[2]}`
  }

  return value
}

export const formatDayBucketLabel = (value?: string) => {
  if (!value) return ''

  const parsed = parseDateTime(value)
  if (parsed) {
    return `${padDatePart(parsed.getMonth() + 1)}-${padDatePart(parsed.getDate())}`
  }

  if (value.length >= 10) {
    return value.slice(5, 10)
  }

  return value
}

export const startOfTodayLocal = () => {
  const now = new Date()
  now.setHours(0, 0, 0, 0)
  return now
}
