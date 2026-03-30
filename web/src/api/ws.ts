const resolveWsBase = () => {
  const explicitWsBase = import.meta.env.VITE_WS_BASE
  if (explicitWsBase) return explicitWsBase

  const explicitApiBase = import.meta.env.VITE_API_BASE
  if (explicitApiBase) return explicitApiBase

  // In the default docker-compose deployment, web serves on 8080 and api on 8013.
  // Connect terminal WS directly to the API port to avoid intermediate proxies dropping Upgrade headers.
  if (window.location.port === '8080') {
    return `${window.location.protocol}//${window.location.hostname}:8013`
  }

  return window.location.origin
}

export const buildWsUrl = (path: string, params: Record<string, string | number | undefined>) => {
  const base = resolveWsBase()
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
