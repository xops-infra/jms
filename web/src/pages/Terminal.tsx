import { useCallback, useEffect, useMemo, useState } from 'react'
import { TerminalView, type TerminalStateEvent } from '../components/TerminalView'
import { FileTransferPanel } from '../components/FileTransferPanel'
import { useAuthStore } from '../store/auth'
import { apiClient } from '../api/client'

type ServerItem = {
  id: string
  name: string
  host: string
  user?: string
  status?: string
  tags?: Record<string, unknown> | string[] | string
  allowed: boolean
}

type SSHOption = {
  user: string
  key_name?: string
  auth_type: string
}

type DeniedSelection = {
  id: string
  name: string
  host: string
}

type TerminalPhase = 'idle' | 'connecting' | 'live' | 'closed' | 'disconnected'

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

const extractTagTokens = (tags?: ServerItem['tags']) => {
  if (!tags) return []
  if (typeof tags === 'string') return [tags]
  if (Array.isArray(tags)) {
    const items = tags.map((tag) => String(tag))
    const json = safeJson(tags)
    return json ? [...items, json] : items
  }
  if (typeof tags === 'object') {
    const tokens: string[] = []
    Object.entries(tags).forEach(([key, value]) => {
      tokens.push(key)
      if (value === null || value === undefined) return
      if (Array.isArray(value)) {
        value.forEach((entry) => {
          if (entry === null || entry === undefined) return
          const text = String(entry)
          tokens.push(text, `${key}:${text}`, `${key}=${text}`)
        })
        return
      }
      if (typeof value === 'object') {
        const json = safeJson(value)
        if (json) tokens.push(json)
        return
      }
      const text = String(value)
      tokens.push(text, `${key}:${text}`, `${key}=${text}`)
    })
    const json = safeJson(tags)
    if (json) tokens.push(json)
    return tokens
  }
  return [String(tags)]
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
  host: string
  serverName?: string
  sshUser: string
  sessionId: string
  reason: string
  onReconnect: () => void
  onRefreshMachines: () => void
}

