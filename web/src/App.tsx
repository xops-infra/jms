import { HashRouter, NavLink, Route, Routes } from 'react-router-dom'
import { Login } from './pages/Login'
import { TerminalPage } from './pages/Terminal'
import { FilesPage } from './pages/Files'
import { useAuthStore } from './store/auth'

const Nav = () => {
  const token = useAuthStore((s) => s.token)
  const clear = useAuthStore((s) => s.clear)
  return (
    <header className="topbar">
      <div className="brand">
        <div className="logo">JMS</div>
        <div>
          <strong>Jump Management System</strong>
          <span>Web Console</span>
        </div>
      </div>
      <nav>
        <NavLink to="/login">登录</NavLink>
        <NavLink to="/terminal">终端</NavLink>
        <NavLink to="/files">文件</NavLink>
      </nav>
      <div className="actions">
        {token ? (
          <button className="ghost" onClick={clear}>
            退出
          </button>
        ) : (
          <span className="muted">未登录</span>
        )}
      </div>
    </header>
  )
}

function App() {
  return (
    <HashRouter>
      <Nav />
      <Routes>
        <Route path="/" element={<Login />} />
        <Route path="/login" element={<Login />} />
        <Route path="/terminal" element={<TerminalPage />} />
        <Route path="/files" element={<FilesPage />} />
      </Routes>
    </HashRouter>
  )
}

export default App

