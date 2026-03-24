import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { apiClient } from '../api/client'
import { RefreshIcon } from './terminalShared'

type SSHLoginRecord = {
  ID?: number
  CreatedAt?: string
  user?: string
  client?: string
  target?: string
  target_instance_id?: string
}

type ScpRecord = {
  ID?: number
  CreatedAt?: string
  action?: string
  from?: string
  to?: string
  user?: string
  client?: string
}

type Paged<T> = {
  items: T[]
  total: number
  limit: number
  offset: number
  has_more: boolean
}

type TerminalAuditFile = {
  name: string
  size: number
  mod_time: string
  host?: string
  user?: string
}

type TerminalListResponse = {
  enabled: boolean
  dir?: string
  files?: TerminalAuditFile[]
  /** 旧版 API 可能无此字段 */
  total?: number
  limit?: number
  offset?: number
  has_more?: boolean
}

const normalizeTerminalList = (data: TerminalListResponse, append: boolean, prevTotal: number) => {
  const files = data.files ?? []
  let total: number
  if (typeof data.total === 'number' && !Number.isNaN(data.total)) {
    total = data.total
  } else if (append) {
    total = prevTotal
  } else {
    total = files.length
  }
  if (total < files.length && !append) total = files.length
  const hasMore = typeof data.has_more === 'boolean' ? data.has_more : false
  return { files, total, hasMore }
}

type AuditTab = 'login' | 'scp' | 'terminal'

const RANGE_HOURS = [24, 72, 168, 720] as const
const RANGE_LABELS: Record<(typeof RANGE_HOURS)[number], string> = {
  24: '24 小时',
  72: '3 天',
  168: '7 天',
  720: '30 天',
}

const PAGE_LIMIT = 50

const parseTab = (raw: string | null): AuditTab => {
  if (raw === 'scp' || raw === 'terminal') return raw
  return 'login'
}

const parseRangeHours = (raw: string | null): number => {
  const n = parseInt(raw || '24', 10)
  return RANGE_HOURS.includes(n as (typeof RANGE_HOURS)[number]) ? n : 24
}

const formatDateTime = (value?: string) => {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
}

