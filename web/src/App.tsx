import { useEffect, type ReactNode } from 'react'
import { HashRouter, NavLink, Navigate, Route, Routes } from 'react-router-dom'
import { apiClient } from './api/client'
import { Login } from './pages/Login'
import { AdminPolicyPage } from './pages/AdminPolicy'
import { AdminShellPage } from './pages/AdminShell'
import { AdminAuditPage } from './pages/AdminAudit'
import { TerminalPage } from './pages/Terminal'
import { WorkspacePage } from './pages/Workspace'
import { useAuthStore } from './store/auth'

type CurrentUserResponse = {
  groups?: string[]
}

const Nav = () => {
  const token = useAuthStore((s) => s.token)
  const user = useAuthStore((s) => s.user)
  const mode = useAuthStore((s) => s.mode)
  const groups = useAuthStore((s) => s.groups)
  const clear = useAuthStore((s) => s.clear)
  const isAdmin = Boolean(groups?.includes('admin'))
  return (
    <header className="topbar">
      <div className="brand">
        <div className="logo">JMS</div>
        <div>
          <strong>JMS</strong>
          <span>Secure Access Console</span>
        </div>
      </div>
      {token && (
        <nav className="topnav" aria-label="主导航">
          <NavLink to="/terminal" className={({ isActive }) => `topnav-link${isActive ? ' active' : ''}`}>
            终端首页
          </NavLink>
          {isAdmin && (
            <NavLink to="/admin/policy" className={({ isActive }) => `topnav-link${isActive ? ' active' : ''}`}>
              Policy 管理
            </NavLink>
          )}
          {isAdmin && (
            <NavLink to="/admin/shell" className={({ isActive }) => `topnav-link${isActive ? ' active' : ''}`}>
              ShellTask 管理
            </NavLink>
          )}
          {isAdmin && (
            <NavLink to="/admin/audit" className={({ isActive }) => `topnav-link${isActive ? ' active' : ''}`}>
              审计
            </NavLink>
          )}
        </nav>
      )}
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
  const groups = useAuthStore((s) => s.groups)
  const setGroups = useAuthStore((s) => s.setGroups)

  useEffect(() => {
    let cancelled = false
    if (!token) {
      setGroups(null)
      return
    }

    setGroups(null)
    void apiClient
      .get<CurrentUserResponse>('/api/v1/user/me')
      .then((res) => {
        if (cancelled) return
        setGroups(res.data.groups || [])
      })
      .catch(() => {
        if (cancelled) return
        setGroups([])
      })

    return () => {
      cancelled = true
    }
  }, [setGroups, token])

  const RequireAuth = ({ children }: { children: ReactNode }) => {
    if (!token) {
      return <Navigate to="/login" replace />
    }
    return <>{children}</>
  }

  const RequireAdmin = ({ children }: { children: ReactNode }) => {
    if (!token) {
      return <Navigate to="/login" replace />
    }
    if (groups === null) {
      return (
        <div className="page console-page">
          <div className="workspace-state-shell">
            <div className="panel workspace-state-card">
              <div className="panel-header">
                <div>
                  <h3>校验管理员权限</h3>
                  <p>正在确认当前账号是否属于 admin 组。</p>
                </div>
              </div>
              <div className="panel-body">
                <div className="empty-state workspace-state-message">请稍候...</div>
              </div>
            </div>
          </div>
        </div>
      )
    }
    if (!groups.includes('admin')) {
      return <Navigate to="/terminal" replace />
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
        <Route
          path="/workspace"
          element={
            <RequireAuth>
              <WorkspacePage />
            </RequireAuth>
          }
        />
        <Route
          path="/admin/policy"
          element={
            <RequireAdmin>
              <AdminPolicyPage />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/shell"
          element={
            <RequireAdmin>
              <AdminShellPage />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/audit"
          element={
            <RequireAdmin>
              <AdminAuditPage />
            </RequireAdmin>
          }
        />
      </Routes>
    </HashRouter>
  )
}

export default App
