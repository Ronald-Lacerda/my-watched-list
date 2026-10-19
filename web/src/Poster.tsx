import { useState } from 'react'
import { Icon } from './icons'
export function Poster({ title, path }: { title: string; path: string | null }) {
  const [broken, setBroken] = useState(false)
  const tone = Array.from(title).reduce((value, char) => value + (char.codePointAt(0) ?? 0), 0) % 4
  return <div className={`poster poster-${tone}`} aria-hidden="true">{path?.startsWith('/') && !broken ? <img src={`https://image.tmdb.org/t/p/w342${path}`} alt="" loading="lazy" decoding="async" onError={() => setBroken(true)} /> : <><span className="poster-orbit" /><span className="poster-letter">{title.slice(0, 1)}</span><span className="poster-caption"><Icon name="play" size={9} /> MWL</span></>}</div>
}