const formatBytes = (n: number) => {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

export const AdminAuditPage = () => {
  const [searchParams, setSearchParams] = useSearchParams()

  const setParams = useCallback(
    (patch: Record<string, string | undefined | null>, replace: boolean = true) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          for (const [k, v] of Object.entries(patch)) {
            if (v === undefined || v === null || v === '') next.delete(k)
            else next.set(k, v)
          }
          return next
        },
        { replace },
      )
    },
    [setSearchParams],
  )

  const tab = useMemo(() => parseTab(searchParams.get('tab')), [searchParams])
  const rangeHours = useMemo(() => parseRangeHours(searchParams.get('range')), [searchParams])
  const loginUserQ = searchParams.get('user') ?? ''
  const loginIpQ = searchParams.get('ip') ?? ''
  const scpUserQ = searchParams.get('suser') ?? ''
  const scpKwQ = searchParams.get('kw') ?? ''
  const scpActionQ = searchParams.get('action') ?? ''
  const fileQ = searchParams.get('file') ?? ''

  const [loginUserDraft, setLoginUserDraft] = useState(loginUserQ)
  const [loginIpDraft, setLoginIpDraft] = useState(loginIpQ)
  const [scpUserDraft, setScpUserDraft] = useState(scpUserQ)
  const [scpKwDraft, setScpKwDraft] = useState(scpKwQ)

  useEffect(() => {
    setLoginUserDraft(loginUserQ)
  }, [loginUserQ])
  useEffect(() => {
    setLoginIpDraft(loginIpQ)
  }, [loginIpQ])
  useEffect(() => {
    setScpUserDraft(scpUserQ)
  }, [scpUserQ])
  useEffect(() => {
    setScpKwDraft(scpKwQ)
  }, [scpKwQ])

  const [loginRows, setLoginRows] = useState<SSHLoginRecord[]>([])
  const [loginTotal, setLoginTotal] = useState(0)
  const [loginHasMore, setLoginHasMore] = useState(false)
  const [loginLoading, setLoginLoading] = useState(false)
  const [loginLoadingMore, setLoginLoadingMore] = useState(false)
  const [loginError, setLoginError] = useState('')

  const [scpRows, setScpRows] = useState<ScpRecord[]>([])
  const [scpTotal, setScpTotal] = useState(0)
  const [scpHasMore, setScpHasMore] = useState(false)
  const [scpLoading, setScpLoading] = useState(false)
  const [scpLoadingMore, setScpLoadingMore] = useState(false)
  const [scpError, setScpError] = useState('')

  const [termFiles, setTermFiles] = useState<TerminalAuditFile[]>([])
  const [termEnabled, setTermEnabled] = useState(false)
  const [termDir, setTermDir] = useState('')
  const [termTotal, setTermTotal] = useState(0)
  const [termHasMore, setTermHasMore] = useState(false)
  const [termLoading, setTermLoading] = useState(false)
  const [termLoadingMore, setTermLoadingMore] = useState(false)
  const [termError, setTermError] = useState('')

  const [replayBusy, setReplayBusy] = useState(false)
  const [replayHint, setReplayHint] = useState('')

  const loginScrollRef = useRef<HTMLDivElement | null>(null)
  const scpScrollRef = useRef<HTMLDivElement | null>(null)
  const termScrollRef = useRef<HTMLDivElement | null>(null)
  const termTotalRef = useRef(0)
  const loginLoadGuard = useRef(false)
  const scpLoadGuard = useRef(false)
  const termLoadGuard = useRef(false)

  const termContainerRef = useRef<HTMLDivElement | null>(null)
  const termRef = useRef<Terminal | null>(null)
  const replayTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const activeLogName = useMemo(() => {
    if (!termFiles.length) return ''
    if (fileQ && termFiles.some((f) => f.name === fileQ)) return fileQ
    return termFiles[0].name
  }, [termFiles, fileQ])

  useEffect(() => {
    if (tab !== 'terminal' || !termFiles.length) return
    if (!fileQ || !termFiles.some((f) => f.name === fileQ)) {
      setParams({ file: termFiles[0].name })
    }
  }, [tab, termFiles, fileQ, setParams])

  const fetchLogin = useCallback(
    async (offset: number, append: boolean) => {
      if (append) {
        setLoginLoadingMore(true)
      } else {
        setLoginLoading(true)
      }
      setLoginError('')
      try {
        const res = await apiClient.get<Paged<SSHLoginRecord>>('/api/v1/audit/login', {
          params: {
            duration: rangeHours,
            limit: PAGE_LIMIT,
            offset,
            ...(loginUserQ.trim() ? { user: loginUserQ.trim() } : {}),
            ...(loginIpQ.trim() ? { ip: loginIpQ.trim() } : {}),
          },
        })
        const data = res.data
        if (append) {
          setLoginRows((prev) => [...prev, ...(data.items || [])])
        } else {
          setLoginRows(data.items || [])
        }
        setLoginTotal(data.total ?? 0)
        setLoginHasMore(Boolean(data.has_more))
      } catch (err: unknown) {
        const e = err as { response?: { data?: string }; message?: string }
        setLoginError(String(e?.response?.data || e?.message || '加载失败'))
        if (!append) setLoginRows([])
        setLoginHasMore(false)
      } finally {
        setLoginLoading(false)
        setLoginLoadingMore(false)
      }
    },
    [rangeHours, loginUserQ, loginIpQ],
  )

  useEffect(() => {
    if (tab !== 'login') return
    let cancelled = false
    void (async () => {
      if (cancelled) return
      await fetchLogin(0, false)
    })()
    return () => {
      cancelled = true
    }
  }, [tab, rangeHours, loginUserQ, loginIpQ, fetchLogin])

  const fetchScp = useCallback(
    async (offset: number, append: boolean) => {
      if (append) {
        setScpLoadingMore(true)
      } else {
        setScpLoading(true)
      }
      setScpError('')
      try {
        const res = await apiClient.get<Paged<ScpRecord>>('/api/v1/audit/scp', {
          params: {
            duration: rangeHours,
            limit: PAGE_LIMIT,
            offset,
            ...(scpUserQ.trim() ? { user: scpUserQ.trim() } : {}),
            ...(scpKwQ.trim() ? { keyword: scpKwQ.trim() } : {}),
            ...(scpActionQ.trim() ? { action: scpActionQ.trim() } : {}),
          },
        })
        const data = res.data
        if (append) {
          setScpRows((prev) => [...prev, ...(data.items || [])])
        } else {
          setScpRows(data.items || [])
        }
        setScpTotal(data.total ?? 0)
        setScpHasMore(Boolean(data.has_more))
      } catch (err: unknown) {
        const e = err as { response?: { data?: string }; message?: string }
        setScpError(String(e?.response?.data || e?.message || '加载失败'))
        if (!append) setScpRows([])
        setScpHasMore(false)
      } finally {
        setScpLoading(false)
        setScpLoadingMore(false)
      }
    },
    [rangeHours, scpUserQ, scpKwQ, scpActionQ],
  )

  useEffect(() => {
    if (tab !== 'scp') return
    let cancelled = false
    void (async () => {
      if (cancelled) return
      await fetchScp(0, false)
    })()
    return () => {
      cancelled = true
    }
  }, [tab, rangeHours, scpUserQ, scpKwQ, scpActionQ, fetchScp])

  const fetchTerminalPage = useCallback(async (offset: number, append: boolean) => {
    if (append) {
      setTermLoadingMore(true)
    } else {
      setTermLoading(true)
      termTotalRef.current = 0
    }
    setTermError('')
    try {
      const res = await apiClient.get<TerminalListResponse>('/api/v1/audit/terminal', {
        params: { limit: PAGE_LIMIT, offset },
      })
      const data = res.data
      const norm = normalizeTerminalList(data, append, termTotalRef.current)
      termTotalRef.current = norm.total
      setTermTotal(norm.total)
      setTermHasMore(norm.hasMore)
      setTermEnabled(data.enabled)
      setTermDir(data.dir || '')
      if (append) {
        setTermFiles((prev) => [...prev, ...norm.files])
      } else {
        setTermFiles(norm.files)
      }
    } catch (err: unknown) {
      const e = err as { response?: { data?: string }; message?: string }
      setTermError(String(e?.response?.data || e?.message || '加载失败'))
      if (!append) {
        setTermFiles([])
        setTermEnabled(false)
        termTotalRef.current = 0
        setTermTotal(0)
      }
      setTermHasMore(false)
    } finally {
      setTermLoading(false)
      setTermLoadingMore(false)
    }
  }, [])

  useEffect(() => {
    if (tab !== 'terminal') return
    let cancelled = false
    void (async () => {
      if (cancelled) return
      await fetchTerminalPage(0, false)
    })()
    return () => {
      cancelled = true
    }
  }, [tab, fetchTerminalPage])

  const onLoginScroll = useCallback(() => {
    const el = loginScrollRef.current
    if (!el || loginLoading || loginLoadingMore || !loginHasMore || loginLoadGuard.current) return
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 100) {
      loginLoadGuard.current = true
      void fetchLogin(loginRows.length, true).finally(() => {
        loginLoadGuard.current = false
      })
    }
  }, [loginLoading, loginLoadingMore, loginHasMore, loginRows.length, fetchLogin])

  const onScpScroll = useCallback(() => {
    const el = scpScrollRef.current
    if (!el || scpLoading || scpLoadingMore || !scpHasMore || scpLoadGuard.current) return
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 100) {
      scpLoadGuard.current = true
      void fetchScp(scpRows.length, true).finally(() => {
        scpLoadGuard.current = false
      })
    }
  }, [scpLoading, scpLoadingMore, scpHasMore, scpRows.length, fetchScp])

  const onTermScroll = useCallback(() => {
    const el = termScrollRef.current
    if (!el || termLoading || termLoadingMore || !termHasMore || termLoadGuard.current) return
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 80) {
      termLoadGuard.current = true
      void fetchTerminalPage(termFiles.length, true).finally(() => {
        termLoadGuard.current = false
      })
    }
  }, [termLoading, termLoadingMore, termHasMore, termFiles.length, fetchTerminalPage])

  useEffect(() => {
    if (tab !== 'terminal') return
    const el = termContainerRef.current
    if (!el || termRef.current) return
    const term = new Terminal({
      fontFamily: '"Fira Code", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      fontSize: 13,
      theme: {
        background: '#0b1220',
        foreground: '#e2e8f0',
      },
      cursorBlink: false,
      disableStdin: true,
      scrollback: 100000,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(el)
    fit.fit()
    termRef.current = term
    const ro = new ResizeObserver(() => fit.fit())
    ro.observe(el)
    return () => {
      ro.disconnect()
      term.dispose()
      termRef.current = null
    }
  }, [tab])

  const stopReplay = useCallback(() => {
    if (replayTimerRef.current) {
      clearInterval(replayTimerRef.current)
      replayTimerRef.current = null
    }
    setReplayBusy(false)
  }, [])

  useEffect(() => {
    return () => stopReplay()
  }, [stopReplay])

  const loadLogInstant = useCallback(async () => {
    if (!activeLogName) return
    stopReplay()
    setReplayHint('')
    const term = termRef.current
    if (!term) return
    setReplayBusy(true)
    try {
      const res = await apiClient.get<ArrayBuffer>(`/api/v1/audit/terminal/${encodeURIComponent(activeLogName)}`, {
        responseType: 'arraybuffer',
      })
      const text = new TextDecoder('utf-8', { fatal: false }).decode(new Uint8Array(res.data))
      term.reset()
      term.write(text)
      setReplayHint(`已加载 ${formatBytes(res.data.byteLength)}，共 ${text.length} 字符`)
    } catch (err: unknown) {
      const e = err as { response?: { data?: string }; message?: string }
      term.reset()
      term.write(`\r\n\x1b[31m加载失败: ${String(e?.response?.data || e?.message || err)}\x1b[0m\r\n`)
      setReplayHint('')
    } finally {
      setReplayBusy(false)
    }
  }, [activeLogName, stopReplay])

  useEffect(() => {
    if (tab !== 'terminal' || !termEnabled || !activeLogName) return
    const timer = window.setTimeout(() => {
      void loadLogInstant()
    }, 0)
    return () => clearTimeout(timer)
  }, [tab, termEnabled, activeLogName, loadLogInstant])

  const replayGradual = useCallback(async () => {
    if (!activeLogName) return
    stopReplay()
    const term = termRef.current
    if (!term) return
    setReplayBusy(true)
    setReplayHint('正在读取…')
    try {
      const res = await apiClient.get<ArrayBuffer>(`/api/v1/audit/terminal/${encodeURIComponent(activeLogName)}`, {
        responseType: 'arraybuffer',
      })
      const text = new TextDecoder('utf-8', { fatal: false }).decode(new Uint8Array(res.data))
      term.reset()
      const chunkSize = 2048
      let offset = 0
      replayTimerRef.current = setInterval(() => {
        if (offset >= text.length) {
          stopReplay()
          setReplayHint('回放结束')
          return
        }
        const next = text.slice(offset, offset + chunkSize)
        term.write(next)
        offset += chunkSize
      }, 12)
      setReplayHint('渐进回放中…')
    } catch (err: unknown) {
      const e = err as { response?: { data?: string }; message?: string }
      term.reset()
      term.write(`\r\n\x1b[31m加载失败: ${String(e?.response?.data || e?.message || err)}\x1b[0m\r\n`)
      setReplayBusy(false)
      setReplayHint('')
    }
  }, [activeLogName, stopReplay])

  const applyLoginQuery = () => {
    setParams({
      user: loginUserDraft.trim() || null,
      ip: loginIpDraft.trim() || null,
    })
  }

  const applyScpQuery = () => {
    setParams({
      suser: scpUserDraft.trim() || null,
      kw: scpKwDraft.trim() || null,
    })
  }

  return (
    <div className="page console-page admin-shell-page admin-audit-page">
      <section className="admin-shell-hero">
        <div>
          <span className="terminal-prompt-eyebrow">Admin Console</span>
          <h1>审计</h1>
          <p>
            查询 SSH 登录与 SCP/Web 文件传输（分页加载，滑到底自动续载）；终端列表支持分页。URL 参数可保存当前子页与筛选条件。
          </p>
        </div>
        <div className="admin-shell-summary">
          <div className="admin-shell-stat">
            <span>登录</span>
            <strong>
              {loginTotal > 0 ? `${loginRows.length}/${loginTotal}` : loginRows.length}
            </strong>
          </div>
          <div className="admin-shell-stat">
            <span>传输</span>
            <strong>
              {scpTotal > 0 ? `${scpRows.length}/${scpTotal}` : scpRows.length}
            </strong>
          </div>
        </div>
      </section>

      <div className="admin-audit-tabs" role="tablist" aria-label="审计类型">
        {(
          [
            ['login', '登录记录'],
            ['scp', '文件传输'],
            ['terminal', '终端回放'],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={tab === id}
            className={`ghost admin-audit-tab${tab === id ? ' active' : ''}`}
            onClick={() => setParams(id === 'login' ? { tab: null } : { tab: id }, false)}
          >
            {label}
          </button>
        ))}
      </div>

      <div className="admin-audit-range-row">
        <span className="admin-audit-range-label">时间范围</span>
        <div className="admin-audit-chip-group" role="group" aria-label="查询时间范围">
          {RANGE_HOURS.map((h) => (
            <button
              key={h}
              type="button"
              className={`ghost admin-audit-chip${rangeHours === h ? ' active' : ''}`}
              onClick={() => setParams({ range: String(h) })}
            >
              {RANGE_LABELS[h]}
            </button>
          ))}
        </div>
      </div>

      {tab === 'login' && (
        <section className="panel admin-shell-panel">
          <div className="panel-header">
            <div>
              <h3>登录记录</h3>
              <p>record_ssh_login，按创建时间倒序分页。</p>
            </div>
            <button
              type="button"
              className="ghost icon-btn"
              onClick={() => void fetchLogin(0, false)}
              disabled={loginLoading}
              title="刷新"
            >
              <RefreshIcon />
            </button>
          </div>
          <div className="panel-body admin-audit-filters">
            <label className="admin-audit-field">
              <span>用户</span>
              <input value={loginUserDraft} onChange={(e) => setLoginUserDraft(e.target.value)} placeholder="可选" />
            </label>
            <label className="admin-audit-field">
              <span>目标 IP</span>
              <input value={loginIpDraft} onChange={(e) => setLoginIpDraft(e.target.value)} placeholder="匹配 target" />
            </label>
            <button type="button" className="primary" onClick={applyLoginQuery} disabled={loginLoading}>
              应用筛选
            </button>
          </div>
          {loginError && <div className="workspace-state-message admin-audit-error">{loginError}</div>}
          <div
            ref={loginScrollRef}
            className="admin-audit-rows"
            onScroll={onLoginScroll}
          >
            {loginRows.map((row, idx) => (
              <div key={`${row.ID ?? idx}-${row.CreatedAt}`} className="admin-audit-row">
                <div className="admin-audit-row-head">
                  <strong>{row.user || '—'}</strong>
                  <span className="admin-audit-time">{formatDateTime(row.CreatedAt)}</span>
                </div>
                <div className="admin-audit-row-meta">
                  <span>客户端 {row.client || '—'}</span>
                  <span>目标 {row.target || '—'}</span>
                  <span>实例 {row.target_instance_id || '—'}</span>
                </div>
              </div>
            ))}
            {!loginLoading && loginRows.length === 0 && <div className="empty-state">暂无数据</div>}
            {loginLoadingMore && <div className="admin-audit-load-hint">加载更多…</div>}
            {!loginHasMore && loginRows.length > 0 && <div className="admin-audit-load-hint muted">已加载全部</div>}
          </div>
        </section>
      )}

      {tab === 'scp' && (
        <section className="panel admin-shell-panel">
          <div className="panel-header">
            <div>
              <h3>文件传输</h3>
              <p>record_scp，关键字匹配 from 或 to。</p>
            </div>
            <button
              type="button"
              className="ghost icon-btn"
              onClick={() => void fetchScp(0, false)}
              disabled={scpLoading}
              title="刷新"
            >
              <RefreshIcon />
            </button>
          </div>
          <div className="panel-body admin-audit-filters admin-audit-filters-wrap">
            <div className="admin-audit-chip-group" role="group" aria-label="传输动作">
              {(
                [
                  ['', '全部'],
                  ['upload', '上传'],
                  ['download', '下载'],
                ] as const
              ).map(([val, label]) => (
                <button
                  key={val || 'all'}
                  type="button"
                  className={`ghost admin-audit-chip${scpActionQ === val ? ' active' : ''}`}
                  onClick={() => setParams({ action: val || null })}
                >
                  {label}
                </button>
              ))}
            </div>
            <label className="admin-audit-field">
              <span>用户</span>
              <input value={scpUserDraft} onChange={(e) => setScpUserDraft(e.target.value)} placeholder="可选" />
            </label>
            <label className="admin-audit-field admin-audit-field-wide">
              <span>关键字</span>
              <input value={scpKwDraft} onChange={(e) => setScpKwDraft(e.target.value)} placeholder="匹配路径 from / to" />
            </label>
            <button type="button" className="primary" onClick={applyScpQuery} disabled={scpLoading}>
              应用筛选
            </button>
          </div>
          {scpError && <div className="workspace-state-message admin-audit-error">{scpError}</div>}
          <div ref={scpScrollRef} className="admin-audit-rows" onScroll={onScpScroll}>
            {scpRows.map((row, idx) => (
              <div key={`${row.ID ?? idx}-${row.CreatedAt}`} className="admin-audit-row">
                <div className="admin-audit-row-head">
                  <strong>{row.action || '—'}</strong>
                  <span className="admin-audit-time">{formatDateTime(row.CreatedAt)}</span>
                </div>
                <div className="admin-audit-row-meta">
                  <span>用户 {row.user || '—'}</span>
                  <span>客户端 {row.client || '—'}</span>
                </div>
                <div className="admin-audit-row-paths">
                  <div>from: {row.from || '—'}</div>
                  <div>to: {row.to || '—'}</div>
                </div>
              </div>
            ))}
            {!scpLoading && scpRows.length === 0 && <div className="empty-state">暂无数据</div>}
            {scpLoadingMore && <div className="admin-audit-load-hint">加载更多…</div>}
            {!scpHasMore && scpRows.length > 0 && <div className="admin-audit-load-hint muted">已加载全部</div>}
          </div>
        </section>
      )}

      {tab === 'terminal' && (
        <div className="admin-audit-terminal-layout">
          <section className="panel admin-shell-panel">
            <div className="panel-header">
              <div>
                <h3>会话文件</h3>
                <p>
                  {termEnabled
                    ? `目录 ${termDir || '（未配置）'} · 共 ${Number.isFinite(termTotal) ? termTotal : 0} 个`
                    : '未开启 withVideo 或未配置目录时无终端录像文件。'}
                </p>
              </div>
              <button
                type="button"
                className="ghost icon-btn"
                onClick={() => void fetchTerminalPage(0, false)}
                disabled={termLoading}
                title="刷新"
              >
                <RefreshIcon />
              </button>
            </div>
            <div className="panel-body">
              {termError && <div className="workspace-state-message admin-audit-error">{termError}</div>}
              <div ref={termScrollRef} className="admin-audit-file-list" onScroll={onTermScroll}>
                {termFiles.map((f) => (
                  <button
                    key={f.name}
                    type="button"
                    className={`admin-audit-file${activeLogName === f.name ? ' active' : ''}`}
                    onClick={() => setParams({ file: f.name })}
                  >
                    <div className="admin-audit-file-name">{f.name}</div>
                    <div className="admin-audit-file-meta">
                      {f.host && f.user ? `${f.host} · ${f.user}` : '—'} · {formatBytes(f.size)} · {formatDateTime(f.mod_time)}
                    </div>
                  </button>
                ))}
                {!termLoading && termEnabled && termFiles.length === 0 && <div className="empty-state">目录下暂无 .log 文件</div>}
                {termLoadingMore && <div className="admin-audit-load-hint">加载更多…</div>}
                {!termHasMore && termFiles.length > 0 && <div className="admin-audit-load-hint muted">已加载全部</div>}
              </div>
            </div>
          </section>
          <section className="panel admin-shell-panel admin-audit-replay-panel">
            <div className="panel-header">
              <div>
                <h3>回放</h3>
                <p className="admin-audit-replay-desc">
                  原始终端输出（含 ANSI）；选中文件自动加载。大文件可渐进回放。
                </p>
              </div>
            </div>
            <div className="panel-body admin-audit-replay-actions">
              <button
                type="button"
                className="primary small"
                onClick={() => void loadLogInstant()}
                disabled={!activeLogName || replayBusy}
              >
                重新加载
              </button>
              <button
                type="button"
                className="ghost small"
                onClick={() => void replayGradual()}
                disabled={!activeLogName || replayBusy}
              >
                渐进回放
              </button>
              <button type="button" className="ghost small" onClick={stopReplay} disabled={!replayBusy}>
                停止
              </button>
              {replayHint && <span className="admin-audit-replay-hint">{replayHint}</span>}
            </div>
            <div ref={termContainerRef} className="admin-audit-terminal-wrap" />
          </section>
        </div>
      )}
    </div>
  )
}
