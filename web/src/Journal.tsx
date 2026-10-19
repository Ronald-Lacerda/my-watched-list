import { useEffect, useRef, useState } from 'react'
import { api, message, scoreLabel, serviceName, type FilterOptions, type HistoryResult, type RatingSummary, type User, type WatchItem } from './api'
import { Icon } from './icons'
import { Poster } from './Poster'
import { RatingDialog } from './RatingDialog'

interface Filters { person: string; service: string; from: string; to: string }
const emptyFilters: Filters = { person: '', service: '', from: '', to: '' }
function readFilters(): Filters { const params = new URLSearchParams(location.search); return { person: params.get('person') ?? '', service: params.get('service') ?? '', from: params.get('from') ?? '', to: params.get('to') ?? '' } }
function dayLabel(day: string): string { return new Intl.DateTimeFormat('pt-BR', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${day}T12:00:00Z`)) }

export function Journal({ user, pending }: { user: User; pending: boolean }) {
  const [filters, setFilters] = useState<Filters>(readFilters)
  const [options, setOptions] = useState<FilterOptions>({ people: [], services: [] })
  const [items, setItems] = useState<WatchItem[]>([]), [total, setTotal] = useState(0), [cursor, setCursor] = useState<string | null>(null)
  const [loading, setLoading] = useState(true), [loadingMore, setLoadingMore] = useState(false), [error, setError] = useState(''), [optionsError, setOptionsError] = useState('')
  const [retry, setRetry] = useState(0), [selected, setSelected] = useState<WatchItem | null>(null), [notice, setNotice] = useState('')
  const request = useRef<AbortController | null>(null)
  useEffect(() => { const restore = () => { request.current?.abort(); setFilters(readFilters()) }; window.addEventListener('popstate', restore); return () => window.removeEventListener('popstate', restore) }, [])
  const query = new URLSearchParams(Object.entries(filters).filter(([, value]) => value !== '')); if (pending) query.set('pending', 'true')
  const queryString = query.toString()
  useEffect(() => { const controller = new AbortController(); setOptionsError(''); api<FilterOptions>('/filters', { signal: controller.signal }).then(value => { if (!controller.signal.aborted) setOptions(value) }).catch(err => { if (!controller.signal.aborted) setOptionsError(message(err)) }); return () => controller.abort() }, [retry])
  useEffect(() => {
    const controller = new AbortController(); request.current?.abort(); request.current = controller; setLoading(true); setLoadingMore(false); setItems([]); setTotal(0); setCursor(null); setError('')
    if (filters.from && filters.to && filters.from > filters.to) { setError('A data inicial precisa ser anterior ou igual à data final.'); setLoading(false); return () => controller.abort() }
    api<HistoryResult>(`/history?${queryString}`, { signal: controller.signal }).then(value => { if (!controller.signal.aborted) { setItems(value.items); setTotal(value.total); setCursor(value.next_cursor) } }).catch(err => { if (!controller.signal.aborted) setError(message(err)) }).finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [queryString, retry])
  useEffect(() => { if (!notice) return; const timer = setTimeout(() => setNotice(''), 4000); return () => clearTimeout(timer) }, [notice])
  function change(next: Filters) { request.current?.abort(); setFilters(next); const params = new URLSearchParams(location.search); for (const key of Object.keys(emptyFilters) as (keyof Filters)[]) { if (next[key]) params.set(key, next[key]); else params.delete(key) } history.replaceState(null, '', location.pathname + (params.size ? `?${params}` : '')) }
  async function more() {
    if (!cursor || loadingMore) return
    const controller = new AbortController(); request.current?.abort(); request.current = controller; setLoadingMore(true); setError('')
    try { const value = await api<HistoryResult>(`/history?${queryString}&cursor=${encodeURIComponent(cursor)}`, { signal: controller.signal }); if (!controller.signal.aborted) { setItems(old => { const seen = new Set(old.map(item => item.id)); return [...old, ...value.items.filter(item => !seen.has(item.id))] }); setTotal(value.total); setCursor(value.next_cursor) } } catch (err) { if (!controller.signal.aborted) setError(message(err)) } finally { if (!controller.signal.aborted) setLoadingMore(false) }
  }
  function saved(summary: RatingSummary) {
    const key = selected?.rating_key
    const hasOwn = summary.ratings.some(rating => rating.user_id === user.id)
    if (pending && hasOwn) { setItems(old => old.filter(item => item.rating_key !== key)); setTotal(value => Math.max(0, value - 1)) }
    else setItems(old => old.map(item => item.rating_key === key ? { ...item, ...summary } : item))
    setSelected(null); setNotice(hasOwn ? 'Sua nota foi salva.' : 'Sua nota foi removida.')
  }
  const groups = new Map<string, WatchItem[]>(); for (const item of items) { const group = groups.get(item.watched_on) ?? []; group.push(item); groups.set(item.watched_on, group) }
  const hasFilters = Object.values(filters).some(Boolean)
  return <>
    <header className="page-heading journal-heading"><div><p className="eyebrow">{pending ? 'CADA OPINIÃO CONTA' : 'UM REGISTRO A DOIS'}</p><h1 tabIndex={-1}>{pending ? 'Para avaliar' : 'Nosso histórico'}</h1><p className="muted">{pending ? 'Histórias que ainda esperam a sua opinião.' : 'O que assistimos, o que sentimos, o que vale lembrar.'}</p></div><span className="journal-heading-icon" aria-hidden="true"><Icon name={pending ? 'star' : 'history'} size={25} /></span></header>
    <section className="filter-panel" aria-label="Filtros do histórico"><div className="filter-heading"><Icon name="filter" size={16} /><span>Encontre uma lembrança</span><button className="clear-filters" onClick={() => change(emptyFilters)} disabled={!hasFilters}>Limpar filtros</button></div><div className="filters"><label className="field">Pessoa<select aria-label="Pessoa" value={filters.person} onChange={event => change({ ...filters, person: event.target.value })}><option value="">Nós dois</option>{options.people.map(person => <option key={person.id} value={person.id}>{person.name}</option>)}</select></label><label className="field">Streaming<select aria-label="Streaming" value={filters.service} onChange={event => change({ ...filters, service: event.target.value })}><option value="">Todos os streamings</option>{options.services.map(service => <option key={service} value={service}>{serviceName(service)}</option>)}</select></label><label className="field">De<input type="date" aria-label="Data inicial" value={filters.from} onChange={event => change({ ...filters, from: event.target.value })} /></label><label className="field">Até<input type="date" aria-label="Data final" value={filters.to} onChange={event => change({ ...filters, to: event.target.value })} /></label></div></section>
    {optionsError && <div className="notice error" role="alert">{optionsError}<button className="text-button" onClick={() => setRetry(value => value + 1)}>Tentar novamente</button></div>}
    <div className="feed-heading"><h2>{pending ? 'Um convite para dar sua nota' : 'Histórias que passaram por aqui'}</h2><span role="status">{loading ? 'Buscando…' : `${total.toLocaleString('pt-BR')} ${pending ? total === 1 ? 'obra' : 'obras' : total === 1 ? 'registro' : 'registros'}`}</span></div>
    {notice && <p className="notice success" role="status">{notice}</p>}
    {error && <div className="notice error" role="alert">{error}<button className="text-button" onClick={() => items.length && cursor ? more() : setRetry(value => value + 1)}>Tentar novamente</button></div>}
    {loading && <div className="feed-loading" role="status"><span className="loading-dot" /> Buscando nossas histórias…</div>}
    {!loading && !error && items.length === 0 && <section className="empty-state"><span className="empty-icon"><Icon name={pending ? 'check' : 'history'} size={30} /></span><h3>{pending ? 'Nenhuma obra para avaliar por aqui.' : 'Nenhuma história neste recorte.'}</h3><p>{pending ? 'Quando uma nova história chegar, sua próxima opinião ganha espaço aqui.' : 'Experimente outra pessoa, outro streaming ou amplie o período.'}</p></section>}
    <div className="journal-feed">{Array.from(groups.entries()).map(([day, records]) => <section className="day-group" key={day} aria-label={dayLabel(day)}><div className="day-heading"><span className="day-dot" /><h2>{dayLabel(day)}</h2><span className="day-line" /><small>{records.length}</small></div>{records.map(item => {
      const own = item.ratings.find(rating => rating.user_id === user.id)
      return <article className="watch-card" key={item.id} data-event-id={item.id} aria-label={`${item.title}, assistido por ${item.person}`}><Poster title={item.title} path={item.poster_path} /><div className="watch-info"><div className="watch-type"><span>{item.media_type === 'tv' ? 'SÉRIE' : item.media_type === 'movie' ? 'FILME' : 'VÍDEO'}</span>{item.year && <span>· {item.year}</span>}</div><h3>{item.title}</h3>{item.media_type === 'tv' && item.source_title !== item.title && <p className="episode-title">{item.source_title}</p>}<div className="watch-meta"><span className="watch-person"><span className="tiny-avatar">{item.person.slice(0, 1)}</span>{item.person}</span><span className={`service-pill ${item.service === 'netflix' ? 'netflix' : ''}`}>{item.service === 'netflix' && <b>N</b>}{serviceName(item.service)}</span></div><div className="individual-scores">{options.people.map(person => { const rating = item.ratings.find(value => value.user_id === person.id); return <span key={person.id}>{person.name}<b>{rating ? scoreLabel(rating.score) : '—'}</b></span> })}</div></div><div className="watch-rating"><div className="couple-score"><Icon name="star" size={15} /><strong>{item.average !== null ? scoreLabel(item.average) : '—'}</strong><span>{item.ratings_count ? `Média · ${item.ratings_count}/${options.people.length || 2}` : 'Sem notas ainda'}</span></div><button className={`rating-button ${own ? 'is-rated' : ''}`} disabled={!item.rating_key} onClick={() => setSelected(item)} aria-label={`${own ? 'Editar nota de' : 'Avaliar'} ${item.title}`} title={item.rating_key ? undefined : 'Esta série precisa de uma nova sincronização antes de receber nota.'}><Icon name={own ? 'pencil' : 'star'} size={14} />{item.rating_key ? own ? `Sua nota: ${scoreLabel(own.score)}` : 'Dar minha nota' : 'Aguardando atualização'}</button></div></article>
    })}</section>)}</div>
    {!loading && items.length > 0 && <footer className="feed-footer"><span>{pending ? 'Uma opinião por obra, no seu tempo.' : `Exibindo ${items.length.toLocaleString('pt-BR')} de ${total.toLocaleString('pt-BR')} registros`}</span>{cursor && <button className="button" onClick={more} disabled={loadingMore}>{loadingMore ? 'Carregando…' : 'Carregar mais'}{!loadingMore && <Icon name="arrow" size={16} />}</button>}</footer>}
    {selected && <RatingDialog item={selected} user={user} people={options.people} onClose={() => setSelected(null)} onSaved={saved} />}
  </>
}
