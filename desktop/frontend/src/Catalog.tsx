import { useState, useEffect, useCallback, useRef } from 'react'
import type { CatalogItem, CatalogSection } from './lib/backend'

const IconFilm = () => (
  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" style={{ opacity: 0.28 }}>
    <rect x="2" y="2" width="20" height="20" rx="2" />
    <path d="M7 2v20M17 2v20M2 12h20M2 7h5M17 7h5M2 17h5M17 17h5" />
  </svg>
)

const IconPlay = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
    <polygon points="5 3 19 12 5 21 5 3" />
  </svg>
)

const IconInfo = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" />
  </svg>
)

const statusLabels: Record<string, string> = {
  FINISHED: 'Finalizado',
  RELEASING: 'Em exibição',
  NOT_YET_RELEASED: 'Em breve',
  CANCELLED: 'Cancelado',
}

interface Props {
  sections: CatalogSection[]
  loading: boolean
  onPlay: (title: string) => void
}

function HeroSkeleton() {
  return <div className="catalog-hero-skeleton" />
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

function Hero({ items, onPlay }: { items: CatalogItem[]; onPlay: (t: string) => void }) {
  const [activeIdx, setActiveIdx] = useState(0)
  const [prevIdx, setPrevIdx] = useState<number | null>(null)
  const [fading, setFading] = useState(false)

  const goTo = useCallback((idx: number) => {
    if (idx === activeIdx) return
    setPrevIdx(activeIdx)
    setFading(true)
    setTimeout(() => {
      setActiveIdx(idx)
      setFading(false)
      setPrevIdx(null)
    }, 500)
  }, [activeIdx])

  useEffect(() => {
    if (items.length <= 1) return
    const id = setInterval(() => {
      setActiveIdx(cur => {
        const next = (cur + 1) % items.length
        setPrevIdx(cur)
        setFading(true)
        setTimeout(() => {
          setFading(false)
          setPrevIdx(null)
        }, 500)
        return next
      })
    }, 8000)
    return () => clearInterval(id)
  }, [items.length])

  const item = items[activeIdx]
  const prev = prevIdx !== null ? items[prevIdx] : null
  const bg = item.bannerImage || item.coverImage
  const prevBg = prev ? (prev.bannerImage || prev.coverImage) : null

  return (
    <div className="catalog-hero">
      {/* Previous bg (fades out) */}
      {prevBg && (
        <div
          className="catalog-hero-bg"
          style={{ backgroundImage: `url(${prevBg})`, opacity: fading ? 0 : 1, transition: 'opacity 0.5s ease' }}
        />
      )}
      {/* Current bg */}
      <div
        className="catalog-hero-bg"
        style={{ backgroundImage: `url(${bg})`, opacity: fading ? 0 : 1, transition: 'opacity 0.5s ease' }}
      />
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
          {item.score > 0 && <span className="catalog-score">★ {item.score.toFixed(1)}</span>}
          {item.episodes > 0 && <span className="catalog-eps">{item.episodes} episódios</span>}
        </div>
        {item.description && (
          <p className="catalog-hero-desc">
            {item.description.length > 180 ? item.description.slice(0, 180) + '…' : item.description}
          </p>
        )}
        <div className="catalog-hero-actions">
          <button
            className="btn btn-accent catalog-hero-btn"
            onClick={e => { e.stopPropagation(); onPlay(item.title) }}
          >
            <IconPlay /> Assistir agora
          </button>
          <button
            className="btn catalog-hero-btn-info"
            onClick={e => { e.stopPropagation(); onPlay(item.title) }}
          >
            <IconInfo /> Mais info
          </button>
        </div>
      </div>

      {/* Rotation dots */}
      {items.length > 1 && (
        <div className="catalog-hero-dots">
          {items.map((_, i) => (
            <button
              key={i}
              className={`catalog-hero-dot${i === activeIdx ? ' active' : ''}`}
              onClick={e => { e.stopPropagation(); goTo(i) }}
            />
          ))}
        </div>
      )}
    </div>
  )
}

const IconChevronLeft = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="15 18 9 12 15 6" />
  </svg>
)

