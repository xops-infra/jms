import { create } from 'zustand'

type AuthState = {
  token: string | null
  setToken: (token: string) => void
  clear: () => void
}

const storageKey = 'jms_token'

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem(storageKey),
  setToken: (token) => {
    localStorage.setItem(storageKey, token)
    set({ token })
  },
  clear: () => {
    localStorage.removeItem(storageKey)
    set({ token: null })
  },
}))

