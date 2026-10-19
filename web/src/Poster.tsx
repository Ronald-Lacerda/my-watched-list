import { useState } from 'react'
import { Icon } from './icons'
export function Poster({ title, path, url, featured = false }: { title: string; path: string | null; url?: string; featured?: boolean }) {
  const [brokenPath, setBrokenPath] = useState<string | null>(null)
  let official = ''
  if (url) { try { const image = new URL(url); if (image.protocol === 'https:' && !image.username && !image.password && !image.port && ['nflxso.net', 'nflxext.com', 'nflximg.net'].some(host => image.hostname === host || image.hostname.endsWith(`.${host}`))) official = url } catch { /* Usa o pôster TMDB ou a capa provisória. */ } }
  const src = official || (path?.startsWith('/') ? `https://image.tmdb.org/t/p/${featured ? 'w780' : 'w342'}${path}` : '')
  const tone = Array.from(title).reduce((value, char) => value + (char.codePointAt(0) ?? 0), 0) % 4
  return <div className={`poster poster-${tone}${featured ? ' poster-featured' : ''}`} aria-hidden="true">{src && brokenPath !== src ? <img src={src} alt="" width={featured ? 780 : 342} height={featured ? 1170 : 513} loading={featured ? 'eager' : 'lazy'} fetchPriority={featured ? 'high' : 'auto'} decoding="async" referrerPolicy="no-referrer" onError={() => setBrokenPath(src)} /> : <><span className="poster-orbit" /><span className="poster-letter">{Array.from(title)[0]}</span><span className="poster-caption"><Icon name="play" size={9} /> My Watched List</span></>}</div>
}
