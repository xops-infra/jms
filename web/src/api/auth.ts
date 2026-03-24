import { useAuthStore } from '../store/auth'

const loginHash = '#/login'

const redirectToLogin = () => {
  useAuthStore.getState().clear()

  if (window.location.hash !== loginHash) {
    window.location.hash = loginHash
  }
}

export const handleUnauthorizedStatus = (status?: number) => {
  if (status !== 401) return false
  redirectToLogin()
  return true
}

export const apiFetch = async (input: RequestInfo | URL, init?: RequestInit) => {
  const response = await fetch(input, init)
  handleUnauthorizedStatus(response.status)
  return response
}
