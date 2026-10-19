type IconName = 'play' | 'clapperboard' | 'history' | 'star' | 'link' | 'logout' | 'close' | 'arrow' | 'check' | 'filter' | 'calendar' | 'alert' | 'pencil' | 'trash'
const paths: Record<IconName, string> = {
  play: 'm9 5 11 7-11 7V5Z', history: 'M3 11a9 9 0 1 1 2.5 7M3 4v7h7m2-4v5l3 2',
  clapperboard: 'm3 8 17-5 1 5-17 5-1-5Zm1 5v8h17V8M8 7l3 4m3-6 3 4M4 15h17',
  star: 'm12 3 2.8 5.7 6.3.9-4.5 4.4 1.1 6.2-5.7-3-5.7 3 1.1-6.2L3.2 9.6l6.3-.9L12 3Z',
  link: 'm10 13 4-4m-6.5 7.5-1 1a4.3 4.3 0 0 1-6-6l4-4a4.3 4.3 0 0 1 6 0m2-1 1-1a4.3 4.3 0 0 1 6 6l-4 4a4.3 4.3 0 0 1-6 0',
  logout: 'M9 4H4v16h5m5-12 4 4-4 4m-6-4h10', close: 'm6 6 12 12M18 6 6 18', arrow: 'M5 12h14m-5-5 5 5-5 5',
  check: 'm5 12 4 4L19 6', filter: 'M4 6h16M7 12h10m-7 6h4', calendar: 'M7 3v4m10-4v4M4 10h16M4 5h16v16H4V5Z',
  alert: 'm12 3 10 18H2L12 3Zm0 6v5m0 3v.1', pencil: 'm15 4 5 5M4 20l5-1L20 8a3.5 3.5 0 0 0-5-5L4 15v5Z', trash: 'M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14M10 11v6m4-6v6',
}
export function Icon({ name, size = 20 }: { name: IconName; size?: number }) { return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d={paths[name]} /></svg> }
export function Brand() { return <div className="brand"><span className="brand-mark"><Icon name="clapperboard" size={25} /></span><span>My Watched List</span></div> }
