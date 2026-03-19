import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiClient } from '../api/client'
import { useAuthStore } from '../store/auth'

type LoginResponse = {
  token: string
  expires_at: number
}

export const Login = () => {
  const navigate = useNavigate()
  const setSession = useAuthStore((s) => s.setSession)
  const [user, setUser] = useState('')
  const [password, setPassword] = useState('')
  const [mode, setMode] = useState<'db' | 'ad'>('ad')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const url = mode === 'ad' ? '/api/v1/login/ad' : '/api/v1/login'
      const res = await apiClient.post<LoginResponse>(url, { user, password })
      setSession(res.data.token, user, mode)
      navigate('/terminal')
    } catch (err: any) {
      setError(err?.response?.data || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="page login">
      <div className="card">
        <div className="brand">
          <div className="logo">JMS</div>
          <div>
            <h1>PatsnapJMS</h1>
            <p>登录后进入统一终端与文件传输控制台</p>
          </div>
        </div>

        <form onSubmit={onSubmit} className="form">
          <label>
            <span>登录方式</span>
            <select value={mode} onChange={(e) => setMode(e.target.value as 'db' | 'ad')}>
              <option value="ad">LDAP/AD</option>
              <option value="db">数据库用户</option>
            </select>
          </label>

          <label>
            <span>用户名</span>
            <input value={user} onChange={(e) => setUser(e.target.value)} placeholder="请输入用户名" />
          </label>

          <label>
            <span>密码</span>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="请输入密码" />
          </label>

          {error && <div className="error">{error}</div>}

          <button className="primary" disabled={loading}>
            {loading ? '登录中…' : '登录'}
          </button>
        </form>

        <div className="hint">
          默认账号（无 DB 时）：<code>jms</code> / <code>jms</code>
        </div>
      </div>
    </div>
  )
}
