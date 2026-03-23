import type { ReactNode } from 'react'
import { useCallback, useDeferredValue, useEffect, useMemo, useRef, useState } from 'react'
import { apiFetch } from '../api/auth'
import { apiClient } from '../api/client'

type FileTransferPanelProps = {
  host: string
  user?: string
  token: string | null
  connected: boolean
  headerAction?: ReactNode
}

type UploadStatus = 'pending' | 'uploading' | 'done' | 'failed' | 'cancelled'
type TransferPage = 'upload' | 'download'

type UploadItem = {
  id: string
  file: File
  progress: number
  status: UploadStatus
  detail?: string
}

type InitResp = {
  upload_id: string
  chunk_size: number
  expires_at: number
}

type RemoteFile = {
  id: string
  path: string
  lastUsed?: number
}

type BrowseItem = {
  name: string
  path: string
  is_dir: boolean
  size: number
  updated_at: number
}

type BrowseResp = {
  path: string
  parent_path?: string | null
  search: string
  limit: number
  truncated: boolean
  items: BrowseItem[]
}

const remoteKeyBase = 'jms_remote_files'
const browseLimit = 200
const rootBrowsePath = '/'

const loadRemoteFiles = (key: string): RemoteFile[] => {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) return parsed
    return []
  } catch {
    return []
  }
}

const formatBytes = (size: number) => {
  if (size < 1024) return `${size} B`
  const kb = size / 1024
  if (kb < 1024) return `${kb.toFixed(1)} KB`
  const mb = kb / 1024
  if (mb < 1024) return `${mb.toFixed(1)} MB`
  const gb = mb / 1024
  return `${gb.toFixed(2)} GB`
}

const resolveTargetPath = (input: string, filename: string) => {
  if (!input) return ''
  if (input.endsWith('/')) return `${input}${filename}`
  return input
}

const resolveDefaultBrowsePath = (sshUser?: string) => {
  if (!sshUser) return rootBrowsePath
  if (sshUser === 'root') return '/root'
  return `/home/${sshUser}`
}

const isMissingPathError = (value: string) => {
  const normalized = value.trim().toLowerCase()
  return (
    normalized.includes('no such file') ||
    normalized.includes('not exist') ||
    normalized.includes('cannot find') ||
    normalized.includes('file does not exist')
  )
}

const buildBreadcrumbs = (value: string) => {
  const normalized = value && value.startsWith('/') ? value : '/'
  if (normalized === '/') return [{ label: '根目录', path: '/' }]

  const parts = normalized.split('/').filter(Boolean)
  const crumbs = [{ label: '根目录', path: '/' }]
  let current = ''
  parts.forEach((part) => {
    current += `/${part}`
    crumbs.push({ label: part, path: current })
  })
  return crumbs
}

const formatTime = (value?: number) => {
  if (!value) return ''
  return new Date(value * 1000).toLocaleString()
}

const isAbortError = (err: unknown) => {
  const error = err as { name?: string; code?: string }
  return error?.name === 'AbortError' || error?.name === 'CanceledError' || error?.code === 'ERR_CANCELED'
}

