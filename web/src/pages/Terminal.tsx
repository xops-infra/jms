import { useCallback, useEffect, useMemo, useState } from 'react'
import { TerminalView } from '../components/TerminalView'
import { FileTransferPanel } from '../components/FileTransferPanel'
import { useAuthStore } from '../store/auth'
import { apiClient } from '../api/client'

type ServerItem = {
  id: string
  name: string
  host: string
  user?: string
  status?: string
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

export const TerminalPage = () => {
  const token = useAuthStore((s) => s.token)
  const authUser = useAuthStore((s) => s.user)
  const [host, setHost] = useState('')
  const [sshUser, setSshUser] = useState('')
  const [sshKey, setSshKey] = useState('')
  const [active, setActive] = useState(false)
  const [sessionId, setSessionId] = useState('')
  const [query, setQuery] = useState('')
  const [servers, setServers] = useState<ServerItem[]>([])
  const [serverLoading, setServerLoading] = useState(false)
  const [serverError, setServerError] = useState('')
  const [drawerPinned, setDrawerPinned] = useState(false)
  const [drawerHover, setDrawerHover] = useState(false)
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

  const drawerOpen = drawerPinned || drawerHover

  useEffect(() => {
    if (!token) setActive(false)
  }, [token])

  const filteredServers = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return servers
    return servers.filter(
      (item) =>
        item.host.toLowerCase().includes(q) ||
        (item.user || '').toLowerCase().includes(q) ||
        (item.name || '').toLowerCase().includes(q),
    )
  }, [query, servers])

  const connectTo = useCallback((nextHost: string, nextUser?: string, nextKey?: string) => {
    setHost(nextHost)
    setSshUser(nextUser || '')
    setSshKey(nextKey || '')
    setSessionId('')
    setActive(true)
  }, [])

  const handleDisconnect = () => {
    setActive(false)
  }

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
                placeholder="搜索名称 / IP / 用户"
              />
              <div className="list">
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
                <span className={`badge ${active ? 'live' : 'idle'}`}>{active ? 'LIVE' : 'IDLE'}</span>
                {active && (
                  <button className="ghost small" onClick={handleDisconnect}>
                    断开
                  </button>
                )}
              </div>
            </div>
            <div className="terminal-wrap">
              {token ? (
                <TerminalView
                  active={active}
                  host={host}
                  user={sshUser || undefined}
                  keyName={sshKey || undefined}
                  token={token}
                  sessionId={sessionId || undefined}
                  onSessionId={(id) => setSessionId(id)}
                />
              ) : (
                <div className="empty">请先登录</div>
              )}
            </div>
          </div>
        </main>

        <aside
          className={`console-drawer ${drawerOpen ? 'open' : 'closed'}`}
          onMouseEnter={() => setDrawerHover(true)}
          onMouseLeave={() => setDrawerHover(false)}
        >
          <button
            type="button"
            className="drawer-rail"
            onClick={() => setDrawerPinned((prev) => !prev)}
            title={drawerPinned ? '取消钉住' : '钉住'}
          >
            <span>文件传输</span>
            <em>{drawerPinned ? '已钉住' : '滑出'}</em>
          </button>
          <div className="drawer-panel">
            <FileTransferPanel
              host={host}
              user={sshUser || undefined}
              token={token}
              headerAction={
                <button className="ghost small" onClick={() => setDrawerPinned((prev) => !prev)}>
                  {drawerPinned ? '取消钉住' : '钉住'}
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
