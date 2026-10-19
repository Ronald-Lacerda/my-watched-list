import { useState } from 'react'
import { Icon } from './icons'
export function Poster({ title, path, featured = false }: { title: string; path: string | null; featured?: boolean }) {
  const [brokenPath, setBrokenPath] = useState<string | null>(null)
  const tone = Array.from(title).reduce((value, char) => value + (char.codePointAt(0) ?? 0), 0) % 4
  return <div className={`poster poster-${tone}${featured ? ' poster-featured' : ''}`} aria-hidden="true">{path?.startsWith('/') && brokenPath !== path ? <img src={`https://image.tmdb.org/t/p/${featured ? 'w780' : 'w342'}${path}`} alt="" width={featured ? 780 : 342} height={featured ? 1170 : 513} loading={featured ? 'eager' : 'lazy'} fetchPriority={featured ? 'high' : 'auto'} decoding="async" onError={() => setBrokenPath(path)} /> : <><span className="poster-orbit" /><span className="poster-letter">{Array.from(title)[0]}</span><span className="poster-caption"><Icon name="play" size={9} /> My Watched List</span></>}</div>
}
