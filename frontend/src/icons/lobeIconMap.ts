import fallbackIcons from './fallbackLobeIcons'

const iconLoaders = import.meta.glob(
  '../../node_modules/@lobehub/icons-static-svg/icons/*.svg',
  {
    import: 'default',
    query: '?raw',
  },
) as Record<string, () => Promise<string>>

const fallbackIconMap = Object.keys(fallbackIcons).reduce<Record<string, string>>((acc, key) => {
  acc[key.toLowerCase()] = fallbackIcons[key]
  return acc
}, {})

const normalizedIconEntries = Object.keys(iconLoaders).map((path) => {
  const name = path
    .split('/')
    .pop()
    ?.replace('.svg', '')
    ?.toLowerCase()

  return [name, path] as const
})

const iconPathMap = normalizedIconEntries.reduce<Record<string, string>>((acc, [name, path]) => {
  if (name) {
    acc[name] = path
  }
  return acc
}, {})

const iconNameSet = new Set<string>([
  ...Object.keys(iconPathMap),
  ...Object.keys(fallbackIconMap),
])

const iconMarkupCache = new Map<string, string>()

const normalizeIconName = (name: string) => name.trim().toLowerCase()

export const getIconOptions = () => Array.from(iconNameSet).sort((left, right) => left.localeCompare(right))

export const loadLobeIcon = async (name: string) => {
  const normalizedName = normalizeIconName(name)
  if (!normalizedName) {
    return ''
  }

  const cached = iconMarkupCache.get(normalizedName)
  if (cached) {
    return cached
  }

  const fallbackIcon = fallbackIconMap[normalizedName]
  if (fallbackIcon) {
    iconMarkupCache.set(normalizedName, fallbackIcon)
    return fallbackIcon
  }

  const iconPath = iconPathMap[normalizedName]
  const loader = iconPath ? iconLoaders[iconPath] : undefined
  if (!loader) {
    return ''
  }

  try {
    const svg = await loader()
    iconMarkupCache.set(normalizedName, svg)
    return svg
  } catch (error) {
    console.error('failed to load icon', normalizedName, error)
    return ''
  }
}
