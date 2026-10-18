export interface Person { id: string; name: string }
export interface User extends Person { email: string }
export interface Rating { user_id: string; name: string; score: number; comment: string; updated_at: string }
export interface RatingSummary { ratings: Rating[]; average: number | null; ratings_count: number }
export interface WatchItem extends RatingSummary {
  id: string; person_id: string; person: string; service: string; title: string; source_title: string
  media_type: string; watched_on: string; watched_at: string | null; poster_path: string | null
  year: number | null; overview: string; genres: string[]; rating_key: string | null
}
export interface HistoryResult { items: WatchItem[]; total: number; next_cursor: string | null }
export interface FilterOptions { people: Person[]; services: string[] }
export interface Connection { id: string; person_id: string; person: string; service: string; profile: string; status: string; last_sync: string | null }
export class ApiError extends Error {
  constructor(public status: number, public code: string, message: string) { super(message) }
}
export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  headers.set('Accept', 'application/json')
  if (options.body !== undefined) headers.set('Content-Type', 'application/json')
  let response: Response
  try { response = await fetch(`/api${path}`, { ...options, headers, credentials: 'same-origin', cache: 'no-store' }) }
  catch (error) {
    if (options.signal?.aborted) throw error
    throw new ApiError(0, 'network', 'Não foi possível conectar. Confira sua conexão e tente novamente.')
  }
  if (!response.ok) {
    const data = await response.json().catch(() => null) as { error?: { code?: string; message?: string } } | null
    if (response.status === 401 && !path.startsWith('/auth/')) window.dispatchEvent(new Event('session-expired'))
    throw new ApiError(response.status, data?.error?.code ?? 'request_failed', data?.error?.message ?? 'Não foi possível concluir. Tente novamente.')
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}
export function message(error: unknown): string { return error instanceof Error ? error.message : 'Não foi possível concluir. Tente novamente.' }
export function serviceName(service: string): string { return ({ netflix: 'Netflix', prime_video: 'Prime Video', disney_plus: 'Disney+', max: 'Max', apple_tv: 'Apple TV+' } as Record<string, string>)[service] ?? service }
export function scoreLabel(score: number): string { return score.toLocaleString('pt-BR', { maximumFractionDigits: 2 }) }
