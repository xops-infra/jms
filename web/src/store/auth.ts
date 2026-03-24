import { create } from 'zustand'

export type AuthMode = 'ad' | 'db'

type AuthState = {
  token: string | null
  user: string | null
  mode: AuthMode | null
  groups: string[] | null
  setSession: (token: string, user: string, mode: AuthMode) => void
  setGroups: (groups: string[] | null) => void
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
  groups: null,
  setSession: (token, user, mode) => {
    localStorage.setItem(tokenKey, token)
    localStorage.setItem(userKey, user)
    localStorage.setItem(modeKey, mode)
    set({ token, user, mode, groups: null })
  },
  setGroups: (groups) => {
    set({ groups })
  },
  clear: () => {
    localStorage.removeItem(tokenKey)
    localStorage.removeItem(userKey)
    localStorage.removeItem(modeKey)
    set({ token: null, user: null, mode: null, groups: null })
  },
}))

let syncBound = false

export const setupAuthStoreSync = () => {
  if (syncBound || typeof window === 'undefined') return
  syncBound = true
  window.addEventListener('storage', (event) => {
    if (event.key && ![tokenKey, userKey, modeKey].includes(event.key)) return
    useAuthStore.setState({ ...readStoredSession(), groups: null })
  })
}
