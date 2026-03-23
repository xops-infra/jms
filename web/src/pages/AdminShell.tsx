import { useCallback, useEffect, useMemo, useState } from 'react'
import { apiClient } from '../api/client'
import { RefreshIcon } from './terminalShared'

type ShellTaskStatus = 'Pending' | 'Running' | 'Success' | 'Failed' | 'NotAllSuccess' | 'Cancelled' | string

type ShellTaskFilter = {
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

type ShellTask = {
  uuid: string
  name: string
  shell: string
  corn: string
  exec_times: number
  status: ShellTaskStatus
  exec_result: string
  servers: ShellTaskFilter
  submit_user: string
  created_at: string
  updated_at: string
}

type ShellTaskRecord = {
  uuid: string
  exec_times: number
  task_id: string
  task_name: string
  shell: string
  server_ip: string
  server_name: string
  cost_time: string
  output: string
  is_success: boolean
  created_at: string
  updated_at: string
}

const taskStatusMeta = (status: ShellTaskStatus) => {
  const normalized = (status || '').trim().toLowerCase()
  if (normalized === 'success') return { className: 'badge live', label: 'SUCCESS' }
  if (normalized === 'running') return { className: 'badge connecting', label: 'RUNNING' }
  if (normalized === 'pending') return { className: 'badge idle', label: 'PENDING' }
  if (normalized === 'notallsuccess') return { className: 'badge closed', label: 'PARTIAL' }
  if (normalized === 'cancelled' || normalized === 'failed') {
    return { className: 'badge warning', label: normalized.toUpperCase() }
  }
  return { className: 'badge', label: status || 'UNKNOWN' }
}

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

const buildFilterLabels = (filters?: ShellTaskFilter) => {
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

const buildTaskSearchText = (task: ShellTask) =>
  [
    task.name,
    task.uuid,
    task.status,
    task.submit_user,
    task.corn,
    task.exec_result,
    ...buildFilterLabels(task.servers),
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()

export const AdminShellPage = () => {
  const [tasks, setTasks] = useState<ShellTask[]>([])
  const [records, setRecords] = useState<ShellTaskRecord[]>([])
  const [selectedTaskId, setSelectedTaskId] = useState('')
  const [selectedRecordId, setSelectedRecordId] = useState('')
  const [query, setQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [taskLoading, setTaskLoading] = useState(true)
  const [recordLoading, setRecordLoading] = useState(false)
  const [taskError, setTaskError] = useState('')
  const [recordError, setRecordError] = useState('')

  const loadTasks = useCallback(async () => {
    setTaskLoading(true)
    setTaskError('')
    try {
      const res = await apiClient.get<ShellTask[]>('/api/v1/shell/task')
      const nextTasks = res.data || []
      setTasks(nextTasks)
      setSelectedTaskId((current) => {
        if (current && nextTasks.some((task) => task.uuid === current)) return current
        return nextTasks[0]?.uuid || ''
      })
    } catch (err: any) {
      setTaskError(err?.response?.data || err?.message || '加载 ShellTask 列表失败')
      setTasks([])
      setSelectedTaskId('')
    } finally {
      setTaskLoading(false)
    }
  }, [])

  const loadRecords = useCallback(async (taskId: string) => {
    if (!taskId) {
      setRecords([])
      setSelectedRecordId('')
      setRecordError('')
      return
    }
    setRecordLoading(true)
    setRecordError('')
    try {
      const res = await apiClient.get<ShellTaskRecord[]>('/api/v1/shell/record', {
        params: { taskid: taskId },
      })
      const nextRecords = res.data || []
      setRecords(nextRecords)
      setSelectedRecordId((current) => {
        if (current && nextRecords.some((record) => record.uuid === current)) return current
        return nextRecords[0]?.uuid || ''
      })
    } catch (err: any) {
      setRecordError(err?.response?.data || err?.message || '加载执行记录失败')
      setRecords([])
      setSelectedRecordId('')
    } finally {
      setRecordLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadTasks()
  }, [loadTasks])

  const filteredTasks = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase()
    return tasks.filter((task) => {
      if (statusFilter !== 'all' && task.status !== statusFilter) return false
      if (!normalizedQuery) return true
      return buildTaskSearchText(task).includes(normalizedQuery)
    })
  }, [query, statusFilter, tasks])

  useEffect(() => {
    if (filteredTasks.length === 0) {
      setSelectedTaskId('')
      return
    }
    if (!filteredTasks.some((task) => task.uuid === selectedTaskId)) {
      setSelectedTaskId(filteredTasks[0].uuid)
    }
  }, [filteredTasks, selectedTaskId])

  useEffect(() => {
    void loadRecords(selectedTaskId)
  }, [loadRecords, selectedTaskId])

  const selectedTask = useMemo(
    () => tasks.find((task) => task.uuid === selectedTaskId) || null,
    [selectedTaskId, tasks],
  )
  const selectedRecord = useMemo(
    () => records.find((record) => record.uuid === selectedRecordId) || records[0] || null,
    [records, selectedRecordId],
  )
  const selectedTaskFilters = useMemo(
    () => buildFilterLabels(selectedTask?.servers),
    [selectedTask],
  )
  const summary = useMemo(() => {
    const base = {
      total: tasks.length,
      running: 0,
      pending: 0,
      success: 0,
      issues: 0,
    }
    tasks.forEach((task) => {
      const normalized = (task.status || '').trim().toLowerCase()
      if (normalized === 'running') base.running += 1
      else if (normalized === 'pending') base.pending += 1
      else if (normalized === 'success') base.success += 1
      else if (normalized === 'failed' || normalized === 'cancelled' || normalized === 'notallsuccess') {
        base.issues += 1
      }
    })
    return base
  }, [tasks])

  return (
    <div className="page console-page admin-shell-page">
      <section className="admin-shell-hero">
        <div>
          <span className="terminal-prompt-eyebrow">Admin Console</span>
          <h1>Shell Task 管理页</h1>
          <p>这里集中查看 scheduler 执行的 ShellTask、提交人、Cron 配置和每台机器的执行输出。</p>
        </div>
        <div className="admin-shell-summary">
          <div className="admin-shell-stat">
            <span>任务总数</span>
            <strong>{summary.total}</strong>
          </div>
          <div className="admin-shell-stat">
            <span>运行中</span>
            <strong>{summary.running}</strong>
          </div>
          <div className="admin-shell-stat">
            <span>待执行</span>
            <strong>{summary.pending}</strong>
          </div>
          <div className="admin-shell-stat">
            <span>成功</span>
            <strong>{summary.success}</strong>
          </div>
        </div>
      </section>

      <div className="admin-shell-layout">
        <section className="panel admin-shell-panel">
          <div className="panel-header">
            <div>
              <h3>任务列表</h3>
              <p>支持按名称、提交人、状态和过滤条件快速检索。</p>
            </div>
            <div className="panel-actions">
              <button
                className="icon-button"
                onClick={() => {
                  void loadTasks()
                }}
                disabled={taskLoading}
                title="刷新任务"
                aria-label="刷新任务"
              >
                <RefreshIcon />
              </button>
            </div>
          </div>

          <div className="panel-body">
            <div className="admin-shell-controls">
              <label>
                <span>搜索</span>
                <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="任务名 / 提交人 / IP / Cron" />
              </label>
              <label>
                <span>状态</span>
                <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
                  <option value="all">全部状态</option>
                  <option value="Pending">Pending</option>
                  <option value="Running">Running</option>
                  <option value="Success">Success</option>
                  <option value="Failed">Failed</option>
                  <option value="NotAllSuccess">NotAllSuccess</option>
                  <option value="Cancelled">Cancelled</option>
                </select>
              </label>
            </div>

            {taskError && <div className="error">{taskError}</div>}

            <div className="admin-shell-task-list">
              {taskLoading ? (
                <div className="empty-state">正在加载 ShellTask...</div>
              ) : filteredTasks.length === 0 ? (
                <div className="empty-state">没有匹配的任务</div>
              ) : (
                filteredTasks.map((task) => {
                  const status = taskStatusMeta(task.status)
                  const filterLabels = buildFilterLabels(task.servers)
                  return (
                    <button
                      type="button"
                      key={task.uuid}
                      className={`admin-shell-task-card ${task.uuid === selectedTaskId ? 'active' : ''}`}
                      onClick={() => setSelectedTaskId(task.uuid)}
                    >
                      <div className="admin-shell-task-card-head">
                        <strong>{task.name}</strong>
                        <span className={status.className}>{status.label}</span>
                      </div>
                      <div className="admin-shell-task-card-meta">
                        <span>{task.submit_user || 'system'}</span>
                        <span>{task.corn ? `Cron ${task.corn}` : '单次任务'}</span>
                        <span>执行 {task.exec_times} 次</span>
                      </div>
                      <div className="admin-shell-chip-row">
                        <span className="pill">UUID {task.uuid}</span>
                        {filterLabels.slice(0, 3).map((label) => (
                          <span key={label} className="pill">
                            {label}
                          </span>
                        ))}
                        {filterLabels.length > 3 && <span className="pill">+{filterLabels.length - 3} 条条件</span>}
                      </div>
                    </button>
                  )
                })
              )}
            </div>
          </div>
        </section>

        <section className="admin-shell-detail">
          <div className="panel admin-shell-panel">
            <div className="panel-header">
              <div>
                <h3>{selectedTask?.name || '任务详情'}</h3>
                <p>{selectedTask ? `UUID: ${selectedTask.uuid}` : '从左侧选择一个 ShellTask 查看详情。'}</p>
              </div>
              {selectedTask && <span className={taskStatusMeta(selectedTask.status).className}>{taskStatusMeta(selectedTask.status).label}</span>}
            </div>

            <div className="panel-body">
              {!selectedTask ? (
                <div className="empty-state">暂无任务详情</div>
              ) : (
                <>
                  <div className="admin-shell-metrics">
                    <div className="admin-shell-metric">
                      <span>提交人</span>
                      <strong>{selectedTask.submit_user || 'system'}</strong>
                    </div>
                    <div className="admin-shell-metric">
                      <span>调度方式</span>
                      <strong>{selectedTask.corn || '单次执行'}</strong>
                    </div>
                    <div className="admin-shell-metric">
                      <span>执行次数</span>
                      <strong>{selectedTask.exec_times}</strong>
                    </div>
                    <div className="admin-shell-metric">
                      <span>最近更新</span>
                      <strong>{formatDateTime(selectedTask.updated_at)}</strong>
                    </div>
                  </div>

                  <div className="admin-shell-section">
                    <div className="admin-shell-section-header">
                      <strong>目标机器过滤条件</strong>
                      <span>{selectedTaskFilters.length > 0 ? `${selectedTaskFilters.length} 条` : '未配置'}</span>
                    </div>
                    <div className="admin-shell-chip-row">
                      {selectedTaskFilters.length > 0 ? (
                        selectedTaskFilters.map((label) => (
                          <span key={label} className="pill">
                            {label}
                          </span>
                        ))
                      ) : (
                        <span className="muted">没有过滤条件</span>
                      )}
                    </div>
                  </div>

                  <div className="admin-shell-section">
                    <div className="admin-shell-section-header">
                      <strong>Shell 内容</strong>
                      <span>创建时间 {formatDateTime(selectedTask.created_at)}</span>
                    </div>
                    <pre className="admin-shell-code">{selectedTask.shell}</pre>
                  </div>

                  <div className="admin-shell-section">
                    <div className="admin-shell-section-header">
                      <strong>任务执行结果</strong>
                      <span>{selectedTask.exec_result ? '来自调度器汇总' : '当前没有汇总输出'}</span>
                    </div>
                    <pre className="admin-shell-output">{selectedTask.exec_result || '暂无执行结果'}</pre>
                  </div>
                </>
              )}
            </div>
          </div>

          <div className="panel admin-shell-panel">
            <div className="panel-header">
              <div>
                <h3>执行记录</h3>
                <p>{selectedTask ? '按任务维度查看每台机器的执行输出。' : '选择任务后显示执行记录。'}</p>
              </div>
              {selectedTask && (
                <div className="panel-actions">
                  <button
                    className="icon-button"
                    onClick={() => {
                      void loadRecords(selectedTask.uuid)
                    }}
                    disabled={recordLoading}
                    title="刷新记录"
                    aria-label="刷新记录"
                  >
                    <RefreshIcon />
                  </button>
                </div>
              )}
            </div>

            <div className="panel-body">
              {recordError && <div className="error">{recordError}</div>}
              {!selectedTask ? (
                <div className="empty-state">请先选择任务</div>
              ) : recordLoading ? (
                <div className="empty-state">正在加载执行记录...</div>
              ) : records.length === 0 ? (
                <div className="empty-state">当前任务还没有执行记录</div>
              ) : (
                <div className="admin-shell-record-layout">
                  <div className="admin-shell-record-list">
                    {records.map((record) => (
                      <button
                        type="button"
                        key={record.uuid}
                        className={`admin-shell-record-card ${record.uuid === selectedRecord?.uuid ? 'active' : ''}`}
                        onClick={() => setSelectedRecordId(record.uuid)}
                      >
                        <div className="admin-shell-record-card-head">
                          <strong>{record.server_name || record.server_ip}</strong>
                          <span className={record.is_success ? 'badge live' : 'badge warning'}>
                            {record.is_success ? 'SUCCESS' : 'FAILED'}
                          </span>
                        </div>
                        <div className="admin-shell-task-card-meta">
                          <span>{record.server_ip}</span>
                          <span>第 {record.exec_times} 次</span>
                          <span>{record.cost_time || '无耗时'}</span>
                        </div>
                        <div className="admin-shell-record-time">{formatDateTime(record.created_at)}</div>
                      </button>
                    ))}
                  </div>

                  <div className="admin-shell-log-viewer">
                    <div className="admin-shell-section-header">
                      <strong>{selectedRecord?.server_name || selectedRecord?.server_ip || '执行输出'}</strong>
                      <span>
                        {selectedRecord
                          ? `${selectedRecord.is_success ? '成功' : '失败'} · ${selectedRecord.cost_time || '无耗时'}`
                          : '暂无记录'}
                      </span>
                    </div>
                    <pre className="admin-shell-output">{selectedRecord?.output || '暂无输出'}</pre>
                  </div>
                </div>
              )}
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}
