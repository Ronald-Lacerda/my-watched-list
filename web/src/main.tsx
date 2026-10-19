import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import './style.css'

function App() {
  const [status, setStatus] = useState('Verificando API…')

  useEffect(() => {
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), 5000)
    fetch('/api/readyz', { signal: controller.signal, cache: 'no-store' })
      .then(async response => {
        if (!response.ok || (await response.json()).status !== 'ready') {
          throw new Error('API indisponível')
        }
        setStatus('API e banco disponíveis')
      })
      .catch(() => setStatus('API indisponível'))
      .finally(() => clearTimeout(timeout))
    return () => {
      clearTimeout(timeout)
      controller.abort()
    }
  }, [])

  return (
    <main>
      <header><span className="mark" aria-hidden="true">▶</span> MY WATCHED LIST</header>
      <section className="intro" aria-labelledby="title">
        <p className="eyebrow">UM REGISTRO A DOIS</p>
        <h1 id="title">O que vimos.<br />O que ficou.</h1>
        <p className="description">Filmes, séries e nossas opiniões. Um lugar para guardar o que assistimos e redescobrir os favoritos de cada ano.</p>
      </section>
      <section className="progress" aria-labelledby="progress-title">
        <div><span className="tag">FASE 0</span><h2 id="progress-title">A base está pronta.</h2></div>
        <p>Este é o ponto de partida. A coleta da Netflix será a próxima etapa; a linha do tempo e as notas chegam depois.</p>
        <p className="status" role="status">{status}</p>
      </section>
      <footer>Nosso histórico começa em breve.</footer>
    </main>
  )
}

createRoot(document.getElementById('root')!).render(<StrictMode><App /></StrictMode>)