const TerminalWorkspacePrompt = ({
  phase,
  host,
  serverName,
  sshUser,
  sessionId,
  reason,
  onReconnect,
  onRefreshMachines,
}: TerminalWorkspacePromptProps) => {
  const hasTarget = Boolean(host)
  const displayTarget = hasTarget ? `${serverName || host}${sshUser ? ` · ${sshUser}` : ''}` : '尚未选择'

  let eyebrow = 'Secure Workspace'
  let title = '选择一台机器开始本次会话'
  let description = '从左侧机器列表选择目标与登录用户，安全终端会在这里展开。'

  if (phase === 'connecting') {
    eyebrow = 'Linking Session'
    title = '正在建立安全连接'
    description = '权限校验、SSH 握手和终端尺寸同步已开始，通常几秒内即可进入 shell。'
  } else if (phase === 'closed') {
    eyebrow = 'Session Complete'
    title = '当前会话已结束'
    description = '远端 shell 已退出。你可以重新连接，或者切换到其他机器继续处理。'
  } else if (phase === 'disconnected') {
    eyebrow = 'Link Interrupted'
    title = '连接已断开'
    description = '浏览器与后端终端链路已经关闭，可以直接重新连接恢复工作节奏。'
  }

  const steps = [
    {
      label: '选择目标',
      value: hasTarget ? displayTarget : '从左侧机器列表开始',
      state: hasTarget ? 'done' : 'current',
    },
    {
      label: '建立链路',
      value:
        phase === 'connecting'
          ? '正在协商 SSH 与 PTY'
          : phase === 'live'
            ? '链路已建立'
            : phase === 'closed'
              ? '会话已完成'
              : phase === 'disconnected'
                ? '等待恢复连接'
                : '等待发起连接',
      state:
        phase === 'connecting'
          ? 'current'
          : phase === 'live' || phase === 'closed' || phase === 'disconnected'
            ? 'done'
            : 'pending',
    },
    {
      label: '进入工作区',
      value:
        phase === 'closed'
          ? 'Shell 已退出'
          : phase === 'disconnected'
            ? '重连后继续处理'
            : phase === 'connecting'
              ? '准备进入命令行'
              : '等待终端激活',
      state: phase === 'closed' || phase === 'disconnected' ? 'done' : 'pending',
    },
  ]

  return (
    <div className={`terminal-overlay terminal-overlay-${phase}`}>
      <div className="terminal-overlay-grid">
        <section className="terminal-prompt-card">
          <span className="terminal-prompt-eyebrow">{eyebrow}</span>
          <h3>{title}</h3>
          <p>{description}</p>

          <div className="terminal-prompt-actions">
            {hasTarget ? (
              <button className="primary" onClick={onReconnect} disabled={phase === 'connecting'}>
                {phase === 'connecting' ? '建立中...' : '重新连接'}
              </button>
            ) : (
              <button className="ghost" onClick={onRefreshMachines}>
                刷新机器
              </button>
            )}
            {hasTarget && (
              <button className="ghost" onClick={onRefreshMachines}>
                刷新列表
              </button>
            )}
          </div>

          <div className="terminal-mode-pills">
            <span>普通 Shell</span>
            <span>无需 tmux</span>
            <span>即时文件传输</span>
          </div>

          <div className="terminal-snapshot">
            <div className="terminal-snapshot-item">
              <span>目标主机</span>
              <strong>{hasTarget ? serverName || host : '等待选择'}</strong>
            </div>
            <div className="terminal-snapshot-item">
              <span>登录用户</span>
              <strong>{sshUser || '待确认'}</strong>
            </div>
            <div className="terminal-snapshot-item">
              <span>工作模式</span>
              <strong>{phase === 'connecting' ? '连接中' : phase === 'closed' ? '会话结束' : phase === 'disconnected' ? '已断开' : '就绪'}</strong>
            </div>
            <div className="terminal-snapshot-item">
              <span>会话标识</span>
              <strong>{sessionId || '建立后生成'}</strong>
            </div>
          </div>

          {reason && <div className="terminal-reason">{reason}</div>}
        </section>

        <section className="terminal-visual-card" aria-hidden="true">
          <div className="terminal-signal">
            <span className="terminal-signal-ring ring-a" />
            <span className="terminal-signal-ring ring-b" />
            <span className="terminal-signal-ring ring-c" />
            <span className="terminal-signal-core" />
            <span className="terminal-signal-dot dot-a" />
            <span className="terminal-signal-dot dot-b" />
            <span className="terminal-signal-dot dot-c" />
          </div>

          <div className="terminal-wave-bars">
            <span />
            <span />
            <span />
            <span />
            <span />
            <span />
          </div>

          <div className="terminal-step-list">
            {steps.map((step) => (
              <div key={step.label} className={`terminal-step ${step.state}`}>
                <strong>{step.label}</strong>
                <span>{step.value}</span>
              </div>
            ))}
          </div>
        </section>
      </div>
    </div>
  )
}

