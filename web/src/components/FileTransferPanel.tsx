import type { ReactNode } from 'react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { apiClient } from '../api/client'

type FileTransferPanelProps = {
  host: string
  user?: string
  token: string | null
  headerAction?: ReactNode
}

type UploadStatus = 'pending' | 'uploading' | 'done' | 'failed'

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

const remoteKeyBase = 'jms_remote_files'

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

export const FileTransferPanel = ({ host, user, token, headerAction }: FileTransferPanelProps) => {
  const [uploadPath, setUploadPath] = useState('/data/')
  const [queue, setQueue] = useState<UploadItem[]>([])
  const storageKey = host ? `${remoteKeyBase}:${host}` : `${remoteKeyBase}:default`
  const [remoteFiles, setRemoteFiles] = useState<RemoteFile[]>(() => loadRemoteFiles(storageKey))
  const [remoteInput, setRemoteInput] = useState('')
  const [status, setStatus] = useState('')
  const [dragActive, setDragActive] = useState(false)
  const [autoStart, setAutoStart] = useState(false)
  const uploadingRef = useRef(false)

  useEffect(() => {
    setRemoteFiles(loadRemoteFiles(storageKey))
  }, [storageKey])

  useEffect(() => {
    localStorage.setItem(storageKey, JSON.stringify(remoteFiles))
  }, [remoteFiles, storageKey])

  const canOperate = useMemo(() => Boolean(token && host), [token, host])

  const addFiles = (files: File[]) => {
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

  const removeItem = (id: string) => {
    setQueue((prev) => prev.filter((item) => item.id !== id))
  }

  const startUploadQueue = async () => {
    if (uploadingRef.current) return
    uploadingRef.current = true
    try {
      for (const item of queue) {
        if (item.status !== 'pending' && item.status !== 'failed') continue
        await uploadOne(item)
      }
    } finally {
      uploadingRef.current = false
    }
  }

  const uploadOne = async (item: UploadItem) => {
    try {
      if (!token || !host) {
        updateItem(item.id, { status: 'failed', detail: '请先登录并选择主机' })
        return
      }

      const path = resolveTargetPath(uploadPath.trim(), item.file.name)
      if (!path) {
        updateItem(item.id, { status: 'failed', detail: '请填写目标路径' })
        return
      }

      updateItem(item.id, { status: 'uploading', detail: '初始化上传' })

      const uploadKey = `upload:${host}:${user || 'default'}:${path}:${item.file.name}:${item.file.size}`

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
        const res = await apiClient.post<InitResp>('/api/v1/files/upload/init', {
          host,
          path,
          user,
          size: item.file.size,
        })
        uploadId = res.data.upload_id
        chunkSize = res.data.chunk_size
        startIndex = 0
        localStorage.setItem(uploadKey, JSON.stringify({ uploadId, chunkSize, nextIndex: 0 }))
      }

      const totalChunks = Math.ceil(item.file.size / chunkSize)

      for (let index = startIndex; index < totalChunks; index += 1) {
        const start = index * chunkSize
        const end = Math.min(start + chunkSize, item.file.size)
        const blob = item.file.slice(start, end)
        updateItem(item.id, { detail: `上传中 ${index + 1}/${totalChunks}` })

        const resp = await fetch(`/api/v1/files/upload/chunk?upload_id=${uploadId}&index=${index}`, {
          method: 'PUT',
          headers: { Authorization: `Bearer ${token}` },
          body: blob,
        })
        if (!resp.ok) {
          updateItem(item.id, { status: 'failed', detail: await resp.text() })
          return
        }

        const nextIndex = index + 1
        localStorage.setItem(uploadKey, JSON.stringify({ uploadId, chunkSize, nextIndex }))
        updateItem(item.id, { progress: Math.round((nextIndex / totalChunks) * 100) })
      }

      updateItem(item.id, { detail: '合并中' })
      await apiClient.post('/api/v1/files/upload/complete', {
        upload_id: uploadId,
        total_chunks: totalChunks,
      })

      localStorage.removeItem(uploadKey)
      updateItem(item.id, { status: 'done', detail: '完成' })
    } catch (err: any) {
      updateItem(item.id, { status: 'failed', detail: err?.message || '上传失败' })
    }
  }

  const addRemotePath = () => {
    const path = remoteInput.trim()
    if (!path) return
    const entry: RemoteFile = {
      id: `${path}-${Math.random().toString(36).slice(2)}`,
      path,
      lastUsed: Date.now(),
    }
    setRemoteFiles((prev) => [entry, ...prev.filter((item) => item.path !== path)].slice(0, 20))
    setRemoteInput('')
  }

  const downloadRemote = async (path: string) => {
    if (!token || !host) {
      setStatus('请先登录并选择主机')
      return
    }
    setStatus('下载中...')
    const url = `/api/v1/files/download?host=${encodeURIComponent(host)}&path=${encodeURIComponent(path)}${user ? `&user=${encodeURIComponent(user)}` : ''}`
    try {
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) {
        setStatus('下载失败')
        return
      }
      const blob = await res.blob()
      const filename = path.split('/').pop() || 'download'
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = filename
      a.click()
      URL.revokeObjectURL(a.href)
      setStatus('下载完成')
      setRemoteFiles((prev) =>
        prev.map((item) => (item.path === path ? { ...item, lastUsed: Date.now() } : item)),
      )
    } catch (err: any) {
      setStatus(err?.message || '下载失败')
    }
  }

  const onDropFiles = (files: FileList | null) => {
    if (!files || files.length === 0) return
    addFiles(Array.from(files))
    if (canOperate && uploadPath.trim()) {
      setAutoStart(true)
    }
  }

  useEffect(() => {
    if (!autoStart) return
    setAutoStart(false)
    void startUploadQueue()
  }, [autoStart, queue])

  return (
    <div className="panel transfer-panel">
      <div className="panel-header">
        <div>
          <h3>文件传输</h3>
          <p>{host ? `当前主机: ${host}${user ? ` · ${user}` : ''}` : '选择主机后可开始传输'}</p>
        </div>
        {headerAction && <div className="panel-actions">{headerAction}</div>}
      </div>
      <div className="panel-body">
        <label>
          <span>目标路径</span>
          <input
            value={uploadPath}
            onChange={(e) => setUploadPath(e.target.value)}
            placeholder="/data/ (以 / 结尾表示目录)"
          />
        </label>

        <div
          className={`dropzone ${dragActive ? 'active' : ''}`}
          onDragOver={(e) => {
            e.preventDefault()
            setDragActive(true)
          }}
          onDragLeave={() => setDragActive(false)}
          onDrop={(e) => {
            e.preventDefault()
            setDragActive(false)
            onDropFiles(e.dataTransfer.files)
          }}
        >
          <input
            type="file"
            multiple
            onChange={(e) => onDropFiles(e.target.files)}
            title="选择文件"
          />
          <div>
            <strong>拖拽文件到此处上传</strong>
            <p>支持批量上传，路径以 / 结尾将自动拼接文件名</p>
          </div>
        </div>

        <div className="row equal">
          <button className="primary" onClick={startUploadQueue} disabled={!canOperate || !queue.length}>
            开始上传
          </button>
          <button className="ghost" onClick={() => setQueue([])} disabled={!queue.length}>
            清空队列
          </button>
        </div>

        <div className="upload-list">
          {queue.length === 0 ? (
            <div className="empty-state">暂无上传任务</div>
          ) : (
            queue.map((item) => (
              <div className={`upload-item ${item.status}`} key={item.id}>
                <div className="file-meta">
                  <strong>{item.file.name}</strong>
                  <span>{formatBytes(item.file.size)}</span>
                </div>
                <div className="file-status">
                  <span>{item.detail || item.status}</span>
                  <div className="mini-progress">
                    <div style={{ width: `${item.progress}%` }} />
                  </div>
                </div>
                <div className="file-actions">
                  {item.status === 'failed' && (
                    <button className="ghost" onClick={() => updateItem(item.id, { status: 'pending', detail: '' })}>
                      重试
                    </button>
                  )}
                  <button className="ghost" onClick={() => removeItem(item.id)}>
                    移除
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      <div className="panel-divider" />

      <div className="panel-body">
        <div className="section-title">远端文件</div>
        <div className="row">
          <input
            value={remoteInput}
            onChange={(e) => setRemoteInput(e.target.value)}
            placeholder="/data/report.zip"
          />
          <button className="ghost" onClick={addRemotePath}>
            添加
          </button>
        </div>
        <div className="remote-list">
          {remoteFiles.length === 0 ? (
            <div className="empty-state">暂无文件记录</div>
          ) : (
            remoteFiles.map((item) => (
              <div className="remote-item" key={item.id}>
                <div>
                  <strong>{item.path}</strong>
                  {item.lastUsed && <span>最近下载: {new Date(item.lastUsed).toLocaleString()}</span>}
                </div>
                <button className="ghost" onClick={() => downloadRemote(item.path)}>
                  下载
                </button>
              </div>
            ))
          )}
        </div>
        {status && <div className="status">{status}</div>}
      </div>
    </div>
  )
}
