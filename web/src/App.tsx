import { useEffect, useState, type ReactNode } from 'react'
import { api, ApiError, message, type User } from './api'
import { AuthScreen } from './AuthScreen'
import { Brand, Icon } from './icons'

export type View = 'history' | 'pending' | 'connections'
export function readView(): View { const value = new URLSearchParams(location.search).get('view'); return value === 'pending' || value === 'connections' ? value : 'history' }
function readInvite() { const token = new URLSearchParams(location.hash.slice(1)).get('ativar'); if (token) history.replaceState(null, '', location.pathname + location.search); return token }
const initialInvite = readInvite()

export function Shell({ user, view, onView, onLogout, children }: { user: User; view: View; onView: (view: View) => void; onLogout: () => Promise<void>; children: ReactNode }) {
  const [logoutError, setLogoutError] = useState(''), [leaving, setLeaving] = useState(false)
  const nav = [{ id: 'history' as const, label: 'Histórico', icon: 'history' as const }, { id: 'pending' as const, label: 'Para avaliar', icon: 'star' as const }, { id: 'connections' as const, label: 'Conexões', icon: 'link' as const }]
  async function leave() { setLeaving(true); setLogoutError(''); try { await onLogout() } catch (err) { setLogoutError(message(err)) } finally { setLeaving(false) } }
  return <div className="app-layout"><aside className="sidebar"><Brand /><p className="nav-label">NOSSO CANTINHO</p><nav className="desktop-nav" aria-label="Navegação principal">{nav.map(item => <button key={item.id} className={`nav-item ${view === item.id ? 'active' : ''}`} aria-current={view === item.id ? 'page' : undefined} onClick={() => onView(item.id)}><Icon name={item.icon} /><span>{item.label}</span>{view === item.id && <span className="nav-active-dot" />}</button>)}</nav><div className="sidebar-bottom"><p className="sidebar-note">Boas histórias.<br /><em>Melhores lembranças.</em></p><div className="account"><span className="avatar">{user.name.slice(0, 1)}</span><span className="account-name"><strong>{user.name}</strong><small>Sua conta pessoal</small></span><button className="icon-button" onClick={leave} disabled={leaving} aria-label="Sair da conta"><Icon name="logout" size={18} /></button></div></div></aside>
    <header className="mobile-header"><Brand /><button className="mobile-account" onClick={leave} disabled={leaving} aria-label="Sair da conta"><span className="avatar">{user.name.slice(0, 1)}</span><Icon name="logout" size={17} /></button></header>
    <main className="app-main">{logoutError && <p role="alert" className="notice error">{logoutError}</p>}{children}</main>
    <nav className="mobile-nav" aria-label="Navegação no celular">{nav.map(item => <button key={item.id} className={view === item.id ? 'active' : ''} aria-current={view === item.id ? 'page' : undefined} onClick={() => onView(item.id)}><Icon name={item.icon} size={21} /><span>{item.label}</span></button>)}</nav>
  </div>
}
export function App() {
  const [invite, setInvite] = useState(initialInvite), [user, setUser] = useState<User | null>(null), [loading, setLoading] = useState(!initialInvite)
  const [error, setError] = useState(''), [retry, setRetry] = useState(0), [view, setView] = useState<View>(readView)
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
  return <Shell user={user} view={view} onView={navigate} onLogout={logout}><header className="page-heading"><p className="eyebrow">UM REGISTRO A DOIS</p><h1>{view === 'history' ? 'Nosso histórico' : view === 'pending' ? 'Para avaliar' : 'Nossas conexões'}</h1><p className="muted">Um lugar para guardar o que assistimos e o que ficou.</p></header><section className="panel"><p>Seu diário está pronto para receber nossas histórias.</p></section></Shell>
}
