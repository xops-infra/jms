import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { FileTransferPanel } from '../components/FileTransferPanel'
import { TerminalView, type TerminalStateEvent } from '../components/TerminalView'
import { apiClient } from '../api/client'
import { useAuthStore } from '../store/auth'
import { type SSHOption, type ServerItem, type TerminalPhase, splitTagLabels } from './terminalShared'

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

const summarizeSessionId = (value: string) => {
  const normalized = value.trim()
  if (normalized.length <= 22) return normalized
  return `${normalized.slice(0, 8)}...${normalized.slice(-8)}`
}

export const WorkspacePage = () => {
  const token = useAuthStore((s) => s.token)
  const [searchParams] = useSearchParams()

  useEffect(() => {
    if (window.opener) {
      window.opener = null
    }
  }, [])
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

      // 始终与 API 匹配结果对齐，避免「重新校验」后 sshSelected 已更新但 host/user/key state 仍陈旧，
      // 导致终端 WS 与 upload/init 使用不同密钥（多把 key 同用户时后端会选错 users[0]）。
      setHost(routeHost)
      setSshUser(matchedOption.user)
      setSshKey(matchedOption.key_name || '')

      if (activateAfterValidate) {
        setSessionId(sessionStorage.getItem(sessionStorageKey) || '')
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
  const workspaceDescription = buildWorkspaceDescription(host || routeHost, sshSelected)
  const sessionSummary = sessionId ? summarizeSessionId(sessionId) : ''
  const workspaceStatusTitle =
    terminalPhase === 'live'
      ? '终端在线'
      : terminalPhase === 'connecting'
        ? '终端连接中'
        : terminalPhase === 'closed'
          ? '会话已结束'
          : terminalPhase === 'disconnected'
            ? '终端已断开'
            : '终端未连接'
  const workspaceStatusClass = terminalPhase === 'live' ? 'live' : 'offline'
  const footerMetaSummary = [
    sessionSummary ? `Session: ${sessionSummary}` : '',
    fileTransferStatus,
  ]
    .filter(Boolean)
    .join(' · ')
  const footerTags = selectedTagGroups.primary

  // 终端与文件传输必须与当前 URL + 校验通过的登录项一致（优先 sshSelected，避免 state 漂移）
  const effectiveHost = host || routeHost
  const effectiveSshUser = sshSelected?.user || sshUser || routeUser || undefined
  const effectiveSshKey = sshSelected?.key_name || sshKey || routeKey || undefined

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
            <div className="terminal-header workspace-header">
              <div className="workspace-header-main">
                <div className="workspace-header-copy">
                  <h2>{overlayTitle}</h2>
                </div>
              </div>
              <div className="workspace-header-side">
                <div className="terminal-meta workspace-header-meta">
                  <span className="workspace-status-indicator" title={workspaceStatusTitle} aria-label={workspaceStatusTitle}>
                    <span className={`workspace-status-dot ${workspaceStatusClass}`} aria-hidden="true" />
                  </span>
                  <span className="workspace-header-detail" title={workspaceDescription}>
                    {workspaceDescription}
                  </span>
                </div>
                <div className="workspace-header-actions">
                  <button className="ghost small" onClick={() => { void validateWorkspace(false) }} disabled={sshLoading}>
                    刷新
                  </button>
                  {terminalPhase !== 'live' && (
                    <button className="primary small" onClick={reconnect} disabled={sshLoading || terminalPhase === 'connecting'}>
                      {connectLabel}
                    </button>
                  )}
                  {active && (
                    <button className="ghost small" onClick={handleDisconnect}>
                      断开
                    </button>
                  )}
                </div>
              </div>
            </div>

            <div className="terminal-wrap">
              <div className="terminal-stage">
                <TerminalView
                  active={active}
                  host={effectiveHost}
                  user={effectiveSshUser}
                  keyName={effectiveSshKey}
                  token={token || ''}
                  sessionId={sessionId || undefined}
                  onSessionId={handleSessionId}
                  onStateChange={handleTerminalStateChange}
                />
                {terminalPhase !== 'live' && (
                  <div className="terminal-overlay terminal-overlay-static">
                    <div className="terminal-overlay-grid">
                      <div className="terminal-overlay-bg" aria-hidden="true">
                        <div className="terminal-signal terminal-signal-background">
                          <span className="terminal-signal-ring ring-a" />
                          <span className="terminal-signal-ring ring-b" />
                          <span className="terminal-signal-ring ring-c" />
                          <span className="terminal-signal-core" />
                          <span className="terminal-signal-dot dot-a" />
                          <span className="terminal-signal-dot dot-b" />
                          <span className="terminal-signal-dot dot-c" />
                        </div>
                      </div>
                      <section className="terminal-prompt-card">
                        <span className="terminal-prompt-eyebrow">{overlayEyebrow}</span>
                        <h3>{overlayTitle}</h3>
                        <p>{overlayDescription}</p>

                        <div className="terminal-inline-panel">
                          <div className="terminal-inline-panel-header">
                            <div>
                              <strong>连接快照</strong>
                              <span>当前页签已固定到这组连接参数，文件传输状态会随终端联动。</span>
                            </div>
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
                        </div>

                        {(terminalReason || validationError) && <div className="terminal-reason">{terminalReason || validationError}</div>}
                      </section>
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="workspace-terminal-footer">
              <div className="workspace-terminal-footer-main">
                <span className="workspace-terminal-footer-connection" title={workspaceDescription}>
                  {workspaceDescription}
                </span>
                {footerMetaSummary && (
                  <span className="workspace-terminal-footer-meta">{footerMetaSummary}</span>
                )}
              </div>
              {footerTags.length > 0 && (
                <div className="workspace-terminal-footer-tags">
                  {footerTags.map((tag) => (
                    <span className="pill workspace-terminal-tag" key={tag} title={tag}>
                      {tag}
                    </span>
                  ))}
                </div>
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
              host={effectiveHost}
              user={effectiveSshUser}
              keyName={effectiveSshKey}
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
