import { useEffect, useState, type ReactNode } from 'react'
import { api, ApiError, message, type User, type Connection } from './api'
import { AuthScreen } from './AuthScreen'
import { Brand, Icon } from './icons'
import { Journal } from './Journal'
import { Connections } from './Connections'

export type View = 'history' | 'pending' | 'connections'
export function readView(): View { const value = new URLSearchParams(location.search).get('view'); return value === 'pending' || value === 'connections' ? value : 'history' }
function readInvite() { const token = new URLSearchParams(location.hash.slice(1)).get('ativar'); if (token) history.replaceState(null, '', location.pathname + location.search); return token }
const initialInvite = readInvite()

export function Shell({ user, view, attention, onView, onLogout, children }: { user: User; view: View; attention: number; onView: (view: View) => void; onLogout: () => Promise<void>; children: ReactNode }) {
  const [logoutError, setLogoutError] = useState(''), [leaving, setLeaving] = useState(false)
  const nav = [{ id: 'history' as const, label: 'Histórico', icon: 'history' as const }, { id: 'pending' as const, label: 'Para avaliar', icon: 'star' as const }, { id: 'connections' as const, label: 'Conexões', icon: 'link' as const }]
  async function leave() { setLeaving(true); setLogoutError(''); try { await onLogout() } catch (err) { setLogoutError(message(err)) } finally { setLeaving(false) } }
  return <div className="app-layout"><a className="skip-link" href="#conteudo">Pular para o conteúdo</a><header className="app-header"><div className="topbar"><Brand /><nav className="desktop-nav" aria-label="Navegação principal">{nav.map(item => <button key={item.id} className={`nav-item ${view === item.id ? 'active' : ''}`} aria-current={view === item.id ? 'page' : undefined} onClick={() => onView(item.id)}><span>{item.label}</span></button>)}</nav><div className="account"><span className="avatar">{user.name.slice(0, 1)}</span><span className="account-name"><strong>{user.name}</strong></span><button className="icon-button" onClick={leave} disabled={leaving} aria-label="Sair da conta"><Icon name="logout" size={18} /></button></div></div></header>
    <main id="conteudo" tabIndex={-1} className="app-main">{logoutError && <p role="alert" className="notice error">{logoutError}</p>}{attention > 0 && view !== 'connections' && <aside className="connection-notice"><Icon name="alert" size={16} /><span>{attention === 1 ? 'Uma conexão precisa' : 'Algumas conexões precisam'} da sua atenção.</span><button onClick={() => onView('connections')}>Ver conexões <Icon name="arrow" size={14} /></button></aside>}{children}</main>
    <nav className="mobile-nav" aria-label="Navegação no celular">{nav.map(item => <button key={item.id} className={view === item.id ? 'active' : ''} aria-current={view === item.id ? 'page' : undefined} onClick={() => onView(item.id)}><Icon name={item.icon} size={21} /><span>{item.label}</span></button>)}</nav>
  </div>
}
export function App() {
  const [invite, setInvite] = useState(initialInvite), [user, setUser] = useState<User | null>(null), [loading, setLoading] = useState(!initialInvite)
  const [error, setError] = useState(''), [retry, setRetry] = useState(0), [view, setView] = useState<View>(readView)
  const [attention, setAttention] = useState(0)
  useEffect(() => { if (!user) { setAttention(0); return }; const controller = new AbortController(); api<{ connections: Connection[] }>('/connections', { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setAttention(value.connections.filter(item => item.status === 'needs_login' || item.status === 'error').length) }).catch(() => {}); return () => controller.abort() }, [user?.id])
  useEffect(() => {
    if (invite) return
    const controller = new AbortController(); setLoading(true); setError('')
    api<{ user: User }>('/auth/me', { signal: controller.signal }).then(result => setUser(result.user)).catch(err => { if (!controller.signal.aborted && !(err instanceof ApiError && err.status === 401)) setError(message(err)) }).finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [invite, retry])
  useEffect(() => {
    const expired = () => setUser(null), back = () => setView(readView())
    const invited = () => { const token = readInvite(); if (token) { setUser(null); setInvite(token); setLoading(false); setError('') } }
    window.addEventListener('session-expired', expired); window.addEventListener('popstate', back); window.addEventListener('hashchange', invited)
    return () => { window.removeEventListener('session-expired', expired); window.removeEventListener('popstate', back); window.removeEventListener('hashchange', invited) }
  }, [])
  function navigate(next: View) { const params = new URLSearchParams(location.search); if (next === 'history') params.delete('view'); else params.set('view', next); history.pushState(null, '', location.pathname + (params.size ? `?${params}` : '')); setView(next) }
  async function logout() { await api<void>('/auth/logout', { method: 'POST' }); setUser(null); setView('history'); history.replaceState(null, '', location.pathname) }
  if (loading) return <div className="startup"><Brand /><p role="status">Preparando nosso diário…</p></div>
  if (error && !user) return <div className="startup"><Brand /><p role="alert" className="notice error">{error}</p><button className="button primary" onClick={() => setRetry(value => value + 1)}>Tentar novamente</button></div>
  if (invite || !user) return <AuthScreen invite={invite} onAuthenticated={value => { setUser(value); setInvite(null) }} />
  return <Shell user={user} view={view} attention={attention} onView={navigate} onLogout={logout}>{view === 'connections' ? <Connections /> : <Journal key={view} user={user} pending={view === 'pending'} />}</Shell>
}
