import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { TerminalView, type TerminalStateEvent } from '../components/TerminalView'
import { FileTransferPanel } from '../components/FileTransferPanel'
import { useAuthStore } from '../store/auth'
import { apiClient } from '../api/client'

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

type TerminalWorkspacePromptProps = {
  phase: TerminalPhase
  selectedServer: ServerItem | null
  selectedTagList: string[]
  sshOptions: SSHOption[]
  sshLoading: boolean
  sshError: string
  sshSelected: SSHOption | null
  host: string
  reason: string
  approvalName: string
  approvalPeriod: string
  approvalStatus: string
  approvalSubmitting: boolean
  onSelectSSH: (option: SSHOption) => void
  onConnect: () => void
  onRefreshSSH: () => void
  onApprovalNameChange: (value: string) => void
  onApprovalPeriodChange: (value: string) => void
  onSubmitApproval: () => void
}

const TerminalWorkspacePrompt = ({
  phase,
  selectedServer,
  selectedTagList,
  sshOptions,
  sshLoading,
  sshError,
  sshSelected,
  host,
  reason,
  approvalName,
  approvalPeriod,
  approvalStatus,
  approvalSubmitting,
  onSelectSSH,
  onConnect,
  onRefreshSSH,
  onApprovalNameChange,
  onApprovalPeriodChange,
  onSubmitApproval,
}: TerminalWorkspacePromptProps) => {
  const isSelectedTarget = Boolean(selectedServer && selectedServer.host === host)
  const canConnect = Boolean(selectedServer?.allowed && sshSelected && !sshLoading)
  const fallbackOnly = sshOptions.length > 0 && sshOptions.every((opt) => opt.source === 'profile_fallback')
  const compactSSHPicker = fallbackOnly || sshOptions.length > 4

  let eyebrow = 'Secure Workspace'
  let title = '选择一台机器开始本次会话'
  let description = '从左侧机器列表选择目标后，机器信息、标签、登录用户和连接操作都会直接展示在这里。'
  const usageTips = selectedServer
    ? selectedServer.allowed
      ? ['左侧选择目标机器', '确认登录用户或密钥', '点击“连接此机器”进入终端']
      : ['左侧选择目标机器', '填写申请名称与有效期', '提交申请后等待权限开通']
    : ['先从左侧列表选择机器', '系统会加载可用登录配置', '确认后即可发起连接']

  if (selectedServer) {
    eyebrow = selectedServer.allowed ? 'Ready To Connect' : 'Permission Required'
    title = selectedServer.name || selectedServer.host
    description = `${selectedServer.host}${selectedServer.user ? ` · 默认用户 ${selectedServer.user}` : ''}`

    if (phase === 'connecting' && isSelectedTarget) {
      eyebrow = 'Linking Session'
      description = '正在建立 SSH 与终端链路，通常几秒内即可进入 shell。'
    } else if (phase === 'closed' && isSelectedTarget) {
      eyebrow = 'Session Complete'
      description = '当前会话已结束，可以直接重新连接或切换其他机器。'
    } else if (phase === 'disconnected' && isSelectedTarget) {
      eyebrow = 'Link Interrupted'
      description = '连接已断开，保留了当前选择的机器和登录配置，可直接恢复。'
    }
  }

  const connectLabel =
    phase === 'connecting' && isSelectedTarget
      ? '建立中...'
      : (phase === 'closed' || phase === 'disconnected') && isSelectedTarget
        ? '重新连接'
        : '连接此机器'

  return (
    <div className={`terminal-overlay terminal-overlay-${phase}`}>
      <div className="terminal-overlay-grid">
        <section className="terminal-prompt-card">
          <span className="terminal-prompt-eyebrow">{eyebrow}</span>
          <h3>{title}</h3>
          <p>{description}</p>

          <div className="terminal-usage-tips" aria-label="使用提示">
            <strong>如何使用</strong>
            <div className="terminal-usage-list">
              {usageTips.map((tip, index) => (
                <div className="terminal-usage-item" key={tip}>
                  <span>{index + 1}</span>
                  <em>{tip}</em>
                </div>
              ))}
            </div>
          </div>

          {selectedServer ? (
            <>
              <div className="terminal-inline-meta">
                <span className={`badge ${selectedServer.allowed ? 'live' : 'warning'}`}>
                  {selectedServer.allowed ? '可连接' : '需申请权限'}
                </span>
                <StatusBadge status={selectedServer.status} prefix="状态: " />
                {selectedTagList.length > 0 ? (
                  selectedTagList.map((tag) => (
                    <span className="pill" key={tag}>
                      {tag}
                    </span>
                  ))
                ) : (
                  <span className="pill">无标签</span>
                )}
              </div>

              {selectedServer.allowed ? (
                <div className="terminal-inline-panel">
                  <div className="terminal-inline-panel-header">
                    <div>
                      <strong>登录用户与密钥</strong>
                      <span>在下方选择一个可用配置后直接连接。</span>
                    </div>
                    <button className="ghost small" onClick={onRefreshSSH} disabled={sshLoading}>
                      刷新选项
                    </button>
                  </div>

                  {sshLoading ? (
                    <div className="empty-state">加载登录用户中...</div>
                  ) : sshError ? (
                    <div className="empty-state">{sshError}</div>
                  ) : sshOptions.length === 0 ? (
                    <div className="empty-state">暂无可用登录用户</div>
                  ) : compactSSHPicker ? (
                    <>
                      {fallbackOnly && (
                        <div className="terminal-fallback-note">
                          当前机器未发现托管登录用户，已加载同 Profile{selectedServer.profile ? `（${selectedServer.profile}）` : ''}下已注册的 Key，可尝试登录。
                        </div>
                      )}
                      <label>
                        <span>{fallbackOnly ? '尝试登录配置' : '登录配置'}</span>
                        <select
                          value={buildSSHOptionValue(sshSelected)}
                          onChange={(e) => {
                            const next = sshOptions.find((opt) => buildSSHOptionValue(opt) === e.target.value)
                            if (next) onSelectSSH(next)
                          }}
                        >
                          {sshOptions.map((opt) => (
                            <option key={buildSSHOptionValue(opt)} value={buildSSHOptionValue(opt)}>
                              {opt.user} · {opt.key_name || opt.auth_type}
                              {opt.source && opt.source !== 'profile_fallback' ? ` · ${opt.source}` : ''}
                            </option>
                          ))}
                        </select>
                      </label>
                    </>
                  ) : (
                    <div className="terminal-inline-options">
                      {sshOptions.map((opt) => {
                        const activeOption = isSameSSHOption(sshSelected, opt)
                        return (
                          <label
                            key={`${opt.user}-${opt.key_name || opt.auth_type}`}
                            className={`terminal-ssh-option ${activeOption ? 'active' : ''}`}
                          >
                            <input
                              type="radio"
                              name="terminal-ssh-user"
                              checked={activeOption}
                              onChange={() => onSelectSSH(opt)}
                            />
                            <div className="terminal-ssh-option-body">
                              <strong>{opt.user}</strong>
                              <span>{opt.key_name || opt.auth_type}</span>
                            </div>
                            <em>{opt.source === 'profile_fallback' ? 'PROFILE KEY' : opt.auth_type}</em>
                          </label>
                        )
                      })}
                    </div>
                  )}

                  <div className="terminal-inline-actions">
                    <button className="primary" onClick={onConnect} disabled={!canConnect || phase === 'connecting'}>
                      {connectLabel}
                    </button>
                  </div>
                </div>
              ) : (
                <div className="terminal-inline-panel">
                  <div className="terminal-inline-panel-header">
                    <div>
                      <strong>申请连接权限</strong>
                      <span>当前账号尚未具备该机器的连接权限，可以直接在这里提交申请。</span>
                    </div>
                  </div>

                  <label>
                    <span>申请名称/备注</span>
                    <input
                      value={approvalName}
                      onChange={(e) => onApprovalNameChange(e.target.value)}
                      placeholder="例如：临时巡检/故障处理"
                    />
                  </label>

                  <label>
                    <span>有效期</span>
                    <select value={approvalPeriod} onChange={(e) => onApprovalPeriodChange(e.target.value)}>
                      <option value="1d">1 天</option>
                      <option value="1w">1 周</option>
                      <option value="1m">1 月</option>
                      <option value="1y">1 年</option>
                    </select>
                  </label>

                  {approvalStatus && <div className="status">{approvalStatus}</div>}

                  <div className="terminal-inline-actions">
                    <button className="primary" onClick={onSubmitApproval} disabled={approvalSubmitting}>
                      {approvalSubmitting ? '提交中...' : '提交申请'}
                    </button>
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="terminal-inline-panel">
              <div className="empty-state">从左侧机器列表选择目标，系统会在这里展示连接配置、标签和可执行操作。</div>
            </div>
          )}

          {reason && <div className="terminal-reason">{reason}</div>}
        </section>

        <section className="terminal-visual-card" aria-hidden="true">
          <div className="terminal-visual-stage">
            <div className="terminal-signal">
              <span className="terminal-signal-ring ring-a" />
              <span className="terminal-signal-ring ring-b" />
              <span className="terminal-signal-ring ring-c" />
              <span className="terminal-signal-core" />
              <span className="terminal-signal-dot dot-a" />
              <span className="terminal-signal-dot dot-b" />
              <span className="terminal-signal-dot dot-c" />
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}

export const TerminalPage = () => {
  const token = useAuthStore((s) => s.token)
  const authUser = useAuthStore((s) => s.user)
  const sshRequestRef = useRef(0)
  const [host, setHost] = useState('')
  const [sshUser, setSshUser] = useState('')
  const [sshKey, setSshKey] = useState('')
  const [active, setActive] = useState(false)
  const [sessionId, setSessionId] = useState('')
  const [terminalPhase, setTerminalPhase] = useState<TerminalPhase>('idle')
  const [terminalReason, setTerminalReason] = useState('')
  const [query, setQuery] = useState('')
  const [selectedHost, setSelectedHost] = useState('')
  const [servers, setServers] = useState<ServerItem[]>([])
  const [serverLoading, setServerLoading] = useState(false)
  const [serverError, setServerError] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [sshOptions, setSshOptions] = useState<SSHOption[]>([])
  const [sshLoading, setSshLoading] = useState(false)
  const [sshError, setSshError] = useState('')
  const [sshSelected, setSshSelected] = useState<SSHOption | null>(null)
  const [approvalName, setApprovalName] = useState('')
  const [approvalPeriod, setApprovalPeriod] = useState('1w')
  const [approvalStatus, setApprovalStatus] = useState('')
  const [approvalSubmitting, setApprovalSubmitting] = useState(false)

  useEffect(() => {
    if (!token) {
      setActive(false)
      setTerminalPhase('idle')
      setTerminalReason('')
      setSessionId('')
      setSshOptions([])
      setSshSelected(null)
    }
  }, [token])

  const serverIndex = useMemo(
    () =>
      servers.map((item) => ({
        item,
        search: buildSearchText(item),
      })),
    [servers],
  )

  const filteredServers = useMemo(() => {
    const tokens = tokenizeQuery(query)
    if (tokens.length === 0) return serverIndex.map((entry) => entry.item)
    return serverIndex
      .filter(({ search }) => tokens.every((token) => search.includes(token)))
      .map(({ item }) => item)
  }, [query, serverIndex])

  useEffect(() => {
    if (servers.length === 0) {
      setSelectedHost('')
      return
    }
    setSelectedHost((prev) => {
      if (prev && servers.some((item) => item.host === prev)) return prev
      return servers[0].host
    })
  }, [servers])

  const selectedServer = useMemo(
    () => servers.find((item) => item.host === selectedHost) || null,
    [servers, selectedHost],
  )

  const selectedTagList = useMemo(
    () => extractTagLabels(selectedServer?.tags).filter(Boolean).slice(0, 8),
    [selectedServer],
  )

  const connectTo = useCallback((nextHost: string, nextUser?: string, nextKey?: string) => {
    setSelectedHost(nextHost)
    setHost(nextHost)
    setSshUser(nextUser || '')
    setSshKey(nextKey || '')
    setSessionId('')
    setTerminalReason('')
    setTerminalPhase('connecting')
    setActive(true)
  }, [])

  const handleDisconnect = useCallback(() => {
    setTerminalReason('已主动断开当前会话，可重新连接或切换目标机器。')
    setTerminalPhase('disconnected')
    setActive(false)
  }, [])

  const handleTerminalStateChange = useCallback((event: TerminalStateEvent) => {
    if (event.phase === 'connecting') {
      setTerminalReason('')
      setTerminalPhase('connecting')
      return
    }
    if (event.phase === 'live') {
      setTerminalReason('')
      setTerminalPhase('live')
      return
    }

    if (event.phase === 'closed') {
      setActive(false)
      setTerminalPhase('closed')
      setTerminalReason(event.reason || '远端 shell 已退出。')
      return
    }

    setActive(false)
    setTerminalPhase('disconnected')
    setTerminalReason(event.reason || '终端链路已关闭，可重新连接继续操作。')
  }, [])

  const terminalBadge = useMemo(() => {
    if (terminalPhase === 'connecting') {
      return { className: 'badge connecting', label: 'CONNECTING' }
    }
    if (terminalPhase === 'live') {
      return { className: 'badge live', label: 'LIVE' }
    }
    if (terminalPhase === 'closed') {
      return { className: 'badge closed', label: 'CLOSED' }
    }
    if (terminalPhase === 'disconnected') {
      return { className: 'badge warning', label: 'OFFLINE' }
    }
    return { className: 'badge', label: 'IDLE' }
  }, [terminalPhase])

  const loadSshOptions = useCallback(async (item: ServerItem) => {
    if (!token) return
    const requestId = ++sshRequestRef.current
    setSshLoading(true)
    setSshError('')
    try {
      const res = await apiClient.get<{ items: SSHOption[] }>(
        `/api/v1/servers/${encodeURIComponent(item.host)}/ssh-users`,
      )
      if (requestId !== sshRequestRef.current) return
      const options = res.data.items || []
      setSshOptions(options)
      setSshSelected((prev) => {
        const preferredCurrent = options.find(
          (opt) =>
            item.host === host &&
            opt.user === sshUser &&
            (opt.key_name || '') === (sshKey || ''),
        )
        return preferredCurrent || options.find((opt) => isSameSSHOption(prev, opt)) || options[0] || null
      })
    } catch (err: any) {
      if (requestId !== sshRequestRef.current) return
      setSshOptions([])
      setSshSelected(null)
      setSshError(err?.response?.data || err?.message || '加载失败')
    } finally {
      if (requestId === sshRequestRef.current) {
        setSshLoading(false)
      }
    }
  }, [host, sshKey, sshUser, token])

  useEffect(() => {
    if (!token || !selectedServer || !selectedServer.allowed) {
      sshRequestRef.current += 1
      setSshLoading(false)
      setSshError('')
      setSshOptions([])
      setSshSelected(null)
      return
    }
    void loadSshOptions(selectedServer)
  }, [token, selectedServer?.host, selectedServer?.allowed, loadSshOptions])

  useEffect(() => {
    if (!selectedServer || selectedServer.allowed) {
      setApprovalStatus('')
      return
    }
    setApprovalName(`connect-${selectedServer.host}-${new Date().toISOString().slice(0, 10)}`)
    setApprovalStatus('')
  }, [selectedServer?.host, selectedServer?.allowed])

  const connectSelectedServer = useCallback(() => {
    if (!selectedServer || !sshSelected) return
    connectTo(selectedServer.host, sshSelected.user, sshSelected.key_name)
  }, [connectTo, selectedServer, sshSelected])

  const submitApproval = async () => {
    if (!authUser || !selectedServer || selectedServer.allowed) return
    setApprovalSubmitting(true)
    setApprovalStatus('')
    try {
      const name = approvalName.trim() || `web-access-${new Date().toISOString()}`
      const ids = selectedServer.id ? [selectedServer.id] : []
      const names = [selectedServer.name || selectedServer.host]
      const hosts = [selectedServer.host]
      const serverFilter: Record<string, any> = {}
      if (ids.length > 0) serverFilter.id = ids
      if (names.length > 0) serverFilter.name = names
      if (hosts.length > 0) serverFilter.ip_addr = hosts

      await apiClient.post('/api/v1/approval', {
        users: [authUser],
        applicant: authUser,
        name,
        period: approvalPeriod,
        actions: ['connect'],
        server_filter: serverFilter,
      })
      setApprovalStatus('已提交申请')
    } catch (err: any) {
      setApprovalStatus(err?.response?.data || err?.message || '提交失败')
    } finally {
      setApprovalSubmitting(false)
    }
  }

  const fetchServers = useCallback(async () => {
    if (!token) return
    setServerLoading(true)
    setServerError('')
    try {
      const res = await apiClient.get<{ items: ServerItem[] }>('/api/v1/servers')
      setServers(res.data.items || [])
    } catch (err: any) {
      setServerError(err?.response?.data || err?.message || '加载失败')
    } finally {
      setServerLoading(false)
    }
  }, [token])

  useEffect(() => {
    if (!token) {
      setServers([])
      return
    }
    void fetchServers()
  }, [token, fetchServers])

  return (
    <div className="page console-page">
      <div className={`console-layout ${drawerOpen ? 'drawer-open' : 'drawer-closed'}`}>
        <aside className="console-sidebar">
          <div className="panel">
            <div className="panel-header">
              <div>
                <h3>我的机器</h3>
                <p>先选择机器，再在中间详情区连接或申请权限</p>
              </div>
              <div className="panel-actions">
                <button
                  className="icon-button"
                  onClick={fetchServers}
                  disabled={serverLoading || !token}
                  title="刷新机器列表"
                  aria-label="刷新机器列表"
                >
                  <RefreshIcon />
                </button>
              </div>
            </div>
            <div className="panel-body">
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="搜索名称 / IP / 用户 / 标签，空格分关键词"
              />
              <div className="list machine-list">
                {serverLoading ? (
                  <div className="empty-state">加载中...</div>
                ) : serverError ? (
                  <div className="empty-state">{serverError}</div>
                ) : filteredServers.length === 0 ? (
                  <div className="empty-state">暂无可用机器</div>
                ) : (
                  filteredServers.map((item) => {
                    const isDenied = !item.allowed
                    const isCurrent = item.host === selectedHost
                    const isConnected = item.host === host && terminalPhase === 'live'
                    return (
                      <div
                        key={item.id || item.host}
                        role="button"
                        tabIndex={0}
                        className={`list-item ${isCurrent ? 'active' : ''} ${isDenied ? 'disabled' : ''}`}
                        onClick={() => setSelectedHost(item.host)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            setSelectedHost(item.host)
                          }
                        }}
                      >
                        <div className="list-main">
                          <strong>{item.name || item.host}</strong>
                          <span>
                            {item.host}
                            {item.user ? ` · ${item.user}` : ''}
                          </span>
                        </div>
                        <div className="list-actions">
                          {isDenied ? (
                            <span className="badge warning">NO ACCESS</span>
                          ) : (
                            <StatusBadge status={item.status} />
                          )}
                          {isConnected && <span className="pill">已连接</span>}
                        </div>
                      </div>
                    )
                  })
                )}
              </div>
            </div>
          </div>
        </aside>

        <main className="console-main">
          <div className="terminal-card">
            <div className="terminal-header">
              <div>
                <h2>Terminal</h2>
                <p>
                  {terminalPhase === 'live' && host
                    ? `${host}${sshUser ? ` · ${sshUser}` : ''}`
                    : selectedServer
                      ? selectedServer.allowed
                        ? '连接配置已加载'
                        : '等待权限申请'
                      : '未选择主机'}
                </p>
              </div>
              <div className="terminal-meta">
                {sessionId && <span className="pill">Session: {sessionId}</span>}
                <span className={terminalBadge.className}>{terminalBadge.label}</span>
                {active && (
                  <button className="ghost small" onClick={handleDisconnect}>
                    断开
                  </button>
                )}
              </div>
            </div>
            <div className="terminal-wrap">
              {token ? (
                <div className="terminal-stage">
                  <TerminalView
                    active={active}
                    host={host}
                    user={sshUser || undefined}
                    keyName={sshKey || undefined}
                    token={token}
                    sessionId={sessionId || undefined}
                    onSessionId={(id) => setSessionId(id)}
                    onStateChange={handleTerminalStateChange}
                  />
                  {terminalPhase !== 'live' && (
                    <TerminalWorkspacePrompt
                      phase={terminalPhase}
                      selectedServer={selectedServer}
                      selectedTagList={selectedTagList}
                      sshOptions={sshOptions}
                      sshLoading={sshLoading}
                      sshError={sshError}
                      sshSelected={sshSelected}
                      host={host}
                      reason={terminalReason}
                      approvalName={approvalName}
                      approvalPeriod={approvalPeriod}
                      approvalStatus={approvalStatus}
                      approvalSubmitting={approvalSubmitting}
                      onSelectSSH={setSshSelected}
                      onConnect={connectSelectedServer}
                      onRefreshSSH={() => {
                        if (!selectedServer?.allowed) return
                        void loadSshOptions(selectedServer)
                      }}
                      onApprovalNameChange={setApprovalName}
                      onApprovalPeriodChange={setApprovalPeriod}
                      onSubmitApproval={submitApproval}
                    />
                  )}
                </div>
              ) : (
                <div className="empty">请先登录</div>
              )}
            </div>
          </div>
        </main>

        <aside className={`console-drawer ${drawerOpen ? 'open' : 'closed'}`}>
          <button
            type="button"
            className="drawer-rail"
            onClick={() => setDrawerOpen((prev) => !prev)}
            title={drawerOpen ? '收起文件传输' : '展开文件传输'}
          >
            <span>文件传输</span>
            <em>{drawerOpen ? '点击收起' : '点击展开'}</em>
          </button>
          <div className="drawer-panel">
            <FileTransferPanel
              host={host}
              user={sshUser || undefined}
              token={token}
              connected={terminalPhase === 'live'}
              headerAction={
                <button className="ghost small" onClick={() => setDrawerOpen(false)}>
                  收起
                </button>
              }
            />
          </div>
        </aside>
      </div>
    </div>
  )
}