export const TerminalPage = () => {
  const token = useAuthStore((s) => s.token)
  const authUser = useAuthStore((s) => s.user)
  const [host, setHost] = useState('')
  const [sshUser, setSshUser] = useState('')
  const [sshKey, setSshKey] = useState('')
  const [active, setActive] = useState(false)
  const [sessionId, setSessionId] = useState('')
  const [terminalPhase, setTerminalPhase] = useState<TerminalPhase>('idle')
  const [terminalReason, setTerminalReason] = useState('')
  const [query, setQuery] = useState('')
  const [servers, setServers] = useState<ServerItem[]>([])
  const [serverLoading, setServerLoading] = useState(false)
  const [serverError, setServerError] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [loginOpen, setLoginOpen] = useState(false)
  const [loginHost, setLoginHost] = useState<ServerItem | null>(null)
  const [sshOptions, setSshOptions] = useState<SSHOption[]>([])
  const [sshLoading, setSshLoading] = useState(false)
  const [sshError, setSshError] = useState('')
  const [sshSelected, setSshSelected] = useState<SSHOption | null>(null)
  const [approvalOpen, setApprovalOpen] = useState(false)
  const [approvalName, setApprovalName] = useState('')
  const [approvalPeriod, setApprovalPeriod] = useState('1w')
  const [approvalStatus, setApprovalStatus] = useState('')
  const [approvalSubmitting, setApprovalSubmitting] = useState(false)
  const [deniedSelected, setDeniedSelected] = useState<DeniedSelection[]>([])

  useEffect(() => {
    if (!token) {
      setActive(false)
      setTerminalPhase('idle')
      setTerminalReason('')
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

  const selectedServer = useMemo(
    () => servers.find((item) => item.host === host),
    [servers, host],
  )

  const connectTo = useCallback((nextHost: string, nextUser?: string, nextKey?: string) => {
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

  const reconnectToCurrent = useCallback(() => {
    if (!host || !token) return
    setTerminalReason('')
    setTerminalPhase('connecting')
    setActive(true)
  }, [host, token])

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

  const toggleDenied = (item: ServerItem) => {
    setDeniedSelected((prev) => {
      const exists = prev.some((entry) => entry.host === item.host)
      if (exists) {
        return prev.filter((entry) => entry.host !== item.host)
      }
      return [
        ...prev,
        {
          id: item.id,
          name: item.name || item.host,
          host: item.host,
        },
      ]
    })
  }

  const openLoginFor = async (item: ServerItem) => {
    if (!token) return
    setLoginHost(item)
    setLoginOpen(true)
    setSshLoading(true)
    setSshError('')
    setSshOptions([])
    try {
      const res = await apiClient.get<{ items: SSHOption[] }>(
        `/api/v1/servers/${encodeURIComponent(item.host)}/ssh-users`,
      )
      const options = res.data.items || []
      setSshOptions(options)
      if (options.length > 0) {
        setSshSelected(options[0])
      }
    } catch (err: any) {
      setSshError(err?.response?.data || err?.message || '加载失败')
    } finally {
      setSshLoading(false)
    }
  }

  const submitApproval = async () => {
    if (!authUser || deniedSelected.length === 0) return
    setApprovalSubmitting(true)
    setApprovalStatus('')
    try {
      const name = approvalName.trim() || `web-access-${new Date().toISOString()}`
      const ids = deniedSelected.map((item) => item.id).filter((v) => v)
      const names = deniedSelected.map((item) => item.name).filter((v) => v)
      const hosts = deniedSelected.map((item) => item.host).filter((v) => v)
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
      setDeniedSelected([])
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
                <p>点击机器后选择登录用户并连接</p>
              </div>
              <div className="panel-actions">
                <button
                  className="ghost small"
                  onClick={() => setApprovalOpen(true)}
                  disabled={deniedSelected.length === 0}
                >
                  申请权限
                </button>
                <button className="ghost small" onClick={fetchServers} disabled={serverLoading || !token}>
                  刷新
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
                    const isSelected = deniedSelected.some((entry) => entry.host === item.host)
                    return (
                      <div
                        key={item.id || item.host}
                        role="button"
                        tabIndex={0}
                        className={`list-item ${item.host === host && item.user === (sshUser || undefined) ? 'active' : ''} ${isDenied ? 'disabled' : ''} ${isSelected ? 'selected' : ''}`}
                        onClick={() => {
                          if (isDenied) {
                            toggleDenied(item)
                          } else {
                            void openLoginFor(item)
                          }
                        }}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            if (isDenied) {
                              toggleDenied(item)
                            } else {
                              void openLoginFor(item)
                            }
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
                            <label
                              className="select-pill"
                              onClick={(e) => {
                                e.stopPropagation()
                              }}
                            >
                              <input
                                type="checkbox"
                                checked={isSelected}
                                onChange={() => toggleDenied(item)}
                              />
                              <span>申请</span>
                            </label>
                          ) : (
                            <span className={`badge ${item.status === 'running' ? 'live' : 'idle'}`}>
                              {item.status || 'READY'}
                            </span>
                          )}
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
                <p>{host ? `${host}${sshUser ? ` · ${sshUser}` : ''}` : '未选择主机'}</p>
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
                      host={host}
                      serverName={selectedServer?.name}
                      sshUser={sshUser}
                      sessionId={sessionId}
                      reason={terminalReason}
                      onReconnect={reconnectToCurrent}
                      onRefreshMachines={() => void fetchServers()}
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
              headerAction={
                <button className="ghost small" onClick={() => setDrawerOpen(false)}>
                  收起
                </button>
              }
            />
          </div>
        </aside>
      </div>

      {loginOpen && (
        <div className="modal-backdrop" onClick={() => setLoginOpen(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div>
                <h3>选择登录用户</h3>
                <p>{loginHost ? `${loginHost.name || loginHost.host} · ${loginHost.host}` : ''}</p>
              </div>
              <button className="ghost small" onClick={() => setLoginOpen(false)}>
                关闭
              </button>
            </div>
            <div className="modal-body">
              {sshLoading ? (
                <div className="empty-state">加载中...</div>
              ) : sshError ? (
                <div className="empty-state">{sshError}</div>
              ) : sshOptions.length === 0 ? (
                <div className="empty-state">暂无可用登录用户</div>
              ) : (
                <div className="option-list">
                  {sshOptions.map((opt) => (
                    <label key={`${opt.user}-${opt.key_name || opt.auth_type}`} className="option-item">
                      <div>
                        <strong>{opt.user}</strong>
                        <span>{opt.key_name || opt.auth_type}</span>
                      </div>
                      <input
                        type="radio"
                        name="ssh-user"
                        checked={
                          sshSelected?.user === opt.user &&
                          (sshSelected?.key_name || '') === (opt.key_name || '') &&
                          sshSelected?.auth_type === opt.auth_type
                        }
                        onChange={() => setSshSelected(opt)}
                      />
                    </label>
                  ))}
                </div>
              )}
            </div>
            <div className="modal-actions">
              <button className="ghost" onClick={() => setLoginOpen(false)}>
                取消
              </button>
              <button
                className="primary"
                disabled={!loginHost || !sshSelected || sshLoading}
                onClick={() => {
                  if (!loginHost || !sshSelected) return
                  connectTo(loginHost.host, sshSelected.user, sshSelected.key_name)
                  setLoginOpen(false)
                }}
              >
                连接
              </button>
            </div>
          </div>
        </div>
      )}

      {approvalOpen && (
        <div className="modal-backdrop" onClick={() => setApprovalOpen(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div>
                <h3>申请权限</h3>
                <p>选择的机器将提交连接权限申请</p>
              </div>
              <button className="ghost small" onClick={() => setApprovalOpen(false)}>
                关闭
              </button>
            </div>
            <div className="modal-body">
              <div className="pill-group">
                {deniedSelected.length === 0 ? (
                  <div className="empty-state">暂无选中机器</div>
                ) : (
                  deniedSelected.map((entry) => (
                    <span
                      key={entry.host}
                      className="pill selectable"
                      onClick={() =>
                        toggleDenied({
                          id: entry.id,
                          name: entry.name,
                          host: entry.host,
                          user: '',
                          status: '',
                          allowed: false,
                        })
                      }
                    >
                      {entry.name} · {entry.host} ✕
                    </span>
                  ))
                )}
              </div>
              <label>
                <span>申请名称/备注</span>
                <input
                  value={approvalName}
                  onChange={(e) => setApprovalName(e.target.value)}
                  placeholder="例如：临时巡检/故障处理"
                />
              </label>
              <label>
                <span>有效期</span>
                <select value={approvalPeriod} onChange={(e) => setApprovalPeriod(e.target.value)}>
                  <option value="1d">1 天</option>
                  <option value="1w">1 周</option>
                  <option value="1m">1 月</option>
                  <option value="1y">1 年</option>
                </select>
              </label>
              {approvalStatus && <div className="status">{approvalStatus}</div>}
            </div>
            <div className="modal-actions">
              <button className="ghost" onClick={() => setApprovalOpen(false)}>
                取消
              </button>
              <button
                className="primary"
                disabled={deniedSelected.length === 0 || approvalSubmitting}
                onClick={submitApproval}
              >
                提交申请
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
