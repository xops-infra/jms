import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { apiClient } from '../api/client'
import { usePageFullscreen } from '../hooks/usePageFullscreen'
import { useAuthStore } from '../store/auth'
import {
  type SSHOption,
  type ServerItem,
  RefreshIcon,
  StatusBadge,
  buildSSHOptionValue,
  buildSearchText,
  isSameSSHOption,
  splitTagLabels,
  tokenizeQuery,
} from './terminalShared'

const padApprovalPart = (value: number, width = 2) => value.toString().padStart(width, '0')

const formatApprovalTimestamp = (value: Date) =>
  `${value.getFullYear()}${padApprovalPart(value.getMonth() + 1)}${padApprovalPart(value.getDate())}-${padApprovalPart(value.getHours())}${padApprovalPart(value.getMinutes())}${padApprovalPart(value.getSeconds())}${padApprovalPart(value.getMilliseconds(), 3)}`

const buildDefaultApprovalName = (host: string) => `connect-${host}-${formatApprovalTimestamp(new Date())}`

const mapApprovalError = (err: any) => {
  const status = err?.response?.status
  const responseText = typeof err?.response?.data === 'string' ? err.response.data.trim() : ''

  if (responseText.includes('policy already exists') || responseText.includes('同名申请已存在')) {
    return '同名申请已存在，请修改申请名称后重试。'
  }
  if (responseText.includes('null value in column "users"') || responseText.includes('审批单已创建，但策略关联失败')) {
    return '审批单可能已创建，但策略关联失败，请联系管理员处理。'
  }
  if (status === 400) {
    return responseText || '申请参数不完整，请检查后重试。'
  }
  if (responseText && !/SQLSTATE|jms_go_policy|null value in column/i.test(responseText)) {
    return responseText
  }
  return '提交申请失败，请稍后重试或联系管理员。'
}

type HomeSelectionPanelProps = {
  selectedServer: ServerItem | null
  selectedSSHUser: string
  sshOptions: SSHOption[]
  sshLoading: boolean
  sshError: string
  sshSelected: SSHOption | null
  approvalName: string
  approvalPeriod: string
  approvalStatus: string
  launchStatus: string
  onSelectSSH: (option: SSHOption) => void
  onApprovalNameChange: (value: string) => void
  onApprovalPeriodChange: (value: string) => void
}

const HomeSelectionPanel = ({
  selectedServer,
  selectedSSHUser,
  sshOptions,
  sshLoading,
  sshError,
  sshSelected,
  approvalName,
  approvalPeriod,
  approvalStatus,
  launchStatus,
  onSelectSSH,
  onApprovalNameChange,
  onApprovalPeriodChange,
}: HomeSelectionPanelProps) => {
  const fallbackOnly = sshOptions.length > 0 && sshOptions.every((opt) => opt.source === 'profile_fallback')
  const compactSSHPicker = fallbackOnly || sshOptions.length > 4

  let eyebrow = 'Workspace Launchpad'
  let title = '选择一台机器准备独立工作区'
  let description = '首页只负责机器列表和连接信息展示，终端与文件传输会在新的页签中完成。'

  if (selectedServer) {
    const defaultSSHUser = selectedSSHUser || selectedServer.user
    eyebrow = selectedServer.allowed ? 'Launch Workspace' : 'Permission Required'
    title = selectedServer.name || selectedServer.host
    description = `${selectedServer.host}${defaultSSHUser ? ` · 默认用户 ${defaultSSHUser}` : ''}${
      selectedServer.allowed ? ' · 将在新页签自动连接' : ''
    }`
  }

  return (
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
          <span className="terminal-prompt-eyebrow">{eyebrow}</span>
          <h3>{title}</h3>
          <p>{description}</p>

          {selectedServer ? (
            <>
              {selectedServer.allowed ? (
                <div className="terminal-inline-panel">
                  <div className="terminal-inline-panel-header">
                    <div>
                      <strong>登录用户与密钥</strong>
                      <span>确认当前配置后，会以新页签方式自动进入终端与文件传输工作区。</span>
                    </div>
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
                </div>
              )}
            </>
          ) : (
            <div className="terminal-inline-panel">
              <div className="empty-state">从左侧机器列表选择目标，系统会在这里展示连接配置、标签和可执行操作。</div>
            </div>
          )}

          {launchStatus && <div className="terminal-reason">{launchStatus}</div>}
        </section>
      </div>
    </div>
  )
}

