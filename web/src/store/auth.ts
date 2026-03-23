import { create } from 'zustand'

export type AuthMode = 'ad' | 'db'

type AuthState = {
  token: string | null
  user: string | null
  mode: AuthMode | null
  setSession: (token: string, user: string, mode: AuthMode) => void
  clear: () => void
}

const tokenKey = 'jms_token'
const userKey = 'jms_user'
const modeKey = 'jms_mode'

const readStoredSession = () => ({
  token: localStorage.getItem(tokenKey),
  user: localStorage.getItem(userKey),
  mode: (localStorage.getItem(modeKey) as AuthMode | null) || null,
})

export const useAuthStore = create<AuthState>((set) => ({
  ...readStoredSession(),
  setSession: (token, user, mode) => {
    localStorage.setItem(tokenKey, token)
    localStorage.setItem(userKey, user)
    localStorage.setItem(modeKey, mode)
    set({ token, user, mode })
  },
  clear: () => {
    localStorage.removeItem(tokenKey)
    localStorage.removeItem(userKey)
    localStorage.removeItem(modeKey)
    set({ token: null, user: null, mode: null })
  },
}))

let syncBound = false

export const setupAuthStoreSync = () => {
  if (syncBound || typeof window === 'undefined') return
  syncBound = true
  window.addEventListener('storage', (event) => {
    if (event.key && ![tokenKey, userKey, modeKey].includes(event.key)) return
    useAuthStore.setState(readStoredSession())
  })
}
