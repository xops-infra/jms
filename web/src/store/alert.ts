import { create } from 'zustand'

type AlertTone = 'default' | 'danger'
type AlertKind = 'alert' | 'confirm'

type AlertDialogState = {
  open: boolean
  kind: AlertKind
  tone: AlertTone
  title: string
  message: string
  confirmText: string
  cancelText: string
  resolver: ((accepted: boolean) => void) | null
  showAlert: (input: {
    title?: string
    message: string
    tone?: AlertTone
    confirmText?: string
  }) => Promise<void>
  showConfirm: (input: {
    title?: string
    message: string
    tone?: AlertTone
    confirmText?: string
    cancelText?: string
  }) => Promise<boolean>
  close: (accepted: boolean) => void
}

const defaultTitle = '系统提示'

export const useAlertStore = create<AlertDialogState>((set, get) => ({
  open: false,
  kind: 'alert',
  tone: 'default',
  title: defaultTitle,
  message: '',
  confirmText: '确定',
  cancelText: '取消',
  resolver: null,
  showAlert: async ({ title, message, tone = 'default', confirmText = '确定' }) =>
    new Promise<void>((resolve) => {
      set({
        open: true,
        kind: 'alert',
        tone,
        title: title || defaultTitle,
        message,
        confirmText,
        cancelText: '取消',
        resolver: (accepted) => {
          if (accepted) resolve()
        },
      })
    }),
  showConfirm: async ({ title, message, tone = 'default', confirmText = '确定', cancelText = '取消' }) =>
    new Promise<boolean>((resolve) => {
      set({
        open: true,
        kind: 'confirm',
        tone,
        title: title || defaultTitle,
        message,
        confirmText,
        cancelText,
        resolver: resolve,
      })
    }),
  close: (accepted) => {
    const { resolver } = get()
    set({
      open: false,
      kind: 'alert',
      tone: 'default',
      title: defaultTitle,
      message: '',
      confirmText: '确定',
      cancelText: '取消',
      resolver: null,
    })
    resolver?.(accepted)
  },
}))
