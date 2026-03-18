import type { ReactNode } from 'react'
import { HashRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Login } from './pages/Login'
import { TerminalPage } from './pages/Terminal'
import { useAuthStore } from './store/auth'

const Nav = () => {
  const token = useAuthStore((s) => s.token)
  const user = useAuthStore((s) => s.user)
  const mode = useAuthStore((s) => s.mode)
  const clear = useAuthStore((s) => s.clear)
  return (
    <header className="topbar">
      <div className="brand">
        <div className="logo">PJ</div>
        <div>
          <strong>PatsnapJMS</strong>
          <span>Secure Access Console</span>
        </div>
      </div>
      <div className="actions">
        <div className={`status-pill ${token ? 'online' : 'offline'}`}>
          <span className="dot" />
          {token ? (
            <span>
              {mode?.toUpperCase() || 'AUTH'}
              {user ? ` · ${user}` : ' · 已登录'}
            </span>
          ) : (
            <span>未登录</span>
          )}
        </div>
        {token && (
          <button className="ghost" onClick={clear}>
            退出
          </button>
        )}
      </div>
    </header>
  )
}

function App() {
  const token = useAuthStore((s) => s.token)
  const RequireAuth = ({ children }: { children: ReactNode }) => {
    if (!token) {
      return <Navigate to="/login" replace />
    }
    return <>{children}</>
  }

  return (
    <HashRouter>
      <Nav />
      <Routes>
        <Route path="/" element={token ? <Navigate to="/terminal" replace /> : <Navigate to="/login" replace />} />
        <Route path="/login" element={<Login />} />
        <Route
          path="/terminal"
          element={
            <RequireAuth>
              <TerminalPage />
            </RequireAuth>
          }
        />
      </Routes>
    </HashRouter>
  )
}

export default App
