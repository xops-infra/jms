type ServerItem = {
  id: string
  name: string
  host: string
  user?: string
  profile?: string
  status?: string
  tags?: Record<string, unknown> | string[] | string
  allowed: boolean
}

type SSHOption = {
  user: string
  key_name?: string
  auth_type: string
  source?: 'managed_key' | 'password' | 'profile_fallback' | string
}

type TerminalPhase = 'idle' | 'connecting' | 'live' | 'closed' | 'disconnected'
type StatusTone = 'live' | 'connecting' | 'warning' | 'closed' | 'idle'
type StatusIconKind = 'running' | 'pending' | 'stopped' | 'warning' | 'closed' | 'idle'

const isSameSSHOption = (left?: SSHOption | null, right?: SSHOption | null) =>
  Boolean(
    left &&
      right &&
      left.user === right.user &&
      (left.key_name || '') === (right.key_name || '') &&
      left.auth_type === right.auth_type,
  )

const buildSSHOptionValue = (option?: SSHOption | null) =>
  option ? `${option.user}:::${option.key_name || ''}:::${option.auth_type}:::${option.source || ''}` : ''

const tokenizeQuery = (value: string) =>
  value
    .toLowerCase()
    .split(/[\s,，]+/)
    .map((token) => token.trim())
    .filter(Boolean)

const safeJson = (value: unknown) => {
  try {
    return JSON.stringify(value)
  } catch {
    return ''
  }
}

const toTagText = (value: unknown) => {
  if (value === null || value === undefined) return ''
  if (typeof value === 'string') return value.trim()
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return ''
}

const maybeTagPair = (value: unknown) => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const record = value as Record<string, unknown>
  const key = toTagText(record.Key ?? record.key ?? record.Name ?? record.name ?? record.TagKey ?? record.tag_key)
  const tagValue = toTagText(
    record.Value ?? record.value ?? record.Val ?? record.val ?? record.TagValue ?? record.tag_value,
  )
  if (!key || !tagValue) return null
  return `${key}:${tagValue}`
}

const collectTagLabels = (value: unknown, parentKey = '', labels: Set<string>) => {
  if (value === null || value === undefined) return

  const pair = maybeTagPair(value)
  if (pair) {
    labels.add(parentKey ? `${parentKey}:${pair}` : pair)
    return
  }

  const text = toTagText(value)
  if (text) {
    labels.add(parentKey ? `${parentKey}:${text}` : text)
    return
  }

  if (Array.isArray(value)) {
    value.forEach((entry) => collectTagLabels(entry, parentKey, labels))
    return
  }

  if (typeof value === 'object') {
    Object.entries(value).forEach(([key, entry]) => {
      const nextKey = parentKey ? `${parentKey}.${key}` : key
      collectTagLabels(entry, nextKey, labels)
    })
  }
}

const extractTagLabels = (tags?: ServerItem['tags']) => {
  if (!tags) return []
  const labels = new Set<string>()
  collectTagLabels(tags, '', labels)
  return Array.from(labels).filter(Boolean)
}

const extractTagTokens = (tags?: ServerItem['tags']) => {
  if (!tags) return []
  const labels = extractTagLabels(tags)
  const tokens = new Set<string>()
  labels.forEach((label) => {
    tokens.add(label)
    label
      .split(/[:=,./\s-]+/)
      .map((part) => part.trim())
      .filter(Boolean)
      .forEach((part) => tokens.add(part))
  })
  const json = safeJson(tags)
  if (json) {
    tokens.add(json)
  }
  return Array.from(tokens)
}

const highPriorityTagPrefixes = [
  'product:',
  'project:',
  'service:',
  'app:',
  'application:',
  'team:',
  'owner:',
  'env:',
  'environment:',
  'stage:',
  'role:',
  'cluster:',
  'eks:cluster-name:',
]

const lowPriorityTagPrefixes = [
  'aws:',
  'kubernetes.io/',
  'k8s.io/',
  'alpha.eksctl.io/',
  'eks.amazonaws.com/',
  'topology.kubernetes.io/',
  'node.kubernetes.io/',
  'karpenter.sh/',
]

const lowPriorityTagHints = ['autoscaling', 'groupname', 'nodegroup', 'fleet-id', ':owned', ':true']

const getTagPriorityScore = (label: string) => {
  const normalized = label.trim().toLowerCase()
  let score = 0

  if (highPriorityTagPrefixes.some((prefix) => normalized.startsWith(prefix))) score += 6
  if (lowPriorityTagPrefixes.some((prefix) => normalized.startsWith(prefix))) score -= 6
  if (lowPriorityTagHints.some((hint) => normalized.includes(hint))) score -= 4

  if (label.length <= 32) score += 2
  else if (label.length <= 48) score += 1
  else if (label.length >= 72) score -= 2

  return score
}

const splitTagLabels = (tags?: ServerItem['tags']) => {
  const labels = extractTagLabels(tags).filter(Boolean)
  if (labels.length === 0) {
    return { primary: [], secondary: [] }
  }

  const ranked = labels
    .map((label, index) => ({
      label,
      index,
      score: getTagPriorityScore(label),
    }))
    .sort((left, right) => right.score - left.score || left.label.length - right.label.length || left.index - right.index)

  const primary = ranked
    .filter((entry) => entry.score > 0)
    .slice(0, 3)
    .map((entry) => entry.label)

  if (primary.length === 0) {
    primary.push(ranked[0].label)
  }

  const primarySet = new Set(primary)
  return {
    primary,
    secondary: labels.filter((label) => !primarySet.has(label)),
  }
}

