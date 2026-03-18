import { useState } from 'react'
import { apiClient } from '../api/client'
import { useAuthStore } from '../store/auth'

type InitResp = {
  upload_id: string
  chunk_size: number
  expires_at: number
}

export const FilesPage = () => {
  const token = useAuthStore((s) => s.token)
  const [host, setHost] = useState('')
  const [path, setPath] = useState('')
  const [sshUser, setSshUser] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [progress, setProgress] = useState(0)
  const [status, setStatus] = useState<string>('')
  const [downloadPath, setDownloadPath] = useState('')

  const uploadKey = file ? `upload:${file.name}:${file.size}` : ''

  const startUpload = async () => {
    if (!file || !token) return
    try {
      setStatus('init')
      setProgress(0)

      let uploadId = ''
      let chunkSize = 0
      let startIndex = 0

      if (uploadKey && localStorage.getItem(uploadKey)) {
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
          user: sshUser || undefined,
          size: file.size,
        })
        uploadId = res.data.upload_id
        chunkSize = res.data.chunk_size
        startIndex = 0
        localStorage.setItem(
          uploadKey,
          JSON.stringify({ uploadId, chunkSize, nextIndex: 0 }),
        )
      }

      const totalChunks = Math.ceil(file.size / chunkSize)
      for (let index = startIndex; index < totalChunks; index += 1) {
        const start = index * chunkSize
        const end = Math.min(start + chunkSize, file.size)
        const blob = file.slice(start, end)
        setStatus(`uploading ${index + 1}/${totalChunks}`)

        const resp = await fetch(
          `/api/v1/files/upload/chunk?upload_id=${uploadId}&index=${index}`,
          {
            method: 'PUT',
            headers: { Authorization: `Bearer ${token}` },
            body: blob,
          },
        )
        if (!resp.ok) {
          throw new Error(await resp.text())
        }

        const nextIndex = index + 1
        localStorage.setItem(
          uploadKey,
          JSON.stringify({ uploadId, chunkSize, nextIndex }),
        )
        setProgress(Math.round((nextIndex / totalChunks) * 100))
      }

      setStatus('finalizing')
      await apiClient.post('/api/v1/files/upload/complete', {
        upload_id: uploadId,
        total_chunks: totalChunks,
      })

      localStorage.removeItem(uploadKey)
      setStatus('done')
    } catch (err: any) {
      setStatus(err?.message || 'upload failed')
    }
  }

  const startDownload = async () => {
    if (!token || !host || !downloadPath) return
    setStatus('downloading')
    const url = `/api/v1/files/download?host=${encodeURIComponent(host)}&path=${encodeURIComponent(downloadPath)}${sshUser ? `&user=${encodeURIComponent(sshUser)}` : ''}`
    try {
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) {
        setStatus('download failed')
        return
      }
      const blob = await res.blob()
      const filename = downloadPath.split('/').pop() || 'download'
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = filename
      a.click()
      URL.revokeObjectURL(a.href)
      setStatus('downloaded')
    } catch (err: any) {
      setStatus(err?.message || 'download failed')
    }
  }

  return (
    <div className="page files-page">
      <div className="card wide">
        <h2>文件上传</h2>
        <div className="grid">
          <label>
            <span>Host</span>
            <input value={host} onChange={(e) => setHost(e.target.value)} placeholder="10.0.0.1" />
          </label>
          <label>
            <span>SSH User (optional)</span>
            <input value={sshUser} onChange={(e) => setSshUser(e.target.value)} placeholder="root" />
          </label>
          <label>
            <span>目标路径</span>
            <input value={path} onChange={(e) => setPath(e.target.value)} placeholder="/data/file.zip" />
          </label>
        </div>
        <div className="grid">
          <input type="file" onChange={(e) => setFile(e.target.files?.[0] || null)} />
          <button className="primary" onClick={startUpload} disabled={!token || !file || !host || !path}>
            开始上传
          </button>
        </div>
        <div className="progress">
          <div style={{ width: `${progress}%` }} />
        </div>
        <div className="status">{status}</div>
      </div>

      <div className="card wide">
        <h2>文件下载</h2>
        <div className="grid">
          <label>
            <span>Host</span>
            <input value={host} onChange={(e) => setHost(e.target.value)} placeholder="10.0.0.1" />
          </label>
          <label>
            <span>SSH User (optional)</span>
            <input value={sshUser} onChange={(e) => setSshUser(e.target.value)} placeholder="root" />
          </label>
          <label>
            <span>远端路径</span>
            <input value={downloadPath} onChange={(e) => setDownloadPath(e.target.value)} placeholder="/data/file.zip" />
          </label>
        </div>
        <button className="ghost" onClick={startDownload} disabled={!token || !host || !downloadPath}>
          下载
        </button>
      </div>
    </div>
  )
}
