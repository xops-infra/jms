export const buildWsUrl = (path: string, params: Record<string, string | number | undefined>) => {
  const base = import.meta.env.VITE_API_BASE || window.location.origin
  const url = new URL(path, base)
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') {
      url.searchParams.set(k, String(v))
    }
  })
  const proto = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.protocol = proto
  return url.toString()
}