export const FileTransferPanel = ({ host, user, token, connected, headerAction }: FileTransferPanelProps) => {
  const [activePage, setActivePage] = useState<TransferPage>('upload')
  const [uploadPath, setUploadPath] = useState('/data/')
  const [queue, setQueue] = useState<UploadItem[]>([])
  const storageKey = host ? `${remoteKeyBase}:${host}:${user || 'default'}` : `${remoteKeyBase}:default`
  const [remoteFiles, setRemoteFiles] = useState<RemoteFile[]>(() => loadRemoteFiles(storageKey))
  const [downloadStatus, setDownloadStatus] = useState('')
  const [dragActive, setDragActive] = useState(false)
  const [browseOpen, setBrowseOpen] = useState(false)
  const [browsePath, setBrowsePath] = useState(resolveDefaultBrowsePath(user))
  const [browseSearchInput, setBrowseSearchInput] = useState('')
  const [browseItems, setBrowseItems] = useState<BrowseItem[]>([])
  const [browseParentPath, setBrowseParentPath] = useState<string | null>(null)
  const [browseTruncated, setBrowseTruncated] = useState(false)
  const [browseLoading, setBrowseLoading] = useState(false)
  const [browseError, setBrowseError] = useState('')
  const [selectedDownloadPath, setSelectedDownloadPath] = useState('')
  const [downloadBusy, setDownloadBusy] = useState(false)
  const [browseReloadToken, setBrowseReloadToken] = useState(0)
  const uploadingRef = useRef(false)
  const canOperateRef = useRef(false)
  const uploadAbortRef = useRef<Record<string, AbortController>>({})
  const uploadAbortReasonRef = useRef<Record<string, 'cancelled' | 'disconnected'>>({})
  const deferredBrowseSearch = useDeferredValue(browseSearchInput.trim())

  useEffect(() => {
    setRemoteFiles(loadRemoteFiles(storageKey))
  }, [storageKey])

  useEffect(() => {
    localStorage.setItem(storageKey, JSON.stringify(remoteFiles))
  }, [remoteFiles, storageKey])

  const canOperate = useMemo(() => Boolean(token && host && connected), [token, host, connected])
  const operationHint = useMemo(() => {
    if (!token) return '请先登录'
    if (!host) return '请先在左侧选择机器并连接用户'
    if (!connected) return '请先建立终端连接，断开后文件传输不可用'
    return ''
  }, [token, host, connected])

  useEffect(() => {
    canOperateRef.current = canOperate
    if (!canOperate) {
      setDragActive(false)
      setBrowseOpen(false)
    }
  }, [canOperate])

  useEffect(() => {
    if (connected) return
    Object.entries(uploadAbortRef.current).forEach(([id, controller]) => {
      uploadAbortReasonRef.current[id] = 'disconnected'
      controller.abort()
    })
  }, [connected])

  useEffect(() => {
    if (!browseOpen) return

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !downloadBusy) {
        setBrowseOpen(false)
      }
    }

    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [browseOpen, downloadBusy])

  const addFiles = (files: File[]) => {
    if (!canOperateRef.current) return
    const items = files.map((file) => ({
      id: `${file.name}-${file.size}-${file.lastModified}-${Math.random().toString(36).slice(2)}`,
      file,
      progress: 0,
      status: 'pending' as UploadStatus,
    }))
    setQueue((prev) => [...items, ...prev].slice(0, 20))
  }

  const updateItem = (id: string, patch: Partial<UploadItem>) => {
    setQueue((prev) => prev.map((item) => (item.id === id ? { ...item, ...patch } : item)))
  }

  const rememberRemoteFile = useCallback((path: string) => {
    setRemoteFiles((prev) => {
      const now = Date.now()
      const existing = prev.find((item) => item.path === path)
      const nextEntry = existing || {
        id: `${path}-${Math.random().toString(36).slice(2)}`,
        path,
      }
      return [{ ...nextEntry, lastUsed: now }, ...prev.filter((item) => item.path !== path)].slice(0, 20)
    })
  }, [])

  const startUploadQueue = useCallback(async () => {
    if (!canOperateRef.current) return
    if (uploadingRef.current) return
    uploadingRef.current = true
    try {
      for (const item of queue) {
        if (item.status !== 'pending') continue
        await uploadOne(item)
      }
    } finally {
      uploadingRef.current = false
    }
  }, [queue])

  const uploadOne = async (item: UploadItem) => {
    let uploadKey = ''
    const controller = new AbortController()
    uploadAbortRef.current[item.id] = controller

    try {
      if (!token || !host || !connected) {
        updateItem(item.id, { status: 'failed', detail: '请先建立终端连接后再传输文件' })
        return
      }

      const path = resolveTargetPath(uploadPath.trim(), item.file.name)
      if (!path) {
        updateItem(item.id, { status: 'failed', detail: '请填写目标路径' })
        return
      }

      updateItem(item.id, { status: 'uploading', detail: '初始化上传' })

      uploadKey = `upload:${host}:${user || 'default'}:${path}:${item.file.name}:${item.file.size}`

      let uploadId = ''
      let chunkSize = 0
      let startIndex = 0

      if (localStorage.getItem(uploadKey)) {
        try {
          const saved = JSON.parse(localStorage.getItem(uploadKey) || '{}')
          uploadId = saved.uploadId
          chunkSize = saved.chunkSize
          startIndex = saved.nextIndex || 0
        } catch {
          localStorage.removeItem(uploadKey)
        }
      }

      if (!uploadId) {
        const res = await apiClient.post<InitResp>(
          '/api/v1/files/upload/init',
          {
            host,
            path,
            user,
            size: item.file.size,
          },
          { signal: controller.signal },
        )
        uploadId = res.data.upload_id
        chunkSize = res.data.chunk_size
        startIndex = 0
        localStorage.setItem(uploadKey, JSON.stringify({ uploadId, chunkSize, nextIndex: 0 }))
      }

      const totalChunks = Math.ceil(item.file.size / chunkSize)

      for (let index = startIndex; index < totalChunks; index += 1) {
        if (!canOperateRef.current) {
          updateItem(item.id, { status: 'failed', detail: '连接已断开，上传已停止' })
          return
        }
        const start = index * chunkSize
        const end = Math.min(start + chunkSize, item.file.size)
        const blob = item.file.slice(start, end)
        updateItem(item.id, { detail: `上传中 ${index + 1}/${totalChunks}` })

        const resp = await apiFetch(`/api/v1/files/upload/chunk?upload_id=${uploadId}&index=${index}`, {
          method: 'PUT',
          headers: { Authorization: `Bearer ${token}` },
          body: blob,
          signal: controller.signal,
        })
        if (resp.status === 401) return
        if (!resp.ok) {
          updateItem(item.id, { status: 'failed', detail: await resp.text() })
          return
        }

        const nextIndex = index + 1
        localStorage.setItem(uploadKey, JSON.stringify({ uploadId, chunkSize, nextIndex }))
        updateItem(item.id, { progress: Math.round((nextIndex / totalChunks) * 100) })
      }

      if (!canOperateRef.current) {
        updateItem(item.id, { status: 'failed', detail: '连接已断开，上传已停止' })
        return
      }
      updateItem(item.id, { detail: '合并中' })
      await apiClient.post(
        '/api/v1/files/upload/complete',
        {
          upload_id: uploadId,
          total_chunks: totalChunks,
        },
        { signal: controller.signal },
      )

      localStorage.removeItem(uploadKey)
      updateItem(item.id, { status: 'done', detail: '上传成功', progress: 100 })
    } catch (err: any) {
      if (isAbortError(err)) {
        if (uploadKey) {
          localStorage.removeItem(uploadKey)
        }
        const reason = uploadAbortReasonRef.current[item.id]
        updateItem(item.id, {
          status: reason === 'disconnected' ? 'failed' : 'cancelled',
          detail: reason === 'disconnected' ? '连接已断开，上传已停止' : '传输已取消',
        })
        return
      }
      updateItem(item.id, { status: 'failed', detail: err?.message || '上传失败' })
    } finally {
      delete uploadAbortRef.current[item.id]
      delete uploadAbortReasonRef.current[item.id]
    }
  }

  const downloadRemote = useCallback(
    async (path: string) => {
      if (!token || !host || !connected) {
        setDownloadStatus('请先建立终端连接后再传输文件')
        return false
      }

      setDownloadBusy(true)
      setDownloadStatus('下载中...')
      const url = `/api/v1/files/download?host=${encodeURIComponent(host)}&path=${encodeURIComponent(path)}${user ? `&user=${encodeURIComponent(user)}` : ''}`

      try {
        const res = await apiFetch(url, {
          headers: { Authorization: `Bearer ${token}` },
        })
        if (res.status === 401) return false
        if (!res.ok) {
          const message = await res.text()
          setDownloadStatus(message || '下载失败')
          return false
        }

        const blob = await res.blob()
        const filename = path.split('/').pop() || 'download'
        const a = document.createElement('a')
        a.href = URL.createObjectURL(blob)
        a.download = filename
        a.click()
        URL.revokeObjectURL(a.href)
        setDownloadStatus('下载完成')
        rememberRemoteFile(path)
        return true
      } catch (err: any) {
        setDownloadStatus(err?.message || '下载失败')
        return false
      } finally {
        setDownloadBusy(false)
      }
    },
    [connected, host, rememberRemoteFile, token, user],
  )

  const openDownloadBrowser = () => {
    if (!canOperateRef.current) {
      setDownloadStatus('请先建立终端连接后再传输文件')
      return
    }

    setDownloadStatus('')
    setBrowseError('')
    setBrowseItems([])
    setBrowseTruncated(false)
    setBrowseSearchInput('')
    setSelectedDownloadPath('')
    setBrowsePath(resolveDefaultBrowsePath(user))
    setBrowseReloadToken((prev) => prev + 1)
    setBrowseOpen(true)
  }

  const closeDownloadBrowser = () => {
    if (downloadBusy) return
    setBrowseOpen(false)
  }

  const navigateBrowsePath = (nextPath: string) => {
    setBrowsePath(nextPath)
    setBrowseSearchInput('')
    setBrowseError('')
    setBrowseTruncated(false)
    setSelectedDownloadPath('')
  }

  const confirmDownload = async () => {
    if (!selectedDownloadPath || downloadBusy) return
    const success = await downloadRemote(selectedDownloadPath)
    if (success) {
      setBrowseOpen(false)
    }
  }

  useEffect(() => {
    if (!browseOpen || !canOperate || !host) return

    const controller = new AbortController()
    setBrowseLoading(true)
    setBrowseError('')

    void apiClient
      .get<BrowseResp>('/api/v1/files/browse', {
        params: {
          host,
          path: browsePath,
          user,
          search: deferredBrowseSearch || undefined,
          limit: browseLimit,
        },
        signal: controller.signal,
      })
      .then((res) => {
        setBrowseItems(res.data.items)
        setBrowseParentPath(res.data.parent_path || null)
        setBrowseTruncated(res.data.truncated)
        setSelectedDownloadPath((prev) =>
          res.data.items.some((item) => !item.is_dir && item.path === prev) ? prev : '',
        )
      })
      .catch((err: any) => {
        if (isAbortError(err)) return
        const message =
          (typeof err?.response?.data === 'string' && err.response.data) ||
          err?.message ||
          '读取服务器目录失败'
        const defaultBrowsePath = resolveDefaultBrowsePath(user)
        if (!deferredBrowseSearch && browsePath === defaultBrowsePath && browsePath !== rootBrowsePath && isMissingPathError(message)) {
          setBrowsePath(rootBrowsePath)
          return
        }
        setBrowseItems([])
        setBrowseParentPath(null)
        setBrowseTruncated(false)
        setSelectedDownloadPath('')
        setBrowseError(message)
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setBrowseLoading(false)
        }
      })

    return () => controller.abort()
  }, [browseOpen, browsePath, browseReloadToken, canOperate, deferredBrowseSearch, host, user])

  const onDropFiles = (files: FileList | null) => {
    if (!canOperateRef.current) return
    if (!files || files.length === 0) return
    addFiles(Array.from(files))
  }

  const visibleUploadItem = useMemo(() => {
    return (
      queue.find((item) => item.status === 'uploading') ||
      queue.find((item) => item.status === 'pending') ||
      queue.find((item) => item.status === 'failed') ||
      queue.find((item) => item.status === 'cancelled') ||
      queue.find((item) => item.status === 'done') ||
      null
    )
  }, [queue])

  const visibleUploadLabel = useMemo(() => {
    if (!visibleUploadItem) return ''
    if (visibleUploadItem.status === 'uploading') return '正在自动上传'
    if (visibleUploadItem.status === 'failed') return '上传失败'
    if (visibleUploadItem.status === 'cancelled') return '上传已取消'
    if (visibleUploadItem.status === 'done') return '上传完成'
    return '准备上传'
  }, [visibleUploadItem])

  const browseBreadcrumbs = useMemo(() => buildBreadcrumbs(browsePath), [browsePath])

  useEffect(() => {
    if (!canOperate) return
    if (!uploadPath.trim()) return
    if (!queue.some((item) => item.status === 'pending')) return
    void startUploadQueue()
  }, [canOperate, queue, uploadPath, startUploadQueue])

  return (
    <>
      <div className="panel transfer-panel">
        <div className="panel-header transfer-panel-header">
          <div className="transfer-panel-heading">
            <h3>文件传输</h3>
            <p>{host ? `当前主机: ${host}${user ? ` · ${user}` : ''}` : '选择主机后可开始传输'}</p>
          </div>
          {headerAction && <div className="panel-actions">{headerAction}</div>}
        </div>

        <div className="panel-body transfer-switcher">
          <div className="transfer-tabs" role="tablist" aria-label="文件传输页面">
            <button
              type="button"
              className={`transfer-tab ${activePage === 'upload' ? 'active' : ''}`}
              onClick={() => setActivePage('upload')}
            >
              上传
            </button>
            <button
              type="button"
              className={`transfer-tab ${activePage === 'download' ? 'active' : ''}`}
              onClick={() => setActivePage('download')}
            >
              下载
            </button>
          </div>
          {operationHint && <div className="transfer-hint">{operationHint}</div>}
        </div>

        <div className="panel-divider" />

        <div className="panel-body transfer-content">
          {activePage === 'upload' && (
            <div className="transfer-section">
              <div className="transfer-auto-note">
                <strong>选择文件后会自动开始上传</strong>
                <span>暂不支持目录上传，如需上传目录请先在本地打包为 zip 或 tar.gz 后再上传。</span>
              </div>

              <label className="transfer-path-label">
                <span>目标路径</span>
                <input
                  value={uploadPath}
                  onChange={(e) => setUploadPath(e.target.value)}
                  placeholder="/data/ (以 / 结尾表示目录)"
                  disabled={!canOperate}
                />
              </label>

              <div
                className={`dropzone ${dragActive ? 'active' : ''} ${canOperate ? '' : 'disabled'}`}
                onDragOver={(e) => {
                  if (!canOperate) return
                  e.preventDefault()
                  setDragActive(true)
                }}
                onDragLeave={() => setDragActive(false)}
                onDrop={(e) => {
                  if (!canOperate) return
                  e.preventDefault()
                  setDragActive(false)
                  onDropFiles(e.dataTransfer.files)
                }}
              >
                <input
                  type="file"
                  multiple
                  onChange={(e) => {
                    onDropFiles(e.target.files)
                    e.currentTarget.value = ''
                  }}
                  title="选择文件"
                  disabled={!canOperate}
                />
                <div>
                  <strong>拖拽或选择文件后自动上传</strong>
                  <p>支持批量文件上传，不支持目录；路径以 / 结尾时会自动拼接文件名</p>
                </div>
              </div>

              {visibleUploadItem && (
                <div className={`transfer-upload-status ${visibleUploadItem.status}`}>
                  <div className="transfer-upload-status-header">
                    <strong>{visibleUploadLabel}</strong>
                    <em className={`upload-status-pill ${visibleUploadItem.status}`}>
                      {visibleUploadItem.status === 'done'
                        ? '成功'
                        : visibleUploadItem.status === 'uploading'
                          ? '传输中'
                          : visibleUploadItem.status === 'failed'
                            ? '失败'
                            : visibleUploadItem.status === 'cancelled'
                              ? '已取消'
                              : '准备中'}
                    </em>
                  </div>
                  <div className="transfer-upload-status-meta">
                    <strong>{visibleUploadItem.file.name}</strong>
                    <span>{formatBytes(visibleUploadItem.file.size)}</span>
                  </div>
                  <div className="file-status">
                    <span>{visibleUploadItem.detail || visibleUploadLabel}</span>
                    <div className="mini-progress">
                      <div style={{ width: `${visibleUploadItem.progress}%` }} />
                    </div>
                  </div>
                  <div className="transfer-upload-status-actions">
                    {(visibleUploadItem.status === 'failed' || visibleUploadItem.status === 'cancelled') && (
                      <button
                        type="button"
                        className="ghost"
                        onClick={() => updateItem(visibleUploadItem.id, { status: 'pending', detail: '', progress: 0 })}
                      >
                        重新上传
                      </button>
                    )}
                    {visibleUploadItem.status === 'uploading' && (
                      <button
                        type="button"
                        className="ghost"
                        onClick={() => {
                          const controller = uploadAbortRef.current[visibleUploadItem.id]
                          if (!controller) return
                          uploadAbortReasonRef.current[visibleUploadItem.id] = 'cancelled'
                          controller.abort()
                        }}
                      >
                        取消当前传输
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          {activePage === 'download' && (
            <div className="transfer-section">
              <div className="transfer-download-note">
                <strong>从服务器选择文件后下载到本地</strong>
                <span>默认打开当前登录用户 home 目录，包含隐藏文件；目录项过多时请直接搜索文件名。</span>
              </div>

              <div className="transfer-download-actions">
                <button type="button" className="primary" onClick={openDownloadBrowser} disabled={!canOperate || downloadBusy}>
                  选择服务器文件
                </button>
              </div>

              <div className="section-title">最近下载</div>
              <div className="remote-list">
                {remoteFiles.length === 0 ? (
                  <div className="empty-state">暂无下载记录</div>
                ) : (
                  remoteFiles.map((item) => (
                    <div className="remote-item" key={item.id}>
                      <div className="remote-item-main">
                        <strong>{item.path}</strong>
                        {item.lastUsed && <span>最近下载: {new Date(item.lastUsed).toLocaleString()}</span>}
                      </div>
                      <button
                        type="button"
                        className="ghost"
                        onClick={() => {
                          void downloadRemote(item.path)
                        }}
                        disabled={!canOperate || downloadBusy}
                      >
                        下载
                      </button>
                    </div>
                  ))
                )}
              </div>
              {downloadStatus && <div className="status">{downloadStatus}</div>}
            </div>
          )}
        </div>
      </div>

      {browseOpen && (
        <div className="modal-backdrop" onClick={closeDownloadBrowser}>
          <div className="modal file-browser-modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div>
                <h3>选择服务器文件</h3>
                <p>{host ? `${host}${user ? ` · ${user}` : ''}` : '当前未连接目标机器'}</p>
              </div>
              <button type="button" className="ghost small" onClick={closeDownloadBrowser} disabled={downloadBusy}>
                关闭
              </button>
            </div>

            <div className="modal-body file-browser-body">
              <div className="file-browser-toolbar">
                <div className="file-browser-breadcrumbs" aria-label="当前目录">
                  {browseBreadcrumbs.map((crumb, index) => (
                    <button
                      type="button"
                      className={`file-browser-crumb ${crumb.path === browsePath ? 'active' : ''}`}
                      key={crumb.path}
                      onClick={() => navigateBrowsePath(crumb.path)}
                      disabled={browseLoading && crumb.path === browsePath}
                    >
                      {index > 0 && <span className="file-browser-crumb-sep">/</span>}
                      <span>{crumb.label}</span>
                    </button>
                  ))}
                </div>

                <div className="file-browser-toolbar-actions">
                  <button
                    type="button"
                    className="ghost small"
                    onClick={() => {
                      if (!browseParentPath) return
                      navigateBrowsePath(browseParentPath)
                    }}
                    disabled={!browseParentPath || browseLoading}
                  >
                    上一级
                  </button>
                  <button
                    type="button"
                    className="ghost small"
                    onClick={() => setBrowseReloadToken((prev) => prev + 1)}
                    disabled={browseLoading}
                  >
                    刷新
                  </button>
                </div>
              </div>

              <label className="file-browser-search">
                <span>搜索当前目录</span>
                <input
                  value={browseSearchInput}
                  onChange={(e) => setBrowseSearchInput(e.target.value)}
                  placeholder="输入文件名或目录名"
                  disabled={browseLoading && !browseItems.length}
                />
              </label>

              <div className="file-browser-summary">
                <span>当前目录: {browsePath}</span>
                {browseTruncated && <strong>结果过多，仅展示前 {browseLimit} 项，请继续搜索</strong>}
              </div>

              {browseError && <div className="error">{browseError}</div>}

              <div className="file-browser-list" role="listbox" aria-label="服务器文件列表">
                {browseItems.length > 0 && (
                  <div className="file-browser-list-head" aria-hidden="true">
                    <span>名称</span>
                    <span>大小</span>
                    <span>修改时间</span>
                  </div>
                )}
                {browseLoading && browseItems.length === 0 ? (
                  <div className="empty-state file-browser-empty">目录读取中...</div>
                ) : browseItems.length === 0 ? (
                  <div className="empty-state file-browser-empty">
                    {deferredBrowseSearch ? '当前搜索没有匹配项' : '当前目录没有可显示的文件'}
                  </div>
                ) : (
                  browseItems.map((item) => {
                    const selected = selectedDownloadPath === item.path
                    return (
                      <div
                        className={`file-browser-row ${selected ? 'active' : ''} ${item.is_dir ? 'dir' : 'file'}`}
                        key={item.path}
                        role="option"
                        aria-selected={selected}
                        tabIndex={0}
                        onClick={() => {
                          if (item.is_dir) {
                            navigateBrowsePath(item.path)
                            return
                          }
                          setSelectedDownloadPath(item.path)
                        }}
                        onKeyDown={(event) => {
                          if (event.key !== 'Enter' && event.key !== ' ') return
                          event.preventDefault()
                          if (item.is_dir) {
                            navigateBrowsePath(item.path)
                            return
                          }
                          setSelectedDownloadPath(item.path)
                        }}
                      >
                        <div className="file-browser-row-main">
                          <div className="file-browser-row-name">
                            <span className={`file-browser-kind ${item.is_dir ? 'dir' : 'file'}`}>
                              {item.is_dir ? 'DIR' : 'FILE'}
                            </span>
                            <div className="file-browser-row-copy">
                              <strong>{item.name}</strong>
                              <span>{item.path}</span>
                            </div>
                          </div>
                          <span className="file-browser-row-size">{item.is_dir ? '-' : formatBytes(item.size)}</span>
                          <span className="file-browser-row-time">
                            {item.is_dir ? '点击进入目录' : formatTime(item.updated_at) || '-'}
                          </span>
                        </div>
                      </div>
                    )
                  })
                )}
              </div>
            </div>

            <div className="modal-actions file-browser-actions">
              <div className="file-browser-selection">
                {selectedDownloadPath ? (
                  <span>已选择: {selectedDownloadPath}</span>
                ) : (
                  <span>请选择一个文件后再下载</span>
                )}
              </div>
              <button type="button" className="ghost" onClick={closeDownloadBrowser} disabled={downloadBusy}>
                取消
              </button>
              <button type="button" className="primary" onClick={confirmDownload} disabled={!selectedDownloadPath || downloadBusy}>
                {downloadBusy ? '下载中...' : '确认下载'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