const buildWorkspaceUrl = (server: ServerItem, option: SSHOption) => {
  const url = new URL(window.location.href)
  const params = new URLSearchParams({
    host: server.host,
    user: option.user,
    auth: option.auth_type,
  })
  if (option.key_name) {
    params.set('key', option.key_name)
  }
  url.hash = `/workspace?${params.toString()}`
  return url.toString()
}

export const TerminalPage = () => {
  const token = useAuthStore((s) => s.token)
  const authUser = useAuthStore((s) => s.user)
  const sshRequestRef = useRef(0)
  const [query, setQuery] = useState('')
  const [selectedHost, setSelectedHost] = useState('')
  const [servers, setServers] = useState<ServerItem[]>([])
  const [serverLoading, setServerLoading] = useState(false)
  const [serverError, setServerError] = useState('')
  const [sshOptions, setSshOptions] = useState<SSHOption[]>([])
  const [sshLoading, setSshLoading] = useState(false)
  const [sshError, setSshError] = useState('')
  const [sshSelected, setSshSelected] = useState<SSHOption | null>(null)
  const [approvalName, setApprovalName] = useState('')
  const [approvalPeriod, setApprovalPeriod] = useState('1w')
  const [approvalStatus, setApprovalStatus] = useState('')
  const [approvalSubmitting, setApprovalSubmitting] = useState(false)
  const [launchStatus, setLaunchStatus] = useState('')
  const { isPageFullscreen: homePageFullscreen, togglePageFullscreen: toggleHomePageFullscreen } = usePageFullscreen()

  useEffect(() => {
    document.title = 'JMS Web Console'
  }, [])

  useEffect(() => {
    if (!token) {
      sshRequestRef.current += 1
      setServers([])
      setSshOptions([])
      setSshSelected(null)
      setLaunchStatus('')
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

  const selectedTagGroups = useMemo(
    () => splitTagLabels(selectedServer?.tags),
    [selectedServer],
  )
  const canConnect = Boolean(selectedServer?.allowed && sshSelected && !sshLoading)

  const loadSshOptions = useCallback(async (item: ServerItem) => {
    if (!token) return
    const requestId = ++sshRequestRef.current
    setSshLoading(true)
    setSshError('')
    setLaunchStatus('')
    try {
      const res = await apiClient.get<{ items: SSHOption[] }>(
        `/api/v1/servers/${encodeURIComponent(item.host)}/ssh-users`,
      )
      if (requestId !== sshRequestRef.current) return
      const options = res.data.items || []
      setSshOptions(options)
      setSshSelected((prev) => options.find((opt) => isSameSSHOption(prev, opt)) || options[0] || null)
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
  }, [token])

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
    setLaunchStatus('')
  }, [selectedHost, sshSelected])

  useEffect(() => {
    if (!selectedServer || selectedServer.allowed) {
      setApprovalStatus('')
      return
    }
    setApprovalName(buildDefaultApprovalName(selectedServer.host))
    setApprovalStatus('')
  }, [selectedServer?.host, selectedServer?.allowed])

  const openWorkspace = useCallback(() => {
    if (!selectedServer || !sshSelected) return
    // 勿在第三个参数里写 noopener：否则规范要求 window.open 返回 null，即使用户未拦截弹窗也会误判
    const nextWindow = window.open(buildWorkspaceUrl(selectedServer, sshSelected), '_blank')
    if (!nextWindow) {
      setLaunchStatus('浏览器拦截了新页签，请允许弹窗后重试。')
    }
  }, [selectedServer, sshSelected])

  const submitApproval = async () => {
    if (!authUser || !selectedServer || selectedServer.allowed) return
    setApprovalSubmitting(true)
    setApprovalStatus('')
    try {
      const name = approvalName.trim() || buildDefaultApprovalName(selectedServer.host)
      const ids = selectedServer.id ? [selectedServer.id] : []
      const names = [selectedServer.name || selectedServer.host]
      const hosts = [selectedServer.host]
      const serverFilter: Record<string, unknown> = {}
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
      setApprovalStatus(mapApprovalError(err))
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
    if (!token) return
    void fetchServers()
  }, [token, fetchServers])

  return (
    <div className="page console-page">
      <div className={`console-layout terminal-home-layout${homePageFullscreen ? ' is-page-fullscreen' : ''}`}>
        <aside className="console-sidebar">
          <div className="panel">
            <div className="panel-header">
              <div>
                <h3>我的机器</h3>
                <p>先选择机器，再在右侧详情区打开独立工作区</p>
              </div>
              <div className="panel-actions">
                <button
                  type="button"
                  className="ghost small"
                  onClick={() => { toggleHomePageFullscreen() }}
                  title={
                    homePageFullscreen
                      ? '退出全屏 (Shift+Esc)'
                      : '全屏：在当前标签内铺满窗口；Shift+Esc 退出'
                  }
                  aria-pressed={homePageFullscreen}
                >
                  {homePageFullscreen ? '退出全屏' : '全屏'}
                </button>
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
          <div className="terminal-card terminal-home-card">
            <div className="terminal-header">
              <div>
                <h2>Workspace Launchpad</h2>
                <p>首页只保留列表与展示，点击后以新页签方式同时连接多个服务器。</p>
              </div>
              <div className="terminal-meta">
                {selectedServer ? (
                  <>
                    <span className={`badge ${selectedServer.allowed ? 'live' : 'warning'}`}>
                      {selectedServer.allowed ? '可连接' : '需申请权限'}
                    </span>
                    {selectedServer.status && <StatusBadge status={selectedServer.status} prefix="状态: " />}
                    {selectedTagGroups.primary.map((tag) => (
                      <span className="pill" key={tag}>
                        {tag}
                      </span>
                    ))}
                    {selectedServer.allowed && sshSelected && (
                      <span className="pill">
                        {sshSelected.user} · {sshSelected.key_name || sshSelected.auth_type}
                      </span>
                    )}
                    {selectedServer.allowed && (
                      <button
                        className="ghost small"
                        onClick={() => {
                          void loadSshOptions(selectedServer)
                        }}
                        disabled={sshLoading}
                      >
                        刷新选项
                      </button>
                    )}
                    {selectedServer.allowed ? (
                      <button className="primary small" onClick={openWorkspace} disabled={!canConnect}>
                        在新页签连接
                      </button>
                    ) : (
                      <button className="primary small" onClick={submitApproval} disabled={approvalSubmitting}>
                        {approvalSubmitting ? '提交中...' : '提交申请'}
                      </button>
                    )}
                  </>
                ) : (
                  <span className="badge">未选择机器</span>
                )}
              </div>
            </div>
            <div className="terminal-wrap">
              <div className="terminal-stage">
                <HomeSelectionPanel
                  selectedServer={selectedServer}
                  selectedSSHUser={sshSelected?.user || ''}
                  sshOptions={sshOptions}
                  sshLoading={sshLoading}
                  sshError={sshError}
                  sshSelected={sshSelected}
                  approvalName={approvalName}
                  approvalPeriod={approvalPeriod}
                  approvalStatus={approvalStatus}
                  launchStatus={launchStatus}
                  onSelectSSH={setSshSelected}
                  onApprovalNameChange={setApprovalName}
                  onApprovalPeriodChange={setApprovalPeriod}
                />
              </div>
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
