import { useState } from 'react'
import { TerminalView } from '../components/TerminalView'
import { useAuthStore } from '../store/auth'

export const TerminalPage = () => {
  const token = useAuthStore((s) => s.token)
  const [host, setHost] = useState('')
  const [sshUser, setSshUser] = useState('')
  const [active, setActive] = useState(false)
  const [sessionId, setSessionId] = useState('')

  return (
    <div className="page terminal-page">
      <div className="toolbar">
        <div className="field">
          <label>Host</label>
          <input value={host} onChange={(e) => setHost(e.target.value)} placeholder="10.0.0.1" />
        </div>
        <div className="field">
          <label>SSH User (optional)</label>
          <input value={sshUser} onChange={(e) => setSshUser(e.target.value)} placeholder="root" />
        </div>
        <div className="field">
          <label>Session ID</label>
          <input value={sessionId} onChange={(e) => setSessionId(e.target.value)} placeholder="auto" />
        </div>
        <div className="actions">
          <button className="primary" onClick={() => setActive(true)} disabled={!token || !host}>
            Connect
          </button>
          <button className="ghost" onClick={() => setActive(false)}>
            Disconnect
          </button>
        </div>
      </div>

      <div className="terminal-wrap">
        {token ? (
          <TerminalView
            active={active}
            host={host}
            user={sshUser || undefined}
            token={token}
            sessionId={sessionId || undefined}
            onSessionId={(id) => setSessionId(id)}
          />
        ) : (
          <div className="empty">请先登录</div>
        )}
      </div>
    </div>
  )
}

