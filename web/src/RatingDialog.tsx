import { useEffect, useRef, useState, type FormEvent } from 'react'
import { api, message, scoreLabel, serviceName, type Person, type RatingSummary, type User, type WatchItem } from './api'
import { Icon } from './icons'
import { Poster } from './Poster'

export function RatingDialog({ item, user, people, onClose, onSaved }: { item: WatchItem; user: User; people: Person[]; onClose: () => void; onSaved: (summary: RatingSummary) => void }) {
  const own = item.ratings.find(rating => rating.user_id === user.id)
  const other = item.ratings.find(rating => rating.user_id !== user.id)
  const partner = people.find(person => person.id !== user.id)
  const [score, setScore] = useState(own ? String(own.score) : '')
  const [comment, setComment] = useState(own?.comment ?? '')
  const [busy, setBusy] = useState(false), [error, setError] = useState('')
  const dialog = useRef<HTMLDialogElement>(null), input = useRef<HTMLInputElement>(null)
  useEffect(() => {
    const opener = document.activeElement as HTMLElement | null
    dialog.current?.showModal(); document.body.classList.add('modal-open'); input.current?.focus()
    return () => { dialog.current?.close(); document.body.classList.remove('modal-open'); if (opener?.isConnected) opener.focus(); else (document.querySelector('.page-heading h1') as HTMLElement | null)?.focus() }
  }, [])
  const length = Array.from(comment).length
  async function save(event: FormEvent) {
    event.preventDefault()
    if (score === '' || !Number.isFinite(Number(score)) || Number(score) < 0 || Number(score) > 10 || Number(score) % 0.5 !== 0 || length > 2000) { setError('Use uma nota de 0 a 10, em passos de 0,5, e comentário de até 2000 caracteres.'); return }
    setBusy(true); setError('')
    try { onSaved(await api<RatingSummary>(`/events/${item.id}/rating`, { method: 'PUT', body: JSON.stringify({ score: Number(score), comment }) })) } catch (err) { setError(message(err)) } finally { setBusy(false) }
  }
  async function remove() { setBusy(true); setError(''); try { onSaved(await api<RatingSummary>(`/events/${item.id}/rating`, { method: 'DELETE' })) } catch (err) { setError(message(err)) } finally { setBusy(false) } }
  return <dialog ref={dialog} className="rating-dialog" aria-labelledby="rating-title" onCancel={event => { event.preventDefault(); if (!busy) onClose() }} onClick={event => { if (event.target === dialog.current && !busy) { const rect = dialog.current.getBoundingClientRect(); if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) onClose() } }}>
    <div className="dialog-content"><button className="icon-button dialog-close" onClick={onClose} disabled={busy} aria-label="Fechar avaliação"><Icon name="close" /></button><p className="dialog-caption">Sua avaliação</p>
      <div className="dialog-heading"><Poster title={item.title} path={item.poster_path} url={item.poster_url} /><div><span className="media-label">{item.media_type === 'tv' ? 'SÉRIE' : item.media_type === 'movie' ? 'FILME' : 'VÍDEO'}{item.year ? ` · ${item.year}` : ''} · {serviceName(item.service)}</span><h2 id="rating-title">{item.title}</h2><p className="muted">Sua opinião, {user.name}.</p>{item.genres.length > 0 && <p className="genre-list">{item.genres.slice(0, 3).join(' · ')}</p>}</div></div>
      {item.overview && <p className="dialog-overview">{item.overview}</p>}
      {item.media_type === 'tv' && <p className="rating-scope">Sua nota vale para a série inteira.</p>}
      <form onSubmit={save}><div className="score-section"><div><label htmlFor="rating-score">Sua nota</label><p className="muted">Quanto essa história ficou com você?</p></div><div className="score-display"><span>{score === '' ? '—' : scoreLabel(Number(score))}</span><small>/ 10</small></div></div><div className="score-controls"><div className="range-wrap"><input type="range" aria-label="Controle de nota" min="0" max="10" step="0.5" value={score === '' ? 5 : Number(score)} onChange={event => setScore(event.target.value)} disabled={busy} /><div className="range-labels"><span>0</span><span>5</span><span>10</span></div></div><input ref={input} id="rating-score" type="number" min="0" max="10" step="0.5" inputMode="decimal" value={score} onChange={event => setScore(event.target.value)} placeholder="Nota" required disabled={busy} /></div><p className="score-hint">Arraste ou digite. Notas de meio em meio.</p>
        <label className="field comment-field">O que você achou? <span className="optional">Opcional</span><textarea rows={3} value={comment} onChange={event => setComment(event.target.value)} placeholder="Uma cena que marcou, uma impressão, uma lembrança…" disabled={busy} /></label><p className={`comment-count ${length > 2000 ? 'over-limit' : ''}`}>{length}/2000</p>
        <aside className="partner-note"><span className="avatar">{(other?.name ?? partner?.name ?? '?').slice(0, 1)}</span><div><strong>{other?.name ?? partner?.name ?? 'Outra pessoa'}</strong><p>{other ? other.comment || 'Sem comentário por enquanto.' : 'Ainda não deixou uma nota para esta história.'}</p></div><span className="partner-score">{other ? scoreLabel(other.score) : '—'}{other && <small>/10</small>}</span></aside>
        {error && <p className="notice error" role="alert">{error}</p>}
        <div className="dialog-actions">{own && <button type="button" className="remove-rating" onClick={remove} disabled={busy}><Icon name="trash" size={15} /> Remover minha nota</button>}<button type="button" className="button subtle cancel-rating" onClick={onClose} disabled={busy}>Cancelar</button><button className="button primary save-rating" type="submit" disabled={busy || score === '' || length > 2000}>{busy ? 'Salvando…' : 'Salvar nota'}{!busy && <Icon name="check" size={17} />}</button></div>
      </form>
    </div>
  </dialog>
}
