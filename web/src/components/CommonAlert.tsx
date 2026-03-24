import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import { useAlertStore } from '../store/alert'

export const CommonAlert = () => {
  const open = useAlertStore((s) => s.open)
  const kind = useAlertStore((s) => s.kind)
  const tone = useAlertStore((s) => s.tone)
  const title = useAlertStore((s) => s.title)
  const message = useAlertStore((s) => s.message)
  const confirmText = useAlertStore((s) => s.confirmText)
  const cancelText = useAlertStore((s) => s.cancelText)
  const close = useAlertStore((s) => s.close)

  useEffect(() => {
    if (!open) return

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      close(false)
    }

    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [close, open])

  if (!open) return null

  return createPortal(
    <div className="modal-backdrop" onClick={() => close(false)}>
      <div className={`modal alert-modal ${tone === 'danger' ? 'danger' : ''}`} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div>
            <h3>{title}</h3>
            <p>{kind === 'confirm' ? '请确认后继续执行。' : '请确认提示内容。'}</p>
          </div>
        </div>
        <div className="modal-body">
          <div className="alert-modal-message">{message}</div>
        </div>
        <div className="modal-actions">
          {kind === 'confirm' && (
            <button className="ghost" onClick={() => close(false)}>
              {cancelText}
            </button>
          )}
          <button className={tone === 'danger' ? 'ghost danger' : 'primary'} onClick={() => close(true)}>
            {confirmText}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  )
}