const IconChevronRight = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="9 18 15 12 9 6" />
  </svg>
)

function CatalogRow({ section, onPlay }: { section: CatalogSection; onPlay: (t: string) => void }) {
  const rowRef = useRef<HTMLDivElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)

  const updateScrollState = useCallback(() => {
    const el = rowRef.current
    if (!el) return
    setCanScrollLeft(el.scrollLeft > 0)
    setCanScrollRight(el.scrollLeft < el.scrollWidth - el.clientWidth - 1)
  }, [])

  useEffect(() => {
    const el = rowRef.current
    if (!el) return
    updateScrollState()
    el.addEventListener('scroll', updateScrollState, { passive: true })
    const ro = new ResizeObserver(updateScrollState)
    ro.observe(el)
    return () => { el.removeEventListener('scroll', updateScrollState); ro.disconnect() }
  }, [updateScrollState])

  const scroll = useCallback((direction: 'left' | 'right') => {
    rowRef.current?.scrollBy({ left: direction === 'left' ? -640 : 640, behavior: 'smooth' })
  }, [])

  return (
    <div className="catalog-section">
      <div className="catalog-section-header">
        <p className="catalog-section-label">{section.label}</p>
      </div>
      <div className="catalog-row-wrap">
        {canScrollLeft && (
          <button className="catalog-row-arrow catalog-row-arrow--prev" onClick={() => scroll('left')} aria-label="Rolar para a esquerda">
            <IconChevronLeft />
          </button>
        )}
        <div className="catalog-row" ref={rowRef}>
          {section.items.map(item => (
            <div key={item.id} className="catalog-card" onClick={() => onPlay(item.title)} title={item.title}>
              <div className="catalog-card-cover">
                {item.coverImage
                  ? <img src={item.coverImage} alt={item.title} loading="lazy" />
                  : <div className="catalog-card-fallback"><IconFilm /></div>
                }
                {item.score > 0 && (
                  <span className="catalog-score-badge">★ {item.score.toFixed(1)}</span>
                )}
              </div>
              <p className="catalog-card-title">{item.title}</p>
              {(item.description || item.genres.length > 0) && (
                <div className="catalog-tooltip">
                  {item.score > 0 && <p className="catalog-tooltip-score">★ {item.score.toFixed(1)}</p>}
                  {item.genres.length > 0 && (
                    <div className="catalog-tooltip-genres">
                      {item.genres.slice(0, 3).map(g => (
                        <span key={g} className="catalog-genre-pill">{g}</span>
                      ))}
                    </div>
                  )}
                  {item.episodes > 0 && <p className="catalog-tooltip-eps">{item.episodes} episódios</p>}
                  {item.description && (
                    <p className="catalog-tooltip-desc">
                      {item.description.length > 140 ? item.description.slice(0, 140) + '…' : item.description}
                    </p>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
        {canScrollRight && (
          <button className="catalog-row-arrow catalog-row-arrow--next" onClick={() => scroll('right')} aria-label="Rolar para a direita">
            <IconChevronRight />
          </button>
        )}
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

  // Collect up to 5 items with banner images across all sections for the rotating hero
  const heroItems: CatalogItem[] = []
  for (const section of sections) {
    for (const item of section.items) {
      if (item.bannerImage && heroItems.length < 5) {
        heroItems.push(item)
      }
    }
  }
  if (heroItems.length === 0 && sections[0]?.items[0]) {
    heroItems.push(sections[0].items[0])
  }

  return (
    <div className="catalog">
      {heroItems.length > 0 && <Hero items={heroItems} onPlay={onPlay} />}
      {sections.map(section => (
        <CatalogRow key={section.label} section={section} onPlay={onPlay} />
      ))}
    </div>
  )
}
