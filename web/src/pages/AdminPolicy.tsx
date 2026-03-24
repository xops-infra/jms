import { useCallback, useEffect, useMemo, useState } from 'react'
import { apiClient } from '../api/client'
import { useAlertStore } from '../store/alert'
import { RefreshIcon } from './terminalShared'

type PolicyAction = 'connect' | 'download' | 'upload' | 'deny_connect' | 'deny_download' | 'deny_upload' | string

type PolicyFilter = {
  id?: string[]
  name?: string[]
  ip_addr?: string[]
  env_type?: string[]
  team?: string[]
  kv?: {
    key?: string
    value?: string
  } | null
}

type PolicyItem = {
  id: string
  name: string
  users: string[]
  actions: PolicyAction[]
  server_filter_v1?: PolicyFilter | null
  server_filter?: PolicyFilter | null
  expires_at: string
  approver: string
  approval_id: string
  is_enabled: boolean
  created_at: string
  updated_at: string
}

type PolicyFormState = {
  id: string
  name: string
  usersText: string
  actions: string[]
  expiresAt: string
  isEnabled: boolean
  approvalId: string
  filterIdText: string
  filterNameText: string
  filterIpText: string
  filterEnvTypeText: string
  filterTeamText: string
  filterKvKey: string
  filterKvValue: string
}

type PolicyMutationRequest = {
  name: string
  users: string[]
  actions: string[]
  server_filter: PolicyFilter
  expires_at: string
  is_enabled: boolean
  approval_id: string
}

const actionOptions = [
  { value: 'connect', label: 'Connect', hint: '允许连接主机' },
  { value: 'download', label: 'Download', hint: '允许下载文件' },
  { value: 'upload', label: 'Upload', hint: '允许上传文件' },
  { value: 'deny_connect', label: 'Deny Connect', hint: '显式拒绝连接' },
  { value: 'deny_download', label: 'Deny Download', hint: '显式拒绝下载' },
  { value: 'deny_upload', label: 'Deny Upload', hint: '显式拒绝上传' },
] as const

const getFilterOfPolicy = (policy?: PolicyItem | null) => policy?.server_filter_v1 || policy?.server_filter || null

const formatDateTime = (value?: string) => {
  if (!value) return '未记录'
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

const toLocalDateTimeInputValue = (value?: string | Date) => {
  const date = value instanceof Date ? value : new Date(value || '')
  if (Number.isNaN(date.getTime())) return ''
  const offset = date.getTimezoneOffset()
  return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16)
}

const createDefaultExpiresAtValue = () => {
  const date = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000)
  date.setSeconds(0, 0)
  return toLocalDateTimeInputValue(date)
}

