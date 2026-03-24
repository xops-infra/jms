import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { CommonAlert } from './components/CommonAlert.tsx'
import './index.css'
import App from './App.tsx'
import { setupAuthStoreSync } from './store/auth'

setupAuthStoreSync()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <>
      <App />
      <CommonAlert />
    </>
  </StrictMode>,
)
