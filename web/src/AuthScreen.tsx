import { useState, type FormEvent } from 'react'
import { api, message, type User } from './api'
import { Brand, Icon } from './icons'

export function AuthScreen({ invite, onAuthenticated }: { invite: string | null; onAuthenticated: (user: User) => void }) {
  const [username, setUsername] = useState(''), [password, setPassword] = useState(''), [confirmation, setConfirmation] = useState('')
  const [busy, setBusy] = useState(false), [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (busy) return
    if (invite && (Array.from(password).length < 12 || Array.from(password).length > 128)) { setError('Use uma senha de 12 a 128 caracteres.'); return }
    if (invite && password !== confirmation) { setError('As senhas precisam ser iguais.'); return }
    setError(''); setBusy(true)
    try {
      const result = await api<{ user: User }>(invite ? '/auth/activate' : '/auth/login', { method: 'POST', body: JSON.stringify(invite ? { token: invite, password } : { username, password }) })
      setPassword(''); setConfirmation(''); onAuthenticated(result.user)
    } catch (err) { setError(message(err)) } finally { setBusy(false) }
  }
  return <div className="auth-layout">
    <section className="auth-story"><Brand /><div className="auth-editorial"><h1>O que vimos.<br />O que ficou.</h1><p>Um lugar para as histórias que assistimos<br />e as opiniões de cada um.</p></div><p className="auth-caption">Nosso diário de filmes e séries.</p></section>
    <main className="auth-main"><div className="auth-form-wrap"><span className="small-mark"><Icon name={invite ? 'star' : 'play'} size={22} /></span><h2>{invite ? 'Vamos começar?' : 'Bem-vindo de volta.'}</h2><p className="muted auth-description">{invite ? 'Crie sua senha para guardar suas opiniões e descobrir o que ficou para vocês.' : 'Entre para revisitar o que assistimos e deixar sua nota.'}</p>
      <form onSubmit={submit}>
        {!invite && <label className="field">Usuário<input type="text" name="username" autoComplete="username" autoCapitalize="none" spellCheck={false} value={username} onChange={event => setUsername(event.target.value)} required minLength={3} maxLength={40} placeholder="ronald" /></label>}
        <label className="field">{invite ? 'Crie uma senha' : 'Senha'}<input type="password" name="password" autoComplete={invite ? 'new-password' : 'current-password'} value={password} onChange={event => setPassword(event.target.value)} required placeholder={invite ? 'Pelo menos 12 caracteres' : 'Sua senha'} /></label>
        {invite && <><label className="field">Confirme a senha<input type="password" name="confirmation" autoComplete="new-password" value={confirmation} onChange={event => setConfirmation(event.target.value)} required placeholder="Repita sua senha" /></label><p className="field-hint">Use entre 12 e 128 caracteres. Sua senha é só sua.</p></>}
        {error && <p className="notice error" role="alert">{error}</p>}
        <button className="button primary auth-submit" disabled={busy} type="submit">{busy ? 'Só um instante…' : invite ? 'Criar minha conta' : 'Entrar'}{!busy && <Icon name="arrow" size={18} />}</button>
      </form><p className="auth-help">{invite ? 'Este link é pessoal, de uso único e válido por uma hora.' : 'Primeiro acesso? Abra o convite pessoal gerado neste computador para criar sua senha.'}</p><div className="auth-form-footer"><span className="status-dot" /> Feito para o nosso tempo juntos.</div>
    </div></main>
  </div>
}
