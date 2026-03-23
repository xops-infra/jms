import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { FileTransferPanel } from '../components/FileTransferPanel'
import { TerminalView, type TerminalStateEvent } from '../components/TerminalView'
import { apiClient } from '../api/client'
import { useAuthStore } from '../store/auth'
import { type SSHOption, type ServerItem, type TerminalPhase, RefreshIcon, StatusBadge, splitTagLabels } from './terminalShared'

type ValidationState = 'loading' | 'ready' | 'blocked'

const buildSessionStorageKey = (host: string, user: string, keyName: string, authType: string) =>
  `jms_workspace_session:${host}:${user}:${keyName}:${authType}`

const mapWorkspaceError = (err: any) => {
  const status = err?.response?.status
  if (status === 403) return '当前账号已无该机器连接权限，请返回首页重新选择。'
  if (status === 404) return '目标机器不存在，或连接配置已失效，请返回首页重新选择。'
  return err?.response?.data || err?.message || '工作区初始化失败，请返回首页重新选择。'
}

const buildWorkspaceDescription = (host: string, option: SSHOption | null) => {
  const identity = option?.key_name || option?.auth_type || '未指定认证方式'
  return `${host}${option?.user ? ` · ${option.user}` : ''} · ${identity}`
}

export const WorkspacePage = () => {
  const token = useAuthStore((s) => s.token)
  const [searchParams] = useSearchParams()
  const routeHost = searchParams.get('host')?.trim() || ''
  const routeUser = searchParams.get('user')?.trim() || ''
  const routeKey = searchParams.get('key')?.trim() || ''
  const routeAuth = searchParams.get('auth')?.trim() || ''
  const validationRequestRef = useRef(0)
  const [active, setActive] = useState(false)
  const [sessionId, setSessionId] = useState('')
  const [terminalPhase, setTerminalPhase] = useState<TerminalPhase>('idle')
  const [terminalReason, setTerminalReason] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(true)
  const [validationState, setValidationState] = useState<ValidationState>('loading')
  const [validationError, setValidationError] = useState('')
  const [selectedServer, setSelectedServer] = useState<ServerItem | null>(null)
  const [sshSelected, setSshSelected] = useState<SSHOption | null>(null)
  const [sshLoading, setSshLoading] = useState(false)
  const [host, setHost] = useState('')
  const [sshUser, setSshUser] = useState('')
  const [sshKey, setSshKey] = useState('')

  const workspaceLabel = useMemo(
    () => (routeHost ? `${routeHost}${routeUser ? ` · ${routeUser}` : ''} | JMS Workspace` : 'JMS Workspace'),
    [routeHost, routeUser],
  )
  const sessionStorageKey = useMemo(
    () => buildSessionStorageKey(routeHost, routeUser, routeKey, routeAuth),
    [routeAuth, routeHost, routeKey, routeUser],
  )
  const selectedTagGroups = useMemo(
    () => splitTagLabels(selectedServer?.tags),
    [selectedServer],
  )

  useEffect(() => {
    document.title = workspaceLabel
  }, [workspaceLabel])

  useEffect(() => {
    if (!token) {
      setActive(false)
      setValidationState('loading')
    }
  }, [token])

  const validateWorkspace = useCallback(async (activateAfterValidate = false) => {
    if (!token) return
    if (!routeHost || !routeUser || !routeAuth) {
      setValidationState('blocked')
      setValidationError('连接参数不完整，请返回首页重新选择。')
      setActive(false)
      setTerminalPhase('idle')
      setTerminalReason('')
      setSelectedServer(null)
      setSshSelected(null)
      return
    }

    const requestId = ++validationRequestRef.current
    setValidationState('loading')
    setValidationError('')
    setTerminalReason('')
    setSshLoading(true)

    try {
      const [serversResult, sshResult] = await Promise.allSettled([
        apiClient.get<{ items: ServerItem[] }>('/api/v1/servers'),
        apiClient.get<{ items: SSHOption[] }>(`/api/v1/servers/${encodeURIComponent(routeHost)}/ssh-users`),
      ])

      if (requestId !== validationRequestRef.current) return

      if (sshResult.status === 'rejected') {
        throw sshResult.reason
      }

      const serverMatch =
        serversResult.status === 'fulfilled'
          ? (serversResult.value.data.items || []).find((item) => item.host === routeHost) || null
          : null

      const options = sshResult.value.data.items || []
      const matchedOption =
        options.find(
          (item) =>
            item.user === routeUser &&
            (item.key_name || '') === routeKey &&
            item.auth_type === routeAuth,
        ) || null

      if (!matchedOption) {
        setValidationState('blocked')
        setValidationError('所选登录配置已失效，请返回首页重新选择。')
        setSelectedServer(serverMatch)
        setSshSelected(null)
        setActive(false)
        setTerminalPhase('idle')
        return
      }

      const nextServer =
        serverMatch || {
          id: routeHost,
          name: routeHost,
          host: routeHost,
          allowed: true,
        }

      setSelectedServer(nextServer)
      setSshSelected(matchedOption)
      setValidationState('ready')

      if (activateAfterValidate) {
        setSessionId(sessionStorage.getItem(sessionStorageKey) || '')
        setHost(routeHost)
        setSshUser(matchedOption.user)
        setSshKey(matchedOption.key_name || '')
        setTerminalPhase('connecting')
        setActive(true)
      }
    } catch (err: any) {
      if (requestId !== validationRequestRef.current) return
      setValidationState('blocked')
      setValidationError(mapWorkspaceError(err))
      setSelectedServer(null)
      setSshSelected(null)
      setActive(false)
      setTerminalPhase('idle')
    } finally {
      if (requestId === validationRequestRef.current) {
        setSshLoading(false)
      }
    }
  }, [routeAuth, routeHost, routeKey, routeUser, sessionStorageKey, token])

  useEffect(() => {
    void validateWorkspace(true)
  }, [validateWorkspace])

  const handleSessionId = useCallback((id: string) => {
    setSessionId(id)
    sessionStorage.setItem(sessionStorageKey, id)
  }, [sessionStorageKey])

  const reconnect = useCallback(() => {
    if (!selectedServer || !sshSelected) return
    setHost(selectedServer.host)
    setSshUser(sshSelected.user)
    setSshKey(sshSelected.key_name || '')
    setTerminalReason('')
    setTerminalPhase('connecting')
    setActive(true)
  }, [selectedServer, sshSelected])

  const handleDisconnect = useCallback(() => {
    setTerminalReason('已主动断开当前会话，可重新连接或返回首页重新选择。')
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

  const overlayEyebrow = useMemo(() => {
    if (terminalPhase === 'connecting') return 'Linking Session'
    if (terminalPhase === 'closed') return 'Session Complete'
    if (terminalPhase === 'disconnected') return 'Link Interrupted'
    return 'Workspace Ready'
  }, [terminalPhase])

  const overlayTitle = selectedServer?.name || routeHost || 'Workspace'
  const overlayDescription =
    terminalPhase === 'connecting'
      ? '正在建立 SSH 与终端链路，成功后文件传输也会一并启用。'
      : terminalPhase === 'closed'
        ? '当前会话已经结束，可以使用相同配置重新连接。'
        : terminalPhase === 'disconnected'
          ? '连接已断开，工作区会保留当前目标和会话标识，便于直接恢复。'
          : '工作区已锁定为当前机器和登录配置，可以直接恢复连接。'
  const connectLabel = terminalPhase === 'connecting' ? '建立中...' : terminalPhase === 'closed' || terminalPhase === 'disconnected' ? '重新连接' : '连接工作区'
  const fileTransferStatus = terminalPhase === 'live' ? '已启用' : '等待终端在线'

  if (validationState === 'blocked') {
    return (
      <div className="page console-page">
        <div className="workspace-state-shell">
          <div className="panel workspace-state-card">
            <div className="panel-header">
              <div>
                <h3>Workspace Blocked</h3>
                <p>当前 URL 无法建立安全工作区，请返回首页重新选择。</p>
              </div>
            </div>
            <div className="panel-body">
              <div className="empty-state workspace-state-message">{validationError}</div>
              <div className="workspace-state-actions">
                <button className="primary" onClick={() => { window.location.hash = '#/terminal' }}>
                  返回首页重新选择
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (validationState === 'loading' && !selectedServer) {
    return (
      <div className="page console-page">
        <div className="workspace-state-shell">
          <div className="panel workspace-state-card">
            <div className="panel-header">
              <div>
                <h3>准备工作区</h3>
                <p>正在校验机器、登录用户和会话恢复信息。</p>
              </div>
            </div>
            <div className="panel-body">
              <div className="empty-state workspace-state-message">校验中，请稍候...</div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="page console-page">
      <div className={`workspace-layout ${drawerOpen ? 'drawer-open' : 'drawer-closed'}`}>
        <main className="console-main">
          <div className="terminal-card workspace-terminal-card">
            <div className="terminal-header">
              <div className="workspace-header-copy">
                <h2>{overlayTitle}</h2>
                <p>{buildWorkspaceDescription(host || routeHost, sshSelected)}</p>
              </div>
              <div className="terminal-meta workspace-header-meta">
                {selectedServer?.status && <StatusBadge status={selectedServer.status} prefix="状态: " />}
                {selectedTagGroups.primary.map((tag) => (
                  <span className="pill" key={tag}>
                    {tag}
                  </span>
                ))}
                {sessionId && <span className="pill">Session: {sessionId}</span>}
                <span className={terminalBadge.className}>{terminalBadge.label}</span>
                <button
                  className="icon-button"
                  onClick={() => {
                    void validateWorkspace(false)
                  }}
                  disabled={sshLoading}
                  title="重新校验工作区"
                  aria-label="重新校验工作区"
                >
                  <RefreshIcon />
                </button>
                <button className="ghost small" onClick={() => { window.location.hash = '#/terminal' }}>
                  返回首页
                </button>
                {active && (
                  <button className="ghost small" onClick={handleDisconnect}>
                    断开
                  </button>
                )}
              </div>
            </div>

            <div className="terminal-wrap">
              <div className="terminal-stage">
                <TerminalView
                  active={active}
                  host={host}
                  user={sshUser || undefined}
                  keyName={sshKey || undefined}
                  token={token || ''}
                  sessionId={sessionId || undefined}
                  onSessionId={handleSessionId}
                  onStateChange={handleTerminalStateChange}
                />
                {terminalPhase !== 'live' && (
                  <div className="terminal-overlay terminal-overlay-static">
                    <div className="terminal-overlay-grid">
                      <section className="terminal-prompt-card">
                        <span className="terminal-prompt-eyebrow">{overlayEyebrow}</span>
                        <h3>{overlayTitle}</h3>
                        <p>{overlayDescription}</p>

                        <div className="terminal-usage-tips" aria-label="工作区说明">
                          <strong>当前工作区</strong>
                          <div className="terminal-usage-list">
                            <div className="terminal-usage-item">
                              <span>1</span>
                              <em>该页签已固定到当前机器与登录配置，不会切换到其他用户或密钥。</em>
                            </div>
                            <div className="terminal-usage-item">
                              <span>2</span>
                              <em>终端连通后，右侧文件传输面板会同步启用；断开后会立即禁用。</em>
                            </div>
                            <div className="terminal-usage-item">
                              <span>3</span>
                              <em>刷新当前页签会复用本页签保存的会话标识，优先恢复已有 tmux 工作区。</em>
                            </div>
                          </div>
                        </div>

                        <div className="terminal-inline-meta">
                          <span className="badge live">已锁定配置</span>
                          {selectedServer?.status && <StatusBadge status={selectedServer.status} prefix="状态: " />}
                          {sshSelected && (
                            <span className="pill">
                              {sshSelected.user} · {sshSelected.key_name || sshSelected.auth_type}
                            </span>
                          )}
                          {selectedTagGroups.secondary.length > 0 && (
                            <span className="pill terminal-inline-meta-summary">补充标签 {selectedTagGroups.secondary.length}</span>
                          )}
                        </div>

                        <div className="terminal-inline-panel">
                          <div className="terminal-inline-panel-header">
                            <div>
                              <strong>连接快照</strong>
                              <span>工作区不会回退到其他登录配置；如果当前配置失效，请返回首页重新选择。</span>
                            </div>
                            <button className="ghost small" onClick={() => { void validateWorkspace(false) }} disabled={sshLoading}>
                              重新校验
                            </button>
                          </div>

                          <div className="terminal-snapshot">
                            <div className="terminal-snapshot-item">
                              <span>目标机器</span>
                              <strong>{host || routeHost}</strong>
                            </div>
                            <div className="terminal-snapshot-item">
                              <span>登录用户</span>
                              <strong>{sshSelected?.user || routeUser}</strong>
                            </div>
                            <div className="terminal-snapshot-item">
                              <span>认证方式</span>
                              <strong>{sshSelected?.key_name || sshSelected?.auth_type || routeAuth}</strong>
                            </div>
                            <div className="terminal-snapshot-item">
                              <span>文件传输</span>
                              <strong>{fileTransferStatus}</strong>
                            </div>
                          </div>

                          <div className="terminal-inline-actions">
                            <button className="primary" onClick={reconnect} disabled={sshLoading || terminalPhase === 'connecting'}>
                              {connectLabel}
                            </button>
                            <button className="ghost" onClick={() => { window.location.hash = '#/terminal' }}>
                              返回首页重新选择
                            </button>
                          </div>
                        </div>

                        {(terminalReason || validationError) && <div className="terminal-reason">{terminalReason || validationError}</div>}
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
                )}
              </div>
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