const getStatusMeta = (status?: string): { label: string; tone: StatusTone; icon: StatusIconKind } => {
  const normalized = (status || '').trim().toLowerCase()

  if (!normalized) return { label: 'UNKNOWN', tone: 'idle', icon: 'idle' }

  if (['running', 'online', 'active', 'ready', 'healthy', 'available'].includes(normalized)) {
    return { label: normalized.toUpperCase(), tone: 'live', icon: 'running' }
  }

  if (['pending', 'starting', 'creating', 'booting', 'provisioning', 'rebooting', 'initializing'].includes(normalized)) {
    return { label: normalized.toUpperCase(), tone: 'connecting', icon: 'pending' }
  }

  if (['stopped', 'offline', 'inactive', 'paused'].includes(normalized)) {
    return { label: normalized.toUpperCase(), tone: 'idle', icon: 'stopped' }
  }

  if (['stopping', 'shutting-down', 'deleting', 'terminating'].includes(normalized)) {
    return { label: normalized.toUpperCase(), tone: 'warning', icon: 'warning' }
  }

  if (['terminated', 'deleted', 'failed', 'error', 'unhealthy'].includes(normalized)) {
    return { label: normalized.toUpperCase(), tone: 'closed', icon: 'closed' }
  }

  return { label: normalized.toUpperCase(), tone: 'idle', icon: 'idle' }
}

const buildSearchText = (item: ServerItem) => {
  const parts: string[] = []
  const push = (value?: string) => {
    if (!value) return
    parts.push(value)
  }
  push(item.name)
  push(item.host)
  push(item.user)
  push(item.status)
  push(item.id)
  extractTagTokens(item.tags).forEach((token) => push(token))
  return parts.join(' ').toLowerCase()
}

const StatusIcon = ({ kind }: { kind: StatusIconKind }) => {
  if (kind === 'running') {
    return (
      <svg viewBox="0 0 16 16" aria-hidden="true">
        <circle cx="8" cy="8" r="3.5" fill="currentColor" />
        <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeOpacity="0.35" strokeWidth="1.5" />
      </svg>
    )
  }

  if (kind === 'pending') {
    return (
      <svg viewBox="0 0 16 16" aria-hidden="true">
        <path d="M8 2.25a5.75 5.75 0 1 0 5.4 7.7" fill="none" stroke="currentColor" strokeLinecap="round" strokeWidth="1.7" />
        <path d="M10.9 1.95v3.55H7.35" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.7" />
      </svg>
    )
  }

  if (kind === 'stopped') {
    return (
      <svg viewBox="0 0 16 16" aria-hidden="true">
        <rect x="3" y="3" width="4" height="10" rx="1.2" fill="currentColor" />
        <rect x="9" y="3" width="4" height="10" rx="1.2" fill="currentColor" opacity="0.55" />
      </svg>
    )
  }

  if (kind === 'warning') {
    return (
      <svg viewBox="0 0 16 16" aria-hidden="true">
        <path d="M8 2.1 14 13H2L8 2.1Z" fill="none" stroke="currentColor" strokeLinejoin="round" strokeWidth="1.5" />
        <path d="M8 5.5v3.7" stroke="currentColor" strokeLinecap="round" strokeWidth="1.7" />
        <circle cx="8" cy="11.7" r="0.9" fill="currentColor" />
      </svg>
    )
  }

  if (kind === 'closed') {
    return (
      <svg viewBox="0 0 16 16" aria-hidden="true">
        <circle cx="8" cy="8" r="5.75" fill="none" stroke="currentColor" strokeWidth="1.5" />
        <path d="M5.2 5.2 10.8 10.8M10.8 5.2 5.2 10.8" stroke="currentColor" strokeLinecap="round" strokeWidth="1.7" />
      </svg>
    )
  }

  return (
    <svg viewBox="0 0 16 16" aria-hidden="true">
      <circle cx="8" cy="8" r="5.75" fill="none" stroke="currentColor" strokeWidth="1.5" />
      <circle cx="8" cy="8" r="1.6" fill="currentColor" />
    </svg>
  )
}

const RefreshIcon = () => (
  <svg viewBox="0 0 16 16" aria-hidden="true">
    <path
      d="M13.2 7.2A5.2 5.2 0 1 1 11.7 3.6"
      fill="none"
      stroke="currentColor"
      strokeLinecap="round"
      strokeWidth="1.6"
    />
    <path
      d="M10.9 2.7h2.7v2.7"
      fill="none"
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth="1.6"
    />
  </svg>
)

const StatusBadge = ({ status, prefix }: { status?: string; prefix?: string }) => {
  const meta = getStatusMeta(status)
  return (
    <span className={`badge status-badge ${meta.tone}`}>
      <StatusIcon kind={meta.icon} />
      <span>{prefix ? `${prefix}${meta.label}` : meta.label}</span>
    </span>
  )
}

export type { ServerItem, SSHOption, TerminalPhase }
export { StatusBadge, RefreshIcon, buildSSHOptionValue, buildSearchText, isSameSSHOption, splitTagLabels, tokenizeQuery }