const parseListInput = (value: string) =>
  Array.from(
    new Set(
      value
        .split(/[\n,，]+/)
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  )

const buildFilterLabels = (filters?: PolicyFilter | null) => {
  const labels: string[] = []
  const push = (prefix: string, values?: string[]) => {
    ;(values || []).filter(Boolean).forEach((value) => labels.push(`${prefix}:${value}`))
  }
  push('IP', filters?.ip_addr)
  push('Name', filters?.name)
  push('Team', filters?.team)
  push('Env', filters?.env_type)
  push('ID', filters?.id)
  if (filters?.kv?.key && filters.kv.value) {
    labels.push(`Tag:${filters.kv.key}=${filters.kv.value}`)
  }
  return labels
}

const buildFilterFromForm = (form: PolicyFormState): PolicyFilter => {
  const filter: PolicyFilter = {}
  const id = parseListInput(form.filterIdText)
  const name = parseListInput(form.filterNameText)
  const ipAddr = parseListInput(form.filterIpText)
  const envType = parseListInput(form.filterEnvTypeText)
  const team = parseListInput(form.filterTeamText)
  const kvKey = form.filterKvKey.trim()
  const kvValue = form.filterKvValue.trim()

  if (id.length > 0) filter.id = id
  if (name.length > 0) filter.name = name
  if (ipAddr.length > 0) filter.ip_addr = ipAddr
  if (envType.length > 0) filter.env_type = envType
  if (team.length > 0) filter.team = team
  if (kvKey && kvValue) {
    filter.kv = {
      key: kvKey,
      value: kvValue,
    }
  }

  return filter
}

const hasPolicyFilter = (filter: PolicyFilter) =>
  Boolean(
    filter.id?.length ||
      filter.name?.length ||
      filter.ip_addr?.length ||
      filter.env_type?.length ||
      filter.team?.length ||
      filter.kv,
  )

const buildPolicySearchText = (policy: PolicyItem) =>
  [
    policy.name,
    policy.id,
    policy.approver,
    policy.approval_id,
    ...policy.users,
    ...policy.actions,
    ...buildFilterLabels(getFilterOfPolicy(policy)),
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()

const isPolicyExpired = (policy?: Pick<PolicyItem, 'expires_at'> | null) => {
  if (!policy?.expires_at) return false
  const expiresAt = new Date(policy.expires_at)
  if (Number.isNaN(expiresAt.getTime())) return false
  return expiresAt.getTime() <= Date.now()
}

const policyStatusMeta = (policy?: PolicyItem | null) => {
  if (!policy) return { className: 'badge idle', label: 'DRAFT' }
  if (isPolicyExpired(policy)) return { className: 'badge closed', label: 'EXPIRED' }
  if (policy.is_enabled) return { className: 'badge live', label: 'ENABLED' }
  if (policy.approval_id) return { className: 'badge connecting', label: 'PENDING APPROVAL' }
  return { className: 'badge warning', label: 'DISABLED' }
}

const comparePolicyUpdatedAt = (left: PolicyItem, right: PolicyItem) => {
  const leftTime = new Date(left.updated_at || left.created_at).getTime()
  const rightTime = new Date(right.updated_at || right.created_at).getTime()
  if (Number.isNaN(leftTime) && Number.isNaN(rightTime)) return left.name.localeCompare(right.name)
  if (Number.isNaN(leftTime)) return 1
  if (Number.isNaN(rightTime)) return -1
  return rightTime - leftTime
}

const createEmptyForm = (): PolicyFormState => ({
  id: '',
  name: '',
  usersText: '',
  actions: ['connect'],
  expiresAt: createDefaultExpiresAtValue(),
  isEnabled: false,
  approvalId: '',
  filterIdText: '',
  filterNameText: '',
  filterIpText: '',
  filterEnvTypeText: '',
  filterTeamText: '',
  filterKvKey: '',
  filterKvValue: '',
})

const formFromPolicy = (policy: PolicyItem): PolicyFormState => {
  const filter = getFilterOfPolicy(policy)
  return {
    id: policy.id,
    name: policy.name,
    usersText: (policy.users || []).join(', '),
    actions: (policy.actions || []).filter(Boolean),
    expiresAt: toLocalDateTimeInputValue(policy.expires_at),
    isEnabled: Boolean(policy.is_enabled),
    approvalId: policy.approval_id || '',
    filterIdText: (filter?.id || []).join(', '),
    filterNameText: (filter?.name || []).join(', '),
    filterIpText: (filter?.ip_addr || []).join(', '),
    filterEnvTypeText: (filter?.env_type || []).join(', '),
    filterTeamText: (filter?.team || []).join(', '),
    filterKvKey: filter?.kv?.key || '',
    filterKvValue: filter?.kv?.value || '',
  }
}

const buildPolicyPayload = (form: PolicyFormState): PolicyMutationRequest => {
  const name = form.name.trim()
  const users = parseListInput(form.usersText)
  const actions = Array.from(new Set(form.actions.map((item) => item.trim()).filter(Boolean)))
  const filter = buildFilterFromForm(form)
  const expiresAt = new Date(form.expiresAt)

  if (!name) throw new Error('策略名称不能为空')
  if (users.length === 0) throw new Error('至少填写一个用户，支持逗号分隔和 * 通配')
  if (actions.length === 0) throw new Error('至少选择一个动作')
  if (!hasPolicyFilter(filter)) throw new Error('至少填写一个服务器过滤条件')
  if (Number.isNaN(expiresAt.getTime())) throw new Error('有效期时间格式不正确')

  return {
    name,
    users,
    actions,
    server_filter: filter,
    expires_at: expiresAt.toISOString(),
    is_enabled: form.isEnabled,
    approval_id: form.approvalId.trim(),
  }
}

const extractPolicyId = (value: unknown) => {
  if (typeof value === 'string') return value
  if (value && typeof value === 'object' && 'id' in value) {
    const policyId = (value as { id?: unknown }).id
    return typeof policyId === 'string' ? policyId : ''
  }
  return ''
}

export const AdminPolicyPage = () => {
  const showConfirm = useAlertStore((s) => s.showConfirm)
  const [policies, setPolicies] = useState<PolicyItem[]>([])
  const [selectedPolicyId, setSelectedPolicyId] = useState('')
  const [editorMode, setEditorMode] = useState<'create' | 'edit'>('create')
  const [form, setForm] = useState<PolicyFormState>(() => createEmptyForm())
  const [query, setQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [actionFilter, setActionFilter] = useState('all')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [formError, setFormError] = useState('')
  const [formMessage, setFormMessage] = useState('')

  const loadPolicies = useCallback(async (preferredId?: string) => {
    setLoading(true)
    setLoadError('')
    try {
      const res = await apiClient.get<PolicyItem[]>('/api/v1/policy')
      const nextPolicies = [...(res.data || [])].sort(comparePolicyUpdatedAt)
      setPolicies(nextPolicies)
      setSelectedPolicyId((current) => {
        if (preferredId && nextPolicies.some((policy) => policy.id === preferredId)) return preferredId
        if (current && nextPolicies.some((policy) => policy.id === current)) return current
        return nextPolicies[0]?.id || ''
      })
    } catch (err: any) {
      setLoadError(err?.response?.data || err?.message || '加载 Policy 列表失败')
      setPolicies([])
      setSelectedPolicyId('')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadPolicies()
  }, [loadPolicies])

  const selectedPolicy = useMemo(
    () => policies.find((policy) => policy.id === selectedPolicyId) || null,
    [policies, selectedPolicyId],
  )

  useEffect(() => {
    if (!selectedPolicy) {
      if (policies.length === 0) {
        setEditorMode('create')
        setForm(createEmptyForm())
      }
      return
    }
    setEditorMode('edit')
    setForm(formFromPolicy(selectedPolicy))
    setFormError('')
  }, [policies.length, selectedPolicy])

  const filteredPolicies = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase()
    return policies.filter((policy) => {
      const status = policyStatusMeta(policy).label
      if (statusFilter !== 'all' && status !== statusFilter) return false
      if (actionFilter !== 'all' && !(policy.actions || []).includes(actionFilter)) return false
      if (!normalizedQuery) return true
      return buildPolicySearchText(policy).includes(normalizedQuery)
    })
  }, [actionFilter, policies, query, statusFilter])

  const summary = useMemo(() => {
    const base = {
      total: policies.length,
      enabled: 0,
      pending: 0,
      expired: 0,
      disabled: 0,
    }
    policies.forEach((policy) => {
      if (isPolicyExpired(policy)) {
        base.expired += 1
        return
      }
      if (policy.is_enabled) {
        base.enabled += 1
        return
      }
      if (policy.approval_id) {
        base.pending += 1
        return
      }
      base.disabled += 1
    })
    return base
  }, [policies])

  const draftFilterLabels = useMemo(() => buildFilterLabels(buildFilterFromForm(form)), [form])

  const handleStartCreate = useCallback(() => {
    setSelectedPolicyId('')
    setEditorMode('create')
    setForm(createEmptyForm())
    setFormError('')
    setFormMessage('')
  }, [])

  const handleReset = useCallback(() => {
    if (selectedPolicy) {
      setForm(formFromPolicy(selectedPolicy))
      setEditorMode('edit')
      setFormError('')
      setFormMessage('')
      return
    }
    handleStartCreate()
  }, [handleStartCreate, selectedPolicy])

  const handleActionToggle = useCallback((action: string) => {
    setForm((current) => {
      const exists = current.actions.includes(action)
      return {
        ...current,
        actions: exists ? current.actions.filter((item) => item !== action) : [...current.actions, action],
      }
    })
  }, [])

  const handleSave = useCallback(() => {
    const run = async () => {
      setSaving(true)
      setFormError('')
      setFormMessage('')
      try {
        const payload = buildPolicyPayload(form)
        if (editorMode === 'create') {
          const res = await apiClient.post('/api/v1/policy', payload)
          const policyId = extractPolicyId(res.data)
          await loadPolicies(policyId || undefined)
          setFormMessage(`已创建策略 ${payload.name}`)
          return
        }

        await apiClient.put(`/api/v1/policy/${encodeURIComponent(form.id)}`, payload)
        await loadPolicies(form.id)
        setFormMessage(`已更新策略 ${payload.name}`)
      } catch (err: any) {
        setFormError(err?.response?.data || err?.message || '保存 Policy 失败')
      } finally {
        setSaving(false)
      }
    }

    void run()
  }, [editorMode, form, loadPolicies])

  const handleDelete = useCallback(() => {
    const run = async () => {
      if (!selectedPolicy) return
      const accepted = await showConfirm({
        title: '删除策略',
        message: `确认删除策略 ${selectedPolicy.name} 吗？删除后将从管理列表中隐藏。`,
        tone: 'danger',
        confirmText: '删除',
        cancelText: '取消',
      })
      if (!accepted) return

      setSaving(true)
      setFormError('')
      setFormMessage('')
      try {
        await apiClient.delete(`/api/v1/policy/${encodeURIComponent(selectedPolicy.id)}`)
        await loadPolicies()
        setFormMessage(`已删除策略 ${selectedPolicy.name}`)
      } catch (err: any) {
        setFormError(err?.response?.data || err?.message || '删除 Policy 失败')
      } finally {
        setSaving(false)
      }
    }

    void run()
  }, [loadPolicies, selectedPolicy, showConfirm])

  const editorStatus = editorMode === 'create' ? { className: 'badge idle', label: 'DRAFT' } : policyStatusMeta(selectedPolicy)

  return (
    <div className="page console-page admin-policy-page">
      <section className="admin-policy-hero">
        <div>
          <span className="terminal-prompt-eyebrow">Admin Console</span>
          <h1>Policy 管理页</h1>
          <p>集中管理访问策略，支持按状态和动作过滤，并对用户、动作、到期时间、服务器过滤条件做增删改查。</p>
        </div>
        <div className="admin-policy-summary">
          <div className="admin-policy-stat">
            <span>策略总数</span>
            <strong>{summary.total}</strong>
          </div>
          <div className="admin-policy-stat">
            <span>已启用</span>
            <strong>{summary.enabled}</strong>
          </div>
          <div className="admin-policy-stat">
            <span>待审批</span>
            <strong>{summary.pending}</strong>
          </div>
          <div className="admin-policy-stat">
            <span>已过期</span>
            <strong>{summary.expired}</strong>
          </div>
        </div>
      </section>

      <div className="admin-policy-layout">
        <section className="panel admin-policy-panel">
          <div className="panel-header">
            <div>
              <h3>策略列表</h3>
              <p>按名称、用户、动作、审批状态和过滤条件快速筛选。</p>
            </div>
            <div className="panel-actions">
              <button className="ghost small" onClick={handleStartCreate}>
                新建策略
              </button>
              <button
                className="icon-button"
                onClick={() => {
                  void loadPolicies()
                }}
                disabled={loading}
                title="刷新策略"
                aria-label="刷新策略"
              >
                <RefreshIcon />
              </button>
            </div>
          </div>

          <div className="panel-body">
            <div className="admin-policy-controls">
              <label>
                <span>搜索</span>
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="策略名 / 用户 / 动作 / IP / 审批人"
                />
              </label>
              <label>
                <span>状态</span>
                <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
                  <option value="all">全部状态</option>
                  <option value="ENABLED">已启用</option>
                  <option value="DISABLED">已禁用</option>
                  <option value="PENDING APPROVAL">待审批</option>
                  <option value="EXPIRED">已过期</option>
                </select>
              </label>
              <label>
                <span>动作</span>
                <select value={actionFilter} onChange={(event) => setActionFilter(event.target.value)}>
                  <option value="all">全部动作</option>
                  {actionOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </label>
            </div>

            {loadError && <div className="error">{loadError}</div>}

            <div className="admin-policy-list">
              {loading ? (
                <div className="empty-state">正在加载 Policy...</div>
              ) : filteredPolicies.length === 0 ? (
                <div className="empty-state">没有匹配的策略</div>
              ) : (
                filteredPolicies.map((policy) => {
                  const status = policyStatusMeta(policy)
                  const filterLabels = buildFilterLabels(getFilterOfPolicy(policy))
                  return (
                    <button
                      type="button"
                      key={policy.id}
                      className={`admin-policy-card ${policy.id === selectedPolicyId ? 'active' : ''}`}
                      onClick={() => setSelectedPolicyId(policy.id)}
                    >
                      <div className="admin-policy-card-head">
                        <strong>{policy.name}</strong>
                        <span className={status.className}>{status.label}</span>
                      </div>
                      <div className="admin-policy-card-meta">
                        <span>{policy.users.length > 0 ? `${policy.users.length} 个用户` : '未配置用户'}</span>
                        <span>{policy.approver ? `审批人 ${policy.approver}` : '未审批'}</span>
                        <span>到期 {formatDateTime(policy.expires_at)}</span>
                      </div>
                      <div className="admin-policy-chip-row">
                        {policy.actions.slice(0, 3).map((action) => (
                          <span key={action} className="pill">
                            {action}
                          </span>
                        ))}
                        {filterLabels.slice(0, 2).map((label) => (
                          <span key={label} className="pill">
                            {label}
                          </span>
                        ))}
                        {filterLabels.length > 2 && <span className="pill">+{filterLabels.length - 2} 条过滤</span>}
                      </div>
                    </button>
                  )
                })
              )}
            </div>
          </div>
        </section>

        <section className="admin-policy-detail">
          <div className="panel admin-policy-panel">
            <div className="panel-header">
              <div>
                <h3>{editorMode === 'create' ? '新建策略' : form.name || '编辑策略'}</h3>
                <p>{editorMode === 'create' ? '填写完整策略信息后创建。' : `Policy ID: ${form.id || '未生成'}`}</p>
              </div>
              <div className="admin-policy-detail-actions">
                <span className={editorStatus.className}>{editorStatus.label}</span>
                <button className="ghost small" onClick={handleReset} disabled={saving}>
                  恢复
                </button>
                {editorMode === 'edit' && (
                  <button className="ghost small danger" onClick={handleDelete} disabled={saving}>
                    删除
                  </button>
                )}
                <button className="primary small" onClick={handleSave} disabled={saving}>
                  {editorMode === 'create' ? '创建策略' : '保存修改'}
                </button>
              </div>
            </div>

            <div className="panel-body">
              {formError && <div className="error">{formError}</div>}
              {formMessage && <div className="status">{formMessage}</div>}

              <div className="admin-policy-meta-grid">
                <div className="admin-policy-meta-card">
                  <span>创建时间</span>
                  <strong>{selectedPolicy ? formatDateTime(selectedPolicy.created_at) : '创建后生成'}</strong>
                </div>
                <div className="admin-policy-meta-card">
                  <span>最近更新</span>
                  <strong>{selectedPolicy ? formatDateTime(selectedPolicy.updated_at) : '尚未保存'}</strong>
                </div>
                <div className="admin-policy-meta-card">
                  <span>审批人</span>
                  <strong>{selectedPolicy?.approver || '未设置'}</strong>
                </div>
                <div className="admin-policy-meta-card">
                  <span>过滤条件</span>
                  <strong>{draftFilterLabels.length > 0 ? `${draftFilterLabels.length} 条` : '未配置'}</strong>
                </div>
              </div>

              <div className="admin-policy-form-grid">
                <label>
                  <span>策略名称</span>
                  <input
                    value={form.name}
                    onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                    placeholder="例如 prod-shell-only"
                  />
                </label>
                <label>
                  <span>到期时间</span>
                  <input
                    type="datetime-local"
                    value={form.expiresAt}
                    onChange={(event) => setForm((current) => ({ ...current, expiresAt: event.target.value }))}
                  />
                </label>
                <label>
                  <span>审批单 ID</span>
                  <input
                    value={form.approvalId}
                    onChange={(event) => setForm((current) => ({ ...current, approvalId: event.target.value }))}
                    placeholder="可为空"
                  />
                </label>
                <label className="admin-policy-toggle">
                  <span>启用状态</span>
                  <span className="admin-policy-toggle-row">
                    <input
                      type="checkbox"
                      checked={form.isEnabled}
                      onChange={(event) => setForm((current) => ({ ...current, isEnabled: event.target.checked }))}
                    />
                    <em>{form.isEnabled ? '保存后立即启用' : '保存为禁用状态'}</em>
                  </span>
                </label>
              </div>

              <div className="admin-policy-section">
                <div className="admin-policy-section-header">
                  <strong>授权用户</strong>
                  <span>支持逗号分隔、换行和 `*` 通配</span>
                </div>
                <label>
                  <span>Users</span>
                  <input
                    value={form.usersText}
                    onChange={(event) => setForm((current) => ({ ...current, usersText: event.target.value }))}
                    placeholder="alice, bob, *"
                  />
                </label>
                <div className="admin-policy-chip-row">
                  {parseListInput(form.usersText).map((user) => (
                    <span key={user} className="pill">
                      {user}
                    </span>
                  ))}
                </div>
              </div>

              <div className="admin-policy-section">
                <div className="admin-policy-section-header">
                  <strong>动作授权</strong>
                  <span>允许和显式拒绝都支持配置</span>
                </div>
                <div className="admin-policy-action-grid">
                  {actionOptions.map((option) => {
                    const active = form.actions.includes(option.value)
                    return (
                      <button
                        type="button"
                        key={option.value}
                        className={`admin-policy-action-card${active ? ' active' : ''}`}
                        onClick={() => handleActionToggle(option.value)}
                      >
                        <strong>{option.label}</strong>
                        <span>{option.hint}</span>
                      </button>
                    )
                  })}
                </div>
              </div>

              <div className="admin-policy-section">
                <div className="admin-policy-section-header">
                  <strong>服务器过滤条件</strong>
                  <span>任一维度可留空，但至少需要配置一条过滤规则</span>
                </div>
                <div className="admin-policy-form-grid admin-policy-filter-grid">
                  <label>
                    <span>机器 ID</span>
                    <input
                      value={form.filterIdText}
                      onChange={(event) => setForm((current) => ({ ...current, filterIdText: event.target.value }))}
                      placeholder="srv-01, srv-02"
                    />
                  </label>
                  <label>
                    <span>机器名</span>
                    <input
                      value={form.filterNameText}
                      onChange={(event) => setForm((current) => ({ ...current, filterNameText: event.target.value }))}
                      placeholder="db-prod-*"
                    />
                  </label>
                  <label>
                    <span>IP 地址</span>
                    <input
                      value={form.filterIpText}
                      onChange={(event) => setForm((current) => ({ ...current, filterIpText: event.target.value }))}
                      placeholder="10.0.0.1, 10.0.0.*"
                    />
                  </label>
                  <label>
                    <span>EnvType</span>
                    <input
                      value={form.filterEnvTypeText}
                      onChange={(event) => setForm((current) => ({ ...current, filterEnvTypeText: event.target.value }))}
                      placeholder="prod, stage"
                    />
                  </label>
                  <label>
                    <span>Team</span>
                    <input
                      value={form.filterTeamText}
                      onChange={(event) => setForm((current) => ({ ...current, filterTeamText: event.target.value }))}
                      placeholder="ops, sre"
                    />
                  </label>
                  <label>
                    <span>标签 Key</span>
                    <input
                      value={form.filterKvKey}
                      onChange={(event) => setForm((current) => ({ ...current, filterKvKey: event.target.value }))}
                      placeholder="project"
                    />
                  </label>
                  <label>
                    <span>标签 Value</span>
                    <input
                      value={form.filterKvValue}
                      onChange={(event) => setForm((current) => ({ ...current, filterKvValue: event.target.value }))}
                      placeholder="jms"
                    />
                  </label>
                </div>
                <div className="admin-policy-chip-row">
                  {draftFilterLabels.length > 0 ? (
                    draftFilterLabels.map((label) => (
                      <span key={label} className="pill">
                        {label}
                      </span>
                    ))
                  ) : (
                    <span className="muted">当前还没有有效的过滤条件</span>
                  )}
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}
