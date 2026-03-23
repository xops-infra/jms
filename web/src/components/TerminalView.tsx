import { useEffect, useRef } from 'react'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { buildWsUrl } from '../api/ws'
import { apiClient } from '../api/client'

export type TerminalViewProps = {
  active: boolean
  host: string
  user?: string
  keyName?: string
  token: string
  sessionId?: string
  onSessionId?: (id: string) => void
  onStateChange?: (event: TerminalStateEvent) => void
}

export type TerminalStatePhase = 'connecting' | 'live' | 'closed' | 'disconnected'

export type TerminalStateEvent = {
  phase: TerminalStatePhase
  reason?: string
}

type WsMessage = {
  type: string
  data?: string
  cols?: number
  rows?: number
  session_id?: string
}

type TerminalAttemptFailure = {
  stage: string
  status: number
  message: string
}

const createAttemptId = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `attempt-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

const formatFailureReason = (failure?: TerminalAttemptFailure | null) => {
  if (!failure?.message) return ''
  return `连接失败（${failure.stage}）: ${failure.message}`
}

export const TerminalView = ({
  active,
  host,
  user,
  keyName,
  token,
  sessionId,
  onSessionId,
  onStateChange,
}: TerminalViewProps) => {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const pingRef = useRef<number | null>(null)
  const sessionIdRef = useRef(sessionId)
  const onSessionIdRef = useRef(onSessionId)
  const onStateChangeRef = useRef(onStateChange)

  useEffect(() => {
    sessionIdRef.current = sessionId
  }, [sessionId])

  useEffect(() => {
    onSessionIdRef.current = onSessionId
  }, [onSessionId])

  useEffect(() => {
    onStateChangeRef.current = onStateChange
  }, [onStateChange])

  useEffect(() => {
    if (!containerRef.current || termRef.current) return
    const term = new Terminal({
      fontFamily: '"Fira Code", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      fontSize: 13,
      theme: {
        background: '#0b1220',
        foreground: '#e2e8f0',
        cursor: '#f59e0b',
      },
      cursorBlink: true,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(containerRef.current)
    fit.fit()
    termRef.current = term
    fitRef.current = fit

    const handleResize = () => {
      fit.fit()
      const cols = term.cols
      const rows = term.rows
      wsRef.current?.send(JSON.stringify({ type: 'resize', cols, rows }))
    }

    window.addEventListener('resize', handleResize)
    return () => {
      window.removeEventListener('resize', handleResize)
      term.dispose()
      termRef.current = null
      fitRef.current = null
    }
  }, [])

  useEffect(() => {
    const term = termRef.current
    if (!term) return

    if (!active || !host || !token) {
      wsRef.current?.close()
      wsRef.current = null
      return
    }

    const cols = term.cols || 120
    const rows = term.rows || 32
    const attemptId = createAttemptId()
    const wsUrl = buildWsUrl('/api/v1/terminal/ws', {
      host,
      user,
      key: keyName,
      cols,
      rows,
      session_id: sessionIdRef.current,
      attempt_id: attemptId,
      token,
    })
    let closedByEffect = false
    let receivedExit = false
    let opened = false

    onStateChangeRef.current?.({ phase: 'connecting' })
    term.writeln('\u001b[38;5;208mConnecting...\u001b[0m')

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      opened = true
      onStateChangeRef.current?.({ phase: 'live' })
      term.writeln('\u001b[38;5;82mConnected\u001b[0m')
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data) as WsMessage
        if (msg.type === 'data' && msg.data) {
          term.write(msg.data)
        }
        if (msg.type === 'session' && msg.session_id) {
          onSessionIdRef.current?.(msg.session_id)
        }
        if (msg.type === 'exit') {
          receivedExit = true
          onStateChangeRef.current?.({
            phase: 'closed',
            reason: msg.data?.trim() || '远端 shell 已退出。',
          })
          term.writeln('\r\n\u001b[38;5;208mSession closed\u001b[0m')
        }
      } catch {
        term.write(String(event.data))
      }
    }

    ws.onclose = async (event) => {
      if (closedByEffect || receivedExit) return

      let reason = event.reason?.trim() || ''
      if (!opened) {
        try {
          const res = await apiClient.get<TerminalAttemptFailure>(`/api/v1/terminal/errors/${encodeURIComponent(attemptId)}`)
          reason = formatFailureReason(res.data) || reason
        } catch {
          // Keep fallback below when server has no recorded detail.
        }
      }
      if (!reason) {
        reason = opened
          ? '终端链路已关闭，可重新连接继续操作。'
          : '终端连接建立失败，请检查网络、代理和目标主机配置。'
      }

      onStateChangeRef.current?.({ phase: 'disconnected', reason })
      term.writeln(`\r\n\u001b[38;5;208m${reason}\u001b[0m`)
    }

    const dispose = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }))
      }
    })

    pingRef.current = window.setInterval(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, 20000)

    return () => {
      closedByEffect = true
      dispose.dispose()
      if (pingRef.current) {
        window.clearInterval(pingRef.current)
        pingRef.current = null
      }
      ws.close()
    }
  }, [active, host, user, keyName, token])

  return <div className="terminal" ref={containerRef} />
}
