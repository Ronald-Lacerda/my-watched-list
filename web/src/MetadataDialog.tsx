import { useEffect, useRef, useState, type FormEvent } from 'react'
import { api, message, type MetadataCandidate, type WatchItem } from './api'
import { Icon } from './icons'
import { Poster } from './Poster'

export function MetadataDialog({ item, onClose, onSaved }: { item: WatchItem; onClose: () => void; onSaved: () => void }) {
  const [query, setQuery] = useState(item.title.replace(/(?:\.{3}|…)$/, '').trim())
  const [kind, setKind] = useState(item.media_type === 'tv' ? 'tv' : item.media_type === 'movie' ? 'movie' : '')
  const [items, setItems] = useState<MetadataCandidate[]>([]), [searched, setSearched] = useState(false)
  const [searching, setSearching] = useState(false), [saving, setSaving] = useState(false), [error, setError] = useState('')
  const dialog = useRef<HTMLDialogElement>(null), input = useRef<HTMLInputElement>(null), request = useRef<AbortController | null>(null)
  useEffect(() => {
    const opener = document.activeElement as HTMLElement | null
    dialog.current?.showModal(); document.body.classList.add('modal-open'); input.current?.focus()
    return () => { request.current?.abort(); dialog.current?.close(); document.body.classList.remove('modal-open'); if (opener?.isConnected) opener.focus(); else (document.querySelector('.page-heading h1') as HTMLElement | null)?.focus() }
  }, [])
  async function search(event: FormEvent) {
    event.preventDefault(); if (Array.from(query.trim()).length < 2 || Array.from(query.trim()).length > 200) { setError('Digite de 2 a 200 caracteres para buscar.'); return }
    request.current?.abort(); const controller = new AbortController(); request.current = controller
    setSearching(true); setError(''); setItems([]); setSearched(false)
    try { const result = await api<{ items: MetadataCandidate[] }>(`/metadata/search?${new URLSearchParams({ query: query.trim(), ...(kind ? { type: kind } : {}) })}`, { signal: controller.signal }); if (!controller.signal.aborted) { setItems(result.items); setSearched(true) } }
    catch (err) { if (!controller.signal.aborted) setError(message(err)) }
    finally { if (!controller.signal.aborted) setSearching(false) }
  }
  async function confirm(candidate?: MetadataCandidate) {
    request.current?.abort(); const controller = new AbortController(); request.current = controller; setSearching(false); setSaving(true); setError('')
    try { await api(`/events/${item.id}/metadata`, { method: 'PUT', body: JSON.stringify(candidate ? { provider: 'tmdb', tmdb_id: candidate.id, media_type: candidate.media_type } : { provider: 'netflix' }), signal: controller.signal }); if (!controller.signal.aborted) onSaved() }
    catch (err) { if (!controller.signal.aborted) setError(message(err)) }
    finally { if (!controller.signal.aborted) setSaving(false) }
  }
  function edited() { request.current?.abort(); setSearching(false); setItems([]); setSearched(false); setError('') }
  return <dialog ref={dialog} className="rating-dialog metadata-dialog" aria-labelledby="metadata-title" onCancel={event => { event.preventDefault(); if (!saving) onClose() }}>
    <div className="dialog-content"><button className="icon-button dialog-close" onClick={onClose} disabled={saving} aria-label="Fechar informações"><Icon name="close" /></button>
      <p className="dialog-caption">Para lembrar dessa história</p><h2 id="metadata-title">Informações da obra</h2><p className="metadata-current">{item.title}</p>
      {item.overview && <p className="dialog-overview">{item.overview}</p>}
      <p className="metadata-help">Escolha a versão correta. As informações serão atualizadas para nós dois; nossas notas continuam iguais.</p>
      <form onSubmit={search} className="metadata-search"><label className="field">Título da obra<input ref={input} value={query} onChange={event => { edited(); setQuery(event.target.value) }} disabled={saving} maxLength={200} required /></label><div className="metadata-search-controls"><label className="field">Tipo<select value={kind} onChange={event => { edited(); setKind(event.target.value) }} disabled={saving || item.media_type === 'tv'}><option value="">Filmes e séries</option><option value="movie">Filme</option><option value="tv">Série</option></select></label><button className="button primary" disabled={saving || searching} type="submit">{searching ? 'Buscando…' : 'Buscar no TMDB'}</button></div></form>
      <div aria-live="polite">{searching && <p className="muted">Buscando versões e sinopses…</p>}{searched && <p className="metadata-result-count">{items.length ? `${items.length} resultado${items.length === 1 ? '' : 's'}. Confira título, ano e sinopse.` : 'Nenhuma versão encontrada. Tente outro nome ou o título original.'}</p>}</div>
      {error && <p className="notice error" role="alert">{error}</p>}
      <ul className="metadata-results">{items.map(candidate => <li key={`${candidate.media_type}:${candidate.id}`}><Poster title={candidate.title} path={candidate.poster_path} /><div><span className="media-label">{candidate.media_type === 'tv' ? 'SÉRIE' : 'FILME'}{candidate.year ? ` · ${candidate.year}` : ''}</span><h3>{candidate.title}</h3><p>{candidate.overview || 'Sinopse não disponível nesta busca.'}</p><button className="button subtle" onClick={() => confirm(candidate)} disabled={saving} aria-label={`Usar ${candidate.title}${candidate.year ? `, ${candidate.year}` : ''}`}>Usar esta versão<Icon name="check" size={14} /></button></div></li>)}</ul>
      {item.service === 'netflix' && item.rating_key && <aside className="metadata-netflix"><strong>Faltou informação no TMDB?</strong><p>Use a imagem e a sinopse da página oficial desta obra na Netflix.</p><button className="button subtle" onClick={() => confirm()} disabled={saving}>Usar informações oficiais da Netflix</button></aside>}
      <div className="dialog-actions"><span role="status">{saving ? 'Salvando informações…' : ''}</span><button className="button subtle" onClick={onClose} disabled={saving}>Fechar</button></div>
    </div>
  </dialog>
}
