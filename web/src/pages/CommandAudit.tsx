import { useEffect, useState } from 'react'
import { apiClient } from '../api/client'

type Command = {
  UUID: string; CreatedAt: string; User: string; Client: string; TargetServer: string
  SSHUser: string; SSHKey: string; KeyFingerprint: string; Command: string
  ExitCode: number; DurationMS: number; TimedOut: boolean; status?: string
}
type Page = { items: Command[]; total: number; has_more: boolean }

export const CommandAudit = ({ rangeHours }: { rangeHours: number }) => {
  const [user, setUser] = useState('')
  const [ip, setIp] = useState('')
  const [query, setQuery] = useState({ user: '', ip: '' })
  const [offset, setOffset] = useState(0)
  const [reload, setReload] = useState(0)
  const [page, setPage] = useState<Page>({ items: [], total: 0, has_more: false })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  // A range change remounts this component, resetting pagination.
  useEffect(() => {
    const controller = new AbortController()
    const load = async () => {
      setBusy(true)
      setError('')
      try {
        const { data } = await apiClient.get<Page>('/api/v1/audit/command', {
          params: { duration: rangeHours, limit: 50, offset, ...query }, signal: controller.signal,
        })
        if (!controller.signal.aborted) setPage({ ...data, items: data.items || [] })
      } catch {
        if (!controller.signal.aborted) {
          setError('命令审计加载失败')
          setPage({ items: [], total: 0, has_more: false })
        }
      } finally {
        if (!controller.signal.aborted) setBusy(false)
      }
    }
    void load()
    return () => controller.abort()
  }, [rangeHours, query, offset, reload])

  return <section className="panel admin-shell-panel">
    <div className="panel-header"><div><h3>SSH 非交互（Agent/CLI）</h3>
      <p>非交互入口也可能由人工脚本使用。未完成记录需核查，不能据此判断命令仍在运行。</p></div>
      <button className="ghost" disabled={busy} onClick={() => setReload((v) => v + 1)}>刷新</button>
    </div>
    <form className="panel-body admin-audit-filters" onSubmit={(e) => {
      e.preventDefault(); setOffset(0); setQuery({ user: user.trim(), ip: ip.trim() })
    }}>
      <label className="admin-audit-field"><span>当前认证用户</span><input value={user} onChange={(e) => setUser(e.target.value)} /></label>
      <label className="admin-audit-field"><span>目标 IP</span><input value={ip} onChange={(e) => setIp(e.target.value)} /></label>
      <button className="primary" disabled={busy}>应用筛选</button>
    </form>
    {error && <div role="alert" className="workspace-state-message">{error}</div>}
    <div className="admin-audit-rows" aria-busy={busy}>
      {!busy && !page.items.length && !error && <div className="empty-state">暂无记录</div>}
      {!busy && page.items.map((row) => <div key={row.UUID} className="admin-audit-row">
        <div className="admin-audit-row-head"><strong>{row.User}</strong><time>{new Date(row.CreatedAt).toLocaleString()}</time></div>
        <div className="admin-audit-row-meta">
          <span>目标 {row.TargetServer} · {row.SSHUser || '—'}</span><span>客户端 {row.Client}</span>
          <span>{row.status === 'running' ? '未完成（可能中断或结果未写入）' : row.status === 'completed' ? '已结束' : '历史记录'}</span>
          {row.status !== 'running' && <span>退出码 {row.ExitCode} · {row.DurationMS} ms{row.TimedOut ? ' · 超时' : ''}</span>}
          <span>凭据 {row.SSHKey || '—'} · 指纹 {row.KeyFingerprint || '—'}</span>
          <span>记录 {row.UUID}</span>
        </div>
        <details><summary>查看命令（可能含敏感参数，仅限审计用途）</summary><pre style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{row.Command}</pre></details>
      </div>)}
    </div>
    <div className="panel-body">
      <button className="ghost" disabled={busy || offset === 0} onClick={() => setOffset(Math.max(0, offset - 50))}>上一页</button>
      <span> 共 {page.total} 条 · 第 {offset / 50 + 1} 页 </span>
      <button className="ghost" disabled={busy || !page.has_more} onClick={() => setOffset(offset + 50)}>下一页</button>
    </div>
  </section>
}
