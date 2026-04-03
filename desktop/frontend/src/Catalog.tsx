import type { CatalogItem, CatalogSection } from './lib/backend'

const IconFilm = () => (
  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" style={{ opacity: 0.28 }}>
    <rect x="2" y="2" width="20" height="20" rx="2" />
    <path d="M7 2v20M17 2v20M2 12h20M2 7h5M17 7h5M2 17h5M17 17h5" />
  </svg>
)

const statusLabels: Record<string, string> = {
  FINISHED: 'Finalizado',
  RELEASING: 'Em exibi??o',
  NOT_YET_RELEASED: 'Em breve',
  CANCELLED: 'Cancelado',
}

const IconPlay = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
    <polygon points="5 3 19 12 5 21 5 3" />
  </svg>
)

interface Props {
  sections: CatalogSection[]
  loading: boolean
  onPlay: (title: string) => void
}

function HeroSkeleton() {
  return (
    <div className="catalog-hero-skeleton" />
  )
}

function RowSkeleton() {
  return (
    <div className="catalog-section">
      <div className="skeleton-line" style={{ width: 140, marginLeft: 0, marginBottom: 12 }} />
      <div className="catalog-row">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="catalog-card-skeleton" />
        ))}
      </div>
    </div>
  )
}

function Hero({ item, onPlay }: { item: CatalogItem; onPlay: (t: string) => void }) {
  const bg = item.bannerImage || item.coverImage

  return (
    <div className="catalog-hero" onClick={() => onPlay(item.title)} title={item.title}>
      <div className="catalog-hero-bg" style={{ backgroundImage: `url(${bg})` }} />
      <div className="catalog-hero-overlay" />
      <div className="catalog-hero-content">
        {item.genres.length > 0 && (
          <div className="catalog-hero-genres">
            {item.genres.slice(0, 3).map(g => (
              <span key={g} className="catalog-genre-pill">{g}</span>
            ))}
            {item.status && (
              <span className="catalog-genre-pill catalog-status-pill">
                {statusLabels[item.status] ?? item.status}
              </span>
            )}
          </div>
        )}
        <h2 className="catalog-hero-title">{item.title}</h2>
        <div className="catalog-hero-meta">
          {item.score > 0 && <span className="catalog-score">* {item.score.toFixed(1)}</span>}
          {item.episodes > 0 && <span className="catalog-eps">{item.episodes} eps</span>}
        </div>
        {item.description && (
          <p className="catalog-hero-desc">{item.description}</p>
        )}
        <button
          className="btn btn-accent catalog-hero-btn"
          onClick={e => { e.stopPropagation(); onPlay(item.title) }}
        >
          <IconPlay /> Buscar e assistir
        </button>
      </div>
    </div>
  )
}

function CatalogRow({ section, onPlay }: { section: CatalogSection; onPlay: (t: string) => void }) {
  return (
    <div className="catalog-section">
      <p className="catalog-section-label">{section.label}</p>
      <div className="catalog-row">
        {section.items.map(item => (
          <div key={item.id} className="catalog-card" onClick={() => onPlay(item.title)} title={item.title}>
            <div className="catalog-card-cover">
              {item.coverImage
                ? <img src={item.coverImage} alt={item.title} loading="lazy" />
                : <div className="catalog-card-fallback"><IconFilm /></div>
              }
              {item.score > 0 && (
                <span className="catalog-score-badge">{'\u2605'} {item.score.toFixed(1)}</span>
              )}
            </div>
            <p className="catalog-card-title">{item.title}</p>
            {(item.description || item.genres.length > 0) && (
              <div className="catalog-tooltip">
                {item.score > 0 && <p className="catalog-tooltip-score">{'\u2605'} {item.score.toFixed(1)}</p>}
                {item.genres.length > 0 && (
                  <div className="catalog-tooltip-genres">
                    {item.genres.slice(0, 3).map(g => (
                      <span key={g} className="catalog-genre-pill">{g}</span>
                    ))}
                  </div>
                )}
                {item.episodes > 0 && <p className="catalog-tooltip-eps">{item.episodes} {'epis\u00f3dios'}</p>}
                {item.description && (
                  <p className="catalog-tooltip-desc">
                    {item.description.length > 140 ? item.description.slice(0, 140) + '\u2026' : item.description}
                  </p>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

export function Catalog({ sections, loading, onPlay }: Props) {
  if (loading) {
    return (
      <div className="catalog">
        <HeroSkeleton />
        <RowSkeleton />
        <RowSkeleton />
      </div>
    )
  }

  if (sections.length === 0) return null

  const hero = sections[0]?.items.find(i => i.bannerImage) ?? sections[0]?.items[0]

  return (
    <div className="catalog">
      {hero && <Hero item={hero} onPlay={onPlay} />}
      {sections.map(section => (
        <CatalogRow key={section.label} section={section} onPlay={onPlay} />
      ))}
    </div>
  )
}
