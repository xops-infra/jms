import { useCallback, useEffect, useState } from 'react'

/**
 * 网页全屏：用 fixed 铺满视口（仍保留浏览器标签栏/地址栏），不调用 requestFullscreen。
 * 退出：按钮，或 Shift+Esc（避免占用终端里单独的 Esc）。
 */
export function usePageFullscreen() {
  const [active, setActive] = useState(false)

  const toggle = useCallback(() => {
    setActive((prev) => !prev)
  }, [])

  useEffect(() => {
    if (!active) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [active])

  useEffect(() => {
    if (!active) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape' || !e.shiftKey) return
      e.preventDefault()
      setActive(false)
    }
    document.addEventListener('keydown', onKey, true)
    return () => document.removeEventListener('keydown', onKey, true)
  }, [active])

  return { isPageFullscreen: active, setPageFullscreen: setActive, togglePageFullscreen: toggle }
}
