import axios from 'axios'
import { handleUnauthorizedStatus } from './auth'
import { useAuthStore } from '../store/auth'

const baseURL = import.meta.env.VITE_API_BASE || window.location.origin

export const apiClient = axios.create({
  baseURL,
})

apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    handleUnauthorizedStatus(error?.response?.status)
    return Promise.reject(error)
  },
)
