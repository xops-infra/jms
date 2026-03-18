import { useEffect, useRef } from 'react'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { buildWsUrl } from '../api/ws'

export type TerminalViewProps = {
  active: boolean
  host: string
  user?: string
  keyName?: string
  token: string
  sessionId?: string
  onSessionId?: (id: string) => void
}

type WsMessage = {
  type: string
  data?: string
  cols?: number
  rows?: number
  session_id?: string
}

export const TerminalView = ({
  active,
  host,
  user,
  keyName,
  token,
  sessionId,
  onSessionId,
}: TerminalViewProps) => {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const pingRef = useRef<number | null>(null)

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
    const wsUrl = buildWsUrl('/api/v1/terminal/ws', {
      host,
      user,
      key: keyName,
      cols,
      rows,
      session_id: sessionId,
      token,
    })

    term.writeln('\u001b[38;5;208mConnecting...\u001b[0m')

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      term.writeln('\u001b[38;5;82mConnected\u001b[0m')
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data) as WsMessage
        if (msg.type === 'data' && msg.data) {
          term.write(msg.data)
        }
        if (msg.type === 'session' && msg.session_id) {
          onSessionId?.(msg.session_id)
        }
        if (msg.type === 'exit') {
          term.writeln('\r\n\u001b[38;5;208mSession closed\u001b[0m')
        }
      } catch {
        term.write(String(event.data))
      }
    }

    ws.onclose = () => {
      term.writeln('\r\n\u001b[38;5;208mDisconnected\u001b[0m')
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
      dispose.dispose()
      if (pingRef.current) {
        window.clearInterval(pingRef.current)
      }
      ws.close()
    }
  }, [active, host, user, token, sessionId, onSessionId])

  return <div className="terminal" ref={containerRef} />
}
