import { useEffect, useState } from 'react'

export function NetworkStatus() {
  const [offline, setOffline] = useState(!navigator.onLine)
  useEffect(() => {
    const update = () => setOffline(!navigator.onLine)
    const unavailable = () => setOffline(true), available = () => setOffline(false)
    window.addEventListener('online', update); window.addEventListener('offline', update)
    window.addEventListener('network-unavailable', unavailable); window.addEventListener('network-available', available)
    return () => { window.removeEventListener('online', update); window.removeEventListener('offline', update); window.removeEventListener('network-unavailable', unavailable); window.removeEventListener('network-available', available) }
  }, [])
  return offline ? <div className="offline-banner" role="alert">Sem conexão. O histórico não será atualizado e as notas não podem ser salvas. Reconecte e tente novamente.</div> : null
}
