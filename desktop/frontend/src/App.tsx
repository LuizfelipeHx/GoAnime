import { useEffect, useMemo, useRef, useState } from 'react'
import Hls from 'hls.js'
import { Catalog } from './Catalog'
import {
  EventsOn,
  Quit,
  WindowFullscreen,
  WindowIsFullscreen,
  WindowIsMaximised,
  WindowMinimise,
  WindowToggleMaximise,
  WindowUnfullscreen,
} from '../wailsjs/runtime/runtime'
import {
  addFavorite,
  downloadEpisode,
  getCatalog,
  getMovieCatalog,
  getEpisodes,
  getFavorites,
  getProxyBaseURL,
  getRelatedAnime,
  getSearchHistory,
  getStream,
  getWatchProgress,
  removeFavorite,
  searchMedia,
  updateWatchProgress,
  type CatalogSection,
  type EpisodeResult,
  type FavoriteEntry,
  type HistoryEntry,
  type MediaRequest,
  type MediaResult,
  type RelatedAnime,
  type SearchCoversEvent,
  type WatchProgressEntry,
} from './lib/backend'

type SourceFilter = 'all' | 'allanime' | 'animefire' | 'flixhq' | 'animesonlinecc' | 'anroll' | 'bakashi'
type TypeFilter = 'all' | 'anime' | 'movie' | 'tv'
type LangFilter = 'all' | 'pt' | 'en'
type LoadState = 'idle' | 'loading' | 'ready' | 'error'
type ViewMode = 'catalog' | 'movies' | 'favorites' | 'watching'
type MovieSourceMode = 'ptbr' | 'extra'
type ToastType = 'success' | 'error' | 'info'
type Toast = { id: number; message: string; type: ToastType }

const sources: { label: string; value: SourceFilter }[] = [
  { label: 'Todas as fontes', value: 'all' },
  { label: 'Bakashi', value: 'bakashi' },
  { label: 'AnimeFire', value: 'animefire' },
  { label: 'AnimesOnlineCC', value: 'animesonlinecc' },
  { label: 'Anroll', value: 'anroll' },
  { label: 'AllAnime', value: 'allanime' },
  { label: 'FlixHQ', value: 'flixhq' },
]

const movieSourcesPrimary: { label: string; value: SourceFilter }[] = [
  { label: 'Fontes principais', value: 'all' },
  { label: 'Bakashi', value: 'bakashi' },
  { label: 'AnimeFire', value: 'animefire' },
  { label: 'AnimesOnlineCC', value: 'animesonlinecc' },
]
const movieSourcesExtra: { label: string; value: SourceFilter }[] = [
  { label: 'Todas as fontes de filme', value: 'all' },
  { label: 'FlixHQ (extra / inglês)', value: 'flixhq' },
  { label: 'AnimeFire', value: 'animefire' },
  { label: 'AnimesOnlineCC', value: 'animesonlinecc' },
]

const typeLabels: Record<TypeFilter, string> = {
  all: 'Tudo',
  anime: 'Anime',
  movie: 'Filme',
  tv: 'Serie',
}

const qualityOptions = ['best', '1080p', '720p', '480p', 'worst']
const browseTypeOptions: TypeFilter[] = ['all', 'anime', 'tv']
const progressMetaKey = 'progress-meta'

function cleanTitle(value: string) {
  return value.replace(/\[(English|Portuguese|Portugu\u00EAs|Dublado|Legendado|Dub|Sub)\]/gi, '').trim()
}

type LangTag = { label: string; variant: 'pt' | 'en' | 'neutral' }

function extractLangTag(name: string): LangTag | null {
  const match = name.match(/\[(English|Portuguese|Portugu[êeÊE]s|Dublado|Legendado|Dub|Sub)\]/i)
  if (!match) return null
  const raw = match[1].toLowerCase()
  if (raw.includes('portugu') || raw === 'dublado' || raw === 'dub') {
    return { label: 'PT-BR', variant: 'pt' }
  }
  if (raw === 'english' || raw === 'sub' || raw === 'legendado') {
    return { label: raw === 'english' ? 'EN' : 'LEG', variant: 'en' }
  }
  return { label: match[1], variant: 'neutral' }
}

function hasPortugueseSignal(item: Pick<MediaResult, 'name' | 'hasPortuguese' | 'hasDub'>) {
  if (item.hasPortuguese || item.hasDub) return true
  return extractLangTag(item.name)?.variant === 'pt'
}

function hasSubtitleSignal(item: Pick<MediaResult, 'name' | 'hasEnglish' | 'hasSub'>) {
  if (item.hasEnglish || item.hasSub) return true
  const tag = extractLangTag(item.name)
  return tag?.variant === 'en' || tag === null
}

function watchHasPortugueseSignal(item: Pick<MediaResult, 'name' | 'watchHasPortuguese' | 'watchHasDub'>) {
  if (item.watchHasPortuguese || item.watchHasDub) return true
  return extractLangTag(item.name)?.variant === 'pt'
}

function watchHasSubtitleSignal(item: Pick<MediaResult, 'name' | 'watchHasEnglish' | 'watchHasSub'>) {
  if (item.watchHasEnglish || item.watchHasSub) return true
  const tag = extractLangTag(item.name)
  return tag?.variant === 'en' || tag === null
}

function formatLanguageSummary(item: Pick<MediaResult, 'name' | 'hasPortuguese' | 'hasEnglish' | 'hasDub' | 'hasSub' | 'watchHasPortuguese' | 'watchHasEnglish' | 'watchHasDub' | 'watchHasSub'>) {
  const watchPt = watchHasPortugueseSignal(item)
  const watchEn = watchHasSubtitleSignal(item)
  const groupPt = hasPortugueseSignal(item)

  let label = 'Idioma n\u00e3o identificado'
  if (item.watchHasDub && item.watchHasSub && watchPt) label = 'PT-BR dublado e legendado'
  else if (item.watchHasDub && watchPt) label = 'PT-BR dublado'
  else if (item.watchHasSub && watchPt) label = 'PT-BR legendado'
  else if (watchPt && watchEn) label = 'PT-BR e ingl\u00eas'
  else if (watchPt) label = 'Portugu\u00eas do Brasil'
  else if (item.watchHasSub || watchEn) label = 'Legendado / ingl\u00eas'
  else {
    const tag = extractLangTag(item.name)
    if (tag?.variant === 'pt') label = 'Portugu\u00eas do Brasil'
    else if (tag?.variant === 'en') label = 'Legendado / ingl\u00eas'
  }

  if (!watchPt && groupPt) {
    return `${label} \u2022 PT-BR em outra fonte`
  }
  return label
}

function formatCardTitle(item: Pick<MediaResult, 'name' | 'canonicalTitle' | 'watchHasPortuguese' | 'watchHasEnglish' | 'watchHasDub' | 'watchHasSub'>) {
  const baseTitle = cleanTitle(item.name)
  if (/^\[(PT-BR|EN|PT-BR\/EN)\]\s*/i.test(baseTitle)) return baseTitle

  const hasPt = watchHasPortugueseSignal(item)
  const hasEn = watchHasSubtitleSignal(item)
  if (hasPt && hasEn) return `[PT-BR/EN] ${baseTitle}`
  if (hasPt) return `[PT-BR] ${baseTitle}`
  if (hasEn) return `[EN] ${baseTitle}`
  return baseTitle
}

function readPref(key: string, fallback: string): string {
  try {
    return localStorage.getItem(key) ?? fallback
  } catch {
    return fallback
  }
}

function writePref(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // localStorage can be unavailable in some WebView contexts.
  }
}

function readJson<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return fallback
    return JSON.parse(raw) as T
  } catch {
    return fallback
  }
}

function writeJson(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // localStorage can be unavailable in some WebView contexts.
  }
}

function toRequest(item: MediaResult): MediaRequest {
  return {
    name: item.name,
    url: item.url,
    source: item.source,
    mediaType: item.mediaType,
    groupKey: item.groupKey,
  }
}

function formatSourceLabel(source: string) {
  const match = sources.find(item => item.value === source)
  return match?.label ?? source
}

function createProgressId(item: Pick<MediaRequest, 'source' | 'url'>): string {
  const source = item.source.toLowerCase()
  if (source === 'allanime') {
    const match = item.url.match(/\/anime\/([^/?#]+)/i)
    if (match?.[1]) {
      try {
        return decodeURIComponent(match[1])
      } catch {
        return match[1]
      }
    }
  }
  return `${item.source}:${item.url}`
}

function favoriteKey(source: string, url: string) {
  return `${source}:${url}`
}

function getEpisodeNumber(ep: EpisodeResult | undefined, fallbackIndex: number) {
  if (!ep) return fallbackIndex + 1
  if (ep.num > 0) return ep.num
  const parsed = Number.parseInt(ep.number, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallbackIndex + 1
}

function formatPercent(value: number) {
  return `${Math.round(value)}%`
}

function formatEpisodeCounter(entry: WatchProgressEntry) {
  return entry.totalEpisodes > 0
    ? `Ep ${entry.episodeNumber}/${entry.totalEpisodes}`
    : `Ep ${entry.episodeNumber}`
}

function formatRemainingEpisodes(entry: WatchProgressEntry) {
  if (entry.totalEpisodes <= 0) return `${formatPercent(entry.progressPercent)} assistido`
  if (entry.remainingEpisodes <= 0) return 'final'
  return entry.remainingEpisodes === 1 ? 'falta 1' : `faltam ${entry.remainingEpisodes}`
}

function enrichProgressEntries(entries: WatchProgressEntry[]) {
  const meta = readJson<Record<string, { totalEpisodes: number; remainingEpisodes: number }>>(progressMetaKey, {})
  return entries.map(entry => {
    const cached = meta[entry.allanimeId]
    if (!cached) return entry

    return {
      ...entry,
      totalEpisodes: entry.totalEpisodes || cached.totalEpisodes || 0,
      remainingEpisodes: entry.remainingEpisodes || cached.remainingEpisodes || 0,
    }
  })
}

function persistProgressMeta(entry: Pick<WatchProgressEntry, 'allanimeId' | 'totalEpisodes' | 'remainingEpisodes'>) {
  if (!entry.allanimeId || entry.totalEpisodes <= 0) return

  const meta = readJson<Record<string, { totalEpisodes: number; remainingEpisodes: number }>>(progressMetaKey, {})
  meta[entry.allanimeId] = {
    totalEpisodes: entry.totalEpisodes,
    remainingEpisodes: Math.max(entry.remainingEpisodes, 0),
  }
  writeJson(progressMetaKey, meta)
}

function findProgressEntry(item: MediaResult | null, entries: WatchProgressEntry[]) {
  if (!item) return null

  const progressId = createProgressId(item)
  const byId = entries.find(entry => entry.allanimeId === progressId)
  if (byId) return byId

  const titleClean = cleanTitle(item.name).toLowerCase()
  return entries.find(entry => {
    const entryTitle = entry.title.toLowerCase()
    return titleClean.includes(entryTitle.slice(0, 12)) || entryTitle.includes(titleClean.slice(0, 12))
  }) ?? null
}

function upsertProgressEntry(
  entries: WatchProgressEntry[],
  nextEntry: WatchProgressEntry,
) {
  const filtered = entries.filter(entry => entry.allanimeId !== nextEntry.allanimeId)
  return [nextEntry, ...filtered].sort((a, b) => b.lastUpdated.localeCompare(a.lastUpdated))
}

const IconSearch = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="11" cy="11" r="8" />
    <line x1="21" y1="21" x2="16.65" y2="16.65" />
  </svg>
)

const IconClock = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="history-icon">
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
  </svg>
)

const IconHeart = ({ active = false }: { active?: boolean }) => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill={active ? 'currentColor' : 'none'} stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M12 21s-6.716-4.35-9.193-8.066C.873 9.916 2.09 5.5 6.228 5.5c2.016 0 3.205 1.243 3.772 2.217C10.567 6.743 11.756 5.5 13.772 5.5c4.138 0 5.355 4.416 3.421 7.434C18.716 16.65 12 21 12 21z" />
  </svg>
)

const IconPlay = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
    <polygon points="5 3 19 12 5 21 5 3" />
  </svg>
)
const IconSkip = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polygon points="5 4 15 12 5 20 5 4" />
    <line x1="19" y1="5" x2="19" y2="19" />
  </svg>
)

const IconMinimise = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <line x1="5" y1="12" x2="19" y2="12" />
  </svg>
)

const IconMaximise = ({ active }: { active: boolean }) => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    {active ? (
      <>
        <path d="M9 5H5v4" />
        <path d="M15 5h4v4" />
        <path d="M19 15v4h-4" />
        <path d="M9 19H5v-4" />
      </>
    ) : (
      <rect x="5" y="5" width="14" height="14" rx="1" />
    )}
  </svg>
)

const IconClose = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <line x1="6" y1="6" x2="18" y2="18" />
    <line x1="18" y1="6" x2="6" y2="18" />
  </svg>
)

const IconClear = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <line x1="6" y1="6" x2="18" y2="18" />
    <line x1="18" y1="6" x2="6" y2="18" />
  </svg>
)

const IconExpand = ({ active = false }: { active?: boolean }) => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    {active ? (
      <>
        <path d="M9 9H5V5" />
        <path d="M15 9h4V5" />
        <path d="M9 15H5v4" />
        <path d="M15 15h4v4" />
      </>
    ) : (
      <>
        <path d="M15 3h6v6" />
        <path d="M9 21H3v-6" />
        <path d="M21 3l-7 7" />
        <path d="M3 21l7-7" />
      </>
    )}
  </svg>
)

const IconDownload = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M12 3v12" />
    <path d="m7 10 5 5 5-5" />
    <path d="M5 21h14" />
  </svg>
)

const IconHome = ({ active = false }: { active?: boolean }) => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill={active ? 'currentColor' : 'none'} stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
    <polyline points="9 22 9 12 15 12 15 22" />
  </svg>
)

const IconFilm = () => (
  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" style={{ opacity: 0.28 }}>
    <rect x="2" y="2" width="20" height="20" rx="2" />
    <path d="M7 2v20M17 2v20M2 12h20M2 7h5M17 7h5M2 17h5M17 17h5" />
  </svg>
)

const IconMovie = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M2 7h20v10H2z" />
    <path d="M7 7 9.5 17" />
    <path d="M14.5 7 17 17" />
  </svg>
)

export default function App() {
  const videoRef = useRef<HTMLVideoElement | null>(null)
  const playerWrapRef = useRef<HTMLDivElement | null>(null)
  const episodeListRef = useRef<HTMLDivElement | null>(null)
  const hlsRef = useRef<Hls | null>(null)
  const lastSavedSecondRef = useRef(0)
  const toastIdRef = useRef(0)

  const [query, setQuery] = useState('')
  const [source, setSource] = useState<SourceFilter>('all')
  const [mediaType, setMediaType] = useState<TypeFilter>('all')
  const [langFilter, setLangFilter] = useState<LangFilter>('all')

  const [searchState, setSearchState] = useState<LoadState>('idle')
  const [results, setResults] = useState<MediaResult[]>([])
  const [searchMessage, setSearchMessage] = useState('')

  const [history, setHistory] = useState<HistoryEntry[]>([])
  const [progress, setProgress] = useState<WatchProgressEntry[]>([])
  const [favorites, setFavorites] = useState<FavoriteEntry[]>([])
  const [catalog, setCatalog] = useState<CatalogSection[]>([])
  const [catalogLoading, setCatalogLoading] = useState(true)
  const [movieCatalog, setMovieCatalog] = useState<CatalogSection[]>([])
  const [movieCatalogLoading, setMovieCatalogLoading] = useState(true)
  const [movieCatalogError, setMovieCatalogError] = useState('')

  const [activeMedia, setActiveMedia] = useState<MediaResult | null>(null)
  const [episodes, setEpisodes] = useState<EpisodeResult[]>([])
  const [episodeIndex, setEpisodeIndex] = useState(0)

  const [mode, setMode] = useState<'sub' | 'dub'>(() => readPref('pref-mode', 'sub') as 'sub' | 'dub')
  const [quality, setQuality] = useState(() => readPref('pref-quality', 'best'))

  const [playerMessage, setPlayerMessage] = useState('Selecione um titulo para começar.')
  const [playerState, setPlayerState] = useState<LoadState>('idle')
  const [isMaximised, setIsMaximised] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [isDownloading, setIsDownloading] = useState(false)
  const [view, setView] = useState<ViewMode>('catalog')
  const [movieSourceMode, setMovieSourceMode] = useState<MovieSourceMode>('ptbr')
  const [toasts, setToasts] = useState<Toast[]>([])
  const [related, setRelated] = useState<RelatedAnime[]>([])
  const [relatedLoading, setRelatedLoading] = useState(false)

  const trimmedQuery = useMemo(() => query.trim(), [query])
  const sourceOptions = useMemo(() => {
    if (view !== 'movies') return sources
    return movieSourceMode === 'extra' ? movieSourcesExtra : movieSourcesPrimary
  }, [view, movieSourceMode])

  const displayResults = useMemo(() => {
    if (langFilter === 'all') return results
    return results.filter(item => {
      if (langFilter === 'pt') return hasPortugueseSignal(item)
      if (langFilter === 'en') return hasSubtitleSignal(item)
      return true
    })
  }, [results, langFilter])

  const visibleResults = useMemo(() => {
    if (view === 'movies' && movieSourceMode === 'ptbr') {
      return displayResults.filter(item => item.source.toLowerCase() !== 'flixhq')
    }
    return displayResults
  }, [displayResults, view, movieSourceMode])

  const movieFavorites = useMemo(() => favorites.filter(item => item.mediaType === 'movie'), [favorites])
  const movieProgress = useMemo(() => progress.filter(item => item.mediaType === 'movie'), [progress])
  const canNext = episodes.length > 0 && episodeIndex < episodes.length - 1
  const currentEpisode = episodes[episodeIndex] ?? null
  const activeProgress = useMemo(() => findProgressEntry(activeMedia, progress), [activeMedia, progress])
  const favoriteKeys = useMemo(() => new Set(favorites.map(item => favoriteKey(item.source, item.url))), [favorites])
  const hasSidebarContent = favorites.length > 0 || progress.length > 0 || history.length > 0
  const isActiveFavorite = activeMedia ? favoriteKeys.has(favoriteKey(activeMedia.source, activeMedia.url)) : false
  const searchPlaceholder = view === 'movies' ? 'Buscar filme...' : 'Buscar anime ou série...'

  const showToast = (message: string, type: ToastType = 'info') => {
    const id = ++toastIdRef.current
    setToasts(prev => [...prev, { id, message, type }])
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 3200)
  }

  useEffect(() => {
    document.title = activeMedia ? `${cleanTitle(activeMedia.name)} — GoAnime` : 'GoAnime Desktop'
  }, [activeMedia])

  useEffect(() => { writePref('pref-mode', mode) }, [mode])
  useEffect(() => { writePref('pref-quality', quality) }, [quality])

  useEffect(() => {
    const syncWindowState = async () => {
      try {
        setIsMaximised(await WindowIsMaximised())
      } catch {
        setIsMaximised(false)
      }

      try {
        setIsFullscreen(document.fullscreenElement !== null || await WindowIsFullscreen())
      } catch {
        setIsFullscreen(document.fullscreenElement !== null)
      }
    }

    const onFullscreenChange = () => {
      setIsFullscreen(document.fullscreenElement !== null)
    }

    void syncWindowState()
    window.addEventListener('resize', syncWindowState)
    window.addEventListener('focus', syncWindowState)
    document.addEventListener('fullscreenchange', onFullscreenChange)
    return () => {
      window.removeEventListener('resize', syncWindowState)
      window.removeEventListener('focus', syncWindowState)
      document.removeEventListener('fullscreenchange', onFullscreenChange)
    }
  }, [])

  useEffect(() => {
    getProxyBaseURL().catch(() => {})
    getSearchHistory().then(items => setHistory(items ?? [])).catch(() => {})
    getWatchProgress().then(items => setProgress(enrichProgressEntries(items ?? []))).catch(() => {})
    getFavorites().then(items => setFavorites(items ?? [])).catch(() => {})
    getCatalog()
      .then(setCatalog)
      .catch(() => {})
      .finally(() => setCatalogLoading(false))

    getMovieCatalog()
      .then(items => {
        setMovieCatalog(items ?? [])
        setMovieCatalogError('')
      })
      .catch(err => {
        setMovieCatalog([])
        setMovieCatalogError(err instanceof Error ? err.message : 'Falha ao carregar catálogo de filmes')
      })
      .finally(() => setMovieCatalogLoading(false))
  }, [])

  useEffect(() => {
    if (!sourceOptions.some(item => item.value === source)) {
      setSource(sourceOptions[0]?.value ?? 'all')
    }
  }, [sourceOptions, source])

  useEffect(() => {
    const unsubscribe = EventsOn('search:covers', (event: SearchCoversEvent) => {
      if (!event) return
      if (event.query.trim().toLowerCase() !== trimmedQuery.toLowerCase()) return
      if (event.source.trim().toLowerCase() !== source.toLowerCase()) return
      if (event.mediaType.trim().toLowerCase() !== mediaType.toLowerCase()) return

      const updates = new Map(
        event.results.map(item => [favoriteKey(item.source, item.url), item] as const),
      )
      if (updates.size === 0) return

      setResults(current => current.map(item => {
        const upd = updates.get(favoriteKey(item.source, item.url))
        if (!upd) return item
        return {
          ...item,
          imageUrl: upd.imageUrl || item.imageUrl,
          score: upd.score ?? item.score,
          description: upd.description || item.description,
          genres: upd.genres?.length ? upd.genres : item.genres,
        }
      }))
    })

    return () => unsubscribe()
  }, [trimmedQuery, source, mediaType])

  useEffect(() => {
    if (trimmedQuery.length < 2) {
      setResults([])
      setSearchState('idle')
      setSearchMessage('')
      return
    }

    if (view === 'favorites' || view === 'watching') {
      setView('catalog')
    }
    let cancelled = false
    const timer = setTimeout(async () => {
      setSearchState('loading')
      setSearchMessage('Buscando...')
      try {
        const data = await searchMedia(trimmedQuery, source, mediaType)
        if (cancelled) return
        setResults(data)
        setSearchState('ready')
        setSearchMessage(data.length ? `${data.length} resultado(s)` : 'Nenhum resultado encontrado.')
      } catch (err) {
        if (cancelled) return
        setSearchState('error')
        setResults([])
        setSearchMessage(err instanceof Error ? err.message : 'Falha na busca')
      }
    }, 280)

    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [trimmedQuery, source, mediaType])

  useEffect(() => () => { hlsRef.current?.destroy() }, [])

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const tag = (event.target as HTMLElement).tagName
      if (tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA') return

      if ((event.key === 'ArrowRight' || event.key.toLowerCase() === 'n') && canNext) {
        setEpisodeIndex(index => index + 1)
      }
      if ((event.key === 'ArrowLeft' || event.key.toLowerCase() === 'p') && episodeIndex > 0) {
        setEpisodeIndex(index => index - 1)
      }
      if (event.key.toLowerCase() === 'f') {
        void handleToggleFullscreen()
      }
    }

    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [canNext, episodeIndex, isFullscreen])

  useEffect(() => {
    const list = episodeListRef.current
    if (!list) return
    const active = list.querySelector<HTMLElement>('.ep-item.current')
    if (active) active.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  }, [episodeIndex])

  const isFavorite = (item: Pick<MediaResult, 'source' | 'url'>) => favoriteKeys.has(favoriteKey(item.source, item.url))

  const persistCurrentProgress = async (forceComplete = false) => {
    if (!activeMedia || !currentEpisode) return

    const video = videoRef.current
    if (!video || !Number.isFinite(video.duration) || video.duration <= 0) return

    const duration = Math.max(1, Math.floor(video.duration))
    const playbackTime = forceComplete
      ? duration
      : Math.min(duration, Math.max(0, Math.floor(video.currentTime)))
    const progressEntry: WatchProgressEntry = {
      allanimeId: createProgressId(activeMedia),
      title: cleanTitle(activeMedia.name),
      episodeNumber: getEpisodeNumber(currentEpisode, episodeIndex),
      playbackTime,
      duration,
      progressPercent: duration > 0 ? (playbackTime / duration) * 100 : 0,
      totalEpisodes: episodes.length,
      remainingEpisodes: Math.max(episodes.length - getEpisodeNumber(currentEpisode, episodeIndex), 0),
      mediaType: activeMedia.mediaType,
      lastUpdated: new Date().toISOString(),
    }

    persistProgressMeta(progressEntry)
    setProgress(current => upsertProgressEntry(current, progressEntry))

    try {
      await updateWatchProgress({
        allanimeId: progressEntry.allanimeId,
        title: progressEntry.title,
        episodeNumber: progressEntry.episodeNumber,
        playbackTime: progressEntry.playbackTime,
        duration: progressEntry.duration,
        mediaType: progressEntry.mediaType,
      })
    } catch {
      getWatchProgress().then(items => setProgress(enrichProgressEntries(items ?? []))).catch(() => {})
    }
  }

  const handleVideoTimeUpdate = () => {
    const video = videoRef.current
    if (!video) return

    const currentSecond = Math.floor(video.currentTime)
    if (Math.abs(currentSecond - lastSavedSecondRef.current) < 5) return

    lastSavedSecondRef.current = currentSecond
    void persistCurrentProgress(false)
  }

  const handleVideoEnded = async () => {
    await persistCurrentProgress(true)
    if (!canNext) return

    const nextIndex = episodeIndex + 1
    setEpisodeIndex(nextIndex)
    setPlayerMessage('Avançando para o próximo episodio...')
    void startPlayback(nextIndex)
  }

  const handleToggleFavorite = async (item: MediaResult) => {
    const entry: FavoriteEntry = {
      title: cleanTitle(item.name),
      imageUrl: item.imageUrl,
      url: item.url,
      source: item.source,
      mediaType: item.mediaType,
      addedAt: new Date().toISOString(),
    }

    if (isFavorite(item)) {
      setFavorites(current => current.filter(fav => favoriteKey(fav.source, fav.url) !== favoriteKey(item.source, item.url)))
      showToast('Removido dos favoritos', 'info')
      try {
        await removeFavorite(entry.title)
      } catch {
        getFavorites().then(items => setFavorites(items ?? [])).catch(() => {})
      }
      return
    }

    setFavorites(current => [entry, ...current.filter(fav => favoriteKey(fav.source, fav.url) !== favoriteKey(item.source, item.url))])
    showToast('Adicionado aos favoritos', 'success')
    try {
      await addFavorite(entry)
    } catch {
      getFavorites().then(items => setFavorites(items ?? [])).catch(() => {})
    }
  }

  const dubSources = new Set(['animefire', 'animesonlinecc'])

  const handleOpenMedia = async (item: MediaResult) => {
    // Auto-detect audio mode from title tag or source
    const langTag = extractLangTag(item.name)
    const isDubSource = dubSources.has(item.source.toLowerCase())
    if (item.watchHasDub || langTag?.variant === 'pt' || isDubSource) {
      setMode('dub')
    } else if (item.watchHasSub || item.watchHasEnglish || langTag?.variant === 'en') {
      setMode('sub')
    }

    setActiveMedia(item)
    setEpisodes([])
    setEpisodeIndex(0)
    setRelated([])
    setRelatedLoading(true)
    setPlayerState('loading')
    setPlayerMessage('Carregando episódios...')
    lastSavedSecondRef.current = 0

    // Fetch related anime in background (doesn't block episodes loading)
    getRelatedAnime(cleanTitle(item.name))
      .then(items => setRelated(items ?? []))
      .catch(() => {})
      .finally(() => setRelatedLoading(false))

    try {
      const response = await getEpisodes(toRequest(item))
      setEpisodes(response.episodes)
      if (response.resolvedSource && response.resolvedUrl) {
        setActiveMedia(current => current
          ? { ...current, source: response.resolvedSource!, url: response.resolvedUrl! }
          : current)
      }
      if (response.note) {
        showToast(response.note, 'info')
      }

      let startIndex = 0
      const match = findProgressEntry(item, progress)
      if (match) {
        const index = response.episodes.findIndex(ep => getEpisodeNumber(ep, 0) === match.episodeNumber)
        if (index >= 0) {
          startIndex = index
        }

        const nextProgress = {
          ...match,
          totalEpisodes: response.episodes.length,
          remainingEpisodes: Math.max(response.episodes.length - match.episodeNumber, 0),
        }
        persistProgressMeta(nextProgress)
        setProgress(current => upsertProgressEntry(current, nextProgress))
      }

      setEpisodeIndex(startIndex)
      setPlayerState('ready')
      const readyMessage = response.episodes.length
        ? `${response.episodes.length} episodio(s) - clique em Assistir`
        : 'Nenhum episodio encontrado.'
      setPlayerMessage(response.note ? `${response.note} ${readyMessage}` : readyMessage)
    } catch (err) {
      setPlayerState('error')
      setPlayerMessage(err instanceof Error ? err.message : 'Erro ao carregar episódios')
    }
  }

  const applySubtitles = (
    video: HTMLVideoElement,
    subtitles: Array<{ proxyUrl: string; url: string; label: string; language: string }> = [],
  ) => {
    Array.from(video.querySelectorAll('track')).forEach(track => track.remove())
    subtitles.forEach(subtitle => {
      const src = subtitle.proxyUrl || subtitle.url
      if (!src) return

      const track = document.createElement('track')
      track.kind = 'subtitles'
      track.label = subtitle.label || subtitle.language || 'Subtitle'
      track.srclang = (subtitle.language || 'en').slice(0, 2).toLowerCase()
      track.src = src
      video.appendChild(track)
    })
  }

  const loadVideo = async (
    url: string,
    contentType: string,
    subtitles: Array<{ proxyUrl: string; url: string; label: string; language: string }> = [],
    resumeAt = 0,
  ) => {
    const video = videoRef.current
    if (!video) return

    hlsRef.current?.destroy()
    hlsRef.current = null
    video.pause()
    video.removeAttribute('src')
    video.load()
    applySubtitles(video, subtitles)

    if (resumeAt > 0) {
      const restorePlayback = () => {
        const safeTime = Math.max(0, Math.min(resumeAt, Math.max(0, video.duration - 2)))
        if (safeTime > 0) {
          video.currentTime = safeTime
        }
      }
      video.addEventListener('loadedmetadata', restorePlayback, { once: true })
    }

    const isHls = url.toLowerCase().includes('.m3u8') || contentType.toLowerCase().includes('mpegurl')

    if (isHls && Hls.isSupported()) {
      const hls = new Hls({ maxBufferLength: 30, enableWorker: true })
      hlsRef.current = hls
      hls.loadSource(url)
      hls.attachMedia(video)
      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          setPlayerMessage(`Falha no stream (${data.type}).`)
          setPlayerState('error')
        }
      })
    } else {
      video.src = url
    }

    try {
      await video.play()
      setPlayerMessage('Reprodu\u00e7\u00e3o iniciada.')
      setPlayerState('ready')
    } catch {
      setPlayerMessage('Pronto - pressione Play.')
      setPlayerState('ready')
    }
  }

  const startPlayback = async (targetIndex: number) => {
    if (!activeMedia) return

    const episode = episodes[targetIndex]
    if (!episode) return

    setPlayerState('loading')
    setPlayerMessage('Buscando stream...')

    try {
      const stream = await getStream({
        media: toRequest(activeMedia),
        episodeUrl: episode.url,
        episodeNumber: episode.num ? String(episode.num) : episode.number,
        mode,
        quality,
      })

      if (stream.resolvedSource && stream.resolvedUrl) {
        setActiveMedia(current => current
          ? { ...current, source: stream.resolvedSource!, url: stream.resolvedUrl! }
          : current)
      }
      if (stream.resolvedEpisodeUrl) {
        setEpisodes(current => current.map((item, index) => index === targetIndex ? { ...item, url: stream.resolvedEpisodeUrl! } : item))
      }
      if (stream.note) {
        showToast(stream.note, 'info')
      }

      const savedEntry = findProgressEntry(activeMedia, progress)
      const resumeAt = savedEntry && savedEntry.episodeNumber === getEpisodeNumber(episode, targetIndex)
        ? savedEntry.playbackTime
        : 0

      setEpisodeIndex(targetIndex)
      lastSavedSecondRef.current = resumeAt
      await loadVideo(
        stream.proxyUrl || stream.streamUrl,
        stream.contentType || 'video/*',
        stream.subtitles,
        resumeAt,
      )
    } catch (err) {
      setPlayerMessage(err instanceof Error ? err.message : 'Erro ao carregar stream')
      setPlayerState('error')
    }
  }

  const handlePlay = async () => {
    await startPlayback(episodeIndex)
  }

  const handleDownload = async () => {
    if (!activeMedia || !currentEpisode) return

    setIsDownloading(true)
    setPlayerMessage('Baixando episodio...')
    try {
      const response = await downloadEpisode({
        media: toRequest(activeMedia),
        episodeUrl: currentEpisode.url,
        episodeNumber: currentEpisode.num ? String(currentEpisode.num) : currentEpisode.number,
        mode,
        quality,
      })
      setPlayerMessage(`${response.message}: ${response.filePath}`)
      showToast('Download conclu\u00eddo', 'success')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro ao baixar episodio'
      setPlayerMessage(msg)
      showToast(msg, 'error')
    } finally {
      setIsDownloading(false)
    }
  }

  const handleToggleMaximise = async () => {
    try {
      WindowToggleMaximise()
      setTimeout(async () => {
        try {
          setIsMaximised(await WindowIsMaximised())
        } catch {
          setIsMaximised(false)
        }
      }, 80)
    } catch {
      setIsMaximised(false)
    }
  }

  const handleToggleFullscreen = async () => {
    try {
      if (document.fullscreenElement) {
        await document.exitFullscreen()
        setIsFullscreen(false)
        return
      }

      if (playerWrapRef.current?.requestFullscreen) {
        await playerWrapRef.current.requestFullscreen()
        setIsFullscreen(true)
        return
      }
    } catch {
      // Fall through to window fullscreen.
    }

    try {
      if (await WindowIsFullscreen()) {
        WindowUnfullscreen()
        setIsFullscreen(false)
      } else {
        WindowFullscreen()
        setIsFullscreen(true)
      }
    } catch {
      setIsFullscreen(document.fullscreenElement !== null)
    }
  }

  const canPlay = activeMedia !== null && episodes.length > 0 && playerState !== 'loading'
  const canDownload = activeMedia !== null && currentEpisode !== null && !isDownloading

  return (
    <div className={`app${activeMedia === null ? ' app--no-player' : ''}`}>
      {/* Sidebar */}
      <aside className="sidebar">
        <div className="logo">
          <div className="logo-icon">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="white"><polygon points="5 3 19 12 5 21 5 3" /></svg>
          </div>
          <div className="logo-text">
            <span className="logo-go">Go</span><span className="logo-anime">Anime</span>
          </div>
          <span className="logo-badge">Desktop</span>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-nav">
            <button
              className={`sidebar-nav-btn${view === 'catalog' && !query ? ' active' : ''}`}
              onClick={() => {
                setView('catalog')
                setQuery('')
                setMediaType('anime')
                setLangFilter('all')
                setSource('all')
              }}
            >
              <IconHome active={view === 'catalog' && !query} />
              <span>Inicio</span>
            </button>
            <button
              className={`sidebar-nav-btn${view === 'movies' && !query ? ' active' : ''}`}
              onClick={() => {
                setView('movies')
                setQuery('')
                setMediaType('movie')
                setLangFilter('all')
                setSource('all')
              }}
            >
              <IconMovie />
              <span>Filmes</span>
            </button>
            <button
              className={`sidebar-nav-btn${view === 'favorites' ? ' active' : ''}`}
              onClick={() => { setView('favorites'); setQuery('') }}
            >
              <IconHeart active={view === 'favorites'} />
              <span>Favoritos</span>
              {favorites.length > 0 && <span className="sidebar-count">{favorites.length}</span>}
            </button>
            <button
              className={`sidebar-nav-btn${view === 'watching' ? ' active' : ''}`}
              onClick={() => { setView('watching'); setQuery('') }}
            >
              <IconClock />
              <span>Continuar assistindo</span>
              {progress.length > 0 && <span className="sidebar-count">{progress.length}</span>}
            </button>
          </div>

          {history.length > 0 && (
            <>
              <p className="sidebar-label">Recentes</p>
              {history.map(entry => (
                <button key={entry.name} className="history-item" onClick={() => setQuery(entry.name)} title={entry.name}>
                  <IconClock />
                  <div className="history-item-body">
                    <span className="history-item-text">{entry.name}</span>
                  </div>
                </button>
              ))}
            </>
          )}
        </div>
      </aside>

      {/* Topbar */}
      <header className="topbar">
        <div className="topbar-main">
          <div className="titlebar-drag" />
          <div className="search-box">
            <IconSearch />
            <input
              value={query}
              onChange={event => setQuery(event.target.value)}
              placeholder={searchPlaceholder}
              autoFocus
            />
            {query && (
              <button className="search-clear" type="button" onClick={() => setQuery('')} title="Limpar busca" aria-label="Limpar busca">
                <IconClear />
              </button>
            )}
          </div>
          <div className="window-controls">
            <button className="wc-btn" type="button" aria-label="Minimizar janela" title="Minimizar" onClick={() => WindowMinimise()}>
              <IconMinimise />
            </button>
            <button className="wc-btn" type="button" aria-label="Maximizar janela" title={isMaximised ? 'Restaurar' : 'Maximizar'} onClick={() => void handleToggleMaximise()}>
              <IconMaximise active={isMaximised} />
            </button>
            <button className="wc-btn wc-close" type="button" aria-label="Fechar janela" title="Fechar" onClick={() => Quit()}>
              <IconClose />
            </button>
          </div>
        </div>

        {(trimmedQuery.length >= 2 || view === 'movies') && <div className="topbar-filters">
          <div className="filter-pills">
            {view === 'movies' ? (
              <>
                <button className="pill active" type="button">Filmes</button>
                <button
                  className={`pill${movieSourceMode === 'ptbr' ? ' active' : ''}`}
                  type="button"
                  onClick={() => setMovieSourceMode('ptbr')}
                >
                  PT-BR primeiro
                </button>
                <button
                  className={`pill${movieSourceMode === 'extra' ? ' active' : ''}`}
                  type="button"
                  onClick={() => setMovieSourceMode('extra')}
                >
                  Fontes extras
                </button>
              </>
            ) : (
              <>
                {browseTypeOptions.map(type => (
                  <button key={type} className={`pill${mediaType === type ? ' active' : ''}`} onClick={() => setMediaType(type)}>
                    {typeLabels[type]}
                  </button>
                ))}
                <div className="filter-sep" />
                <button className={`pill pill-lang${langFilter === 'all' ? ' active' : ''}`} onClick={() => setLangFilter('all')}>Todos</button>
                <button className={`pill pill-lang pill-pt${langFilter === 'pt' ? ' active' : ''}`} onClick={() => setLangFilter('pt')}>PT-BR</button>
                <button className={`pill pill-lang pill-en${langFilter === 'en' ? ' active' : ''}`} onClick={() => setLangFilter('en')}>Legendado</button>
              </>
            )}
          </div>

          <div className="source-select">
            <select value={source} onChange={event => setSource(event.target.value as SourceFilter)}>
              {sourceOptions.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}
            </select>
          </div>
        </div>}
      </header>

      {/* Conteúdo principal */}
      <main className="main">
        {searchMessage && (
          <div className="status-bar">
            <span className={`status-dot ${searchState}`} />
            <span>{searchMessage}</span>
          </div>
        )}

        {searchState === 'loading' && (
          <div className="results-grid">
            {Array.from({ length: 8 }).map((_, index) => (
              <div key={index} className="skeleton-card">
                <div className="skeleton-cover" />
                <div className="skeleton-line" />
                <div className="skeleton-line short" />
              </div>
            ))}
          </div>
        )}

        {view === 'favorites' && searchState === 'idle' && trimmedQuery.length < 2 && (
          <div className="favorites-page">
            <div className="page-header">
              <h2>Favoritos</h2>
              <span className="page-header-count">{favorites.length} {'t\u00edtulo(s)'}</span>
            </div>
            {favorites.length === 0 ? (
              <div className="empty-state">
                <div className="empty-icon">{'\u2661'}</div>
                <h3>Nenhum favorito ainda</h3>
                <p>Busque um {'t\u00edtulo'} e clique no {'cora\u00e7\u00e3o'} para salvar nos favoritos.</p>
              </div>
            ) : (
              <div className="results-grid">
                {favorites.map((fav, index) => (
                  <article
                    key={favoriteKey(fav.source, fav.url)}
                    className={`media-card${activeMedia?.url === fav.url ? ' active' : ''}`}
                    style={{ '--card-i': index } as React.CSSProperties}
                    onClick={() => void handleOpenMedia({
                      name: fav.title,
                      imageUrl: fav.imageUrl,
                      url: fav.url,
                      source: fav.source,
                      mediaType: fav.mediaType as MediaResult['mediaType'],
                    })}
                    title={fav.title}
                  >
                    <div className="cover">
                      <button
                        className="btn-fav active"
                        type="button"
                        onClick={e => {
                          e.stopPropagation()
                          void handleToggleFavorite({
                            name: fav.title,
                            imageUrl: fav.imageUrl,
                            url: fav.url,
                            source: fav.source,
                            mediaType: fav.mediaType as MediaResult['mediaType'],
                          })
                        }}
                        title="Remover dos favoritos"
                        aria-label="Remover dos favoritos"
                      >
                        <IconHeart active />
                      </button>
                      {fav.imageUrl
                        ? <img src={fav.imageUrl} alt={fav.title} loading="lazy" />
                        : <div className="cover-fallback"><IconFilm /></div>}
                      <div className="cover-overlay">
                        <span className="chip chip-type">{typeLabels[fav.mediaType as TypeFilter] ?? fav.mediaType}</span>
                        <span className="chip chip-source">{fav.source}</span>
                      </div>
                    </div>
                    <div className="card-info">
                      <p className="card-title">{fav.title}</p>
                      {(() => {
                        const p = progress.find(e => e.title.toLowerCase() === fav.title.toLowerCase() || fav.title.toLowerCase().includes(e.title.toLowerCase().slice(0, 12)))
                        if (!p || p.progressPercent <= 0) return null
                        return (
                          <div className="card-progress">
                            <span className="card-progress-label">{formatEpisodeCounter(p)} {'\u00b7'} {formatPercent(p.progressPercent)}</span>
                            <div className="ep-progress-wrap">
                              <div className="ep-progress-bar" style={{ width: `${Math.max(p.progressPercent, 4)}%` }} />
                            </div>
                          </div>
                        )
                      })()}
                    </div>
                  </article>
                ))}
              </div>
            )}
          </div>
        )}

        {view === 'watching' && searchState === 'idle' && trimmedQuery.length < 2 && (
          <div className="favorites-page">
            <div className="page-header">
              <h2>Continuar assistindo</h2>
              <span className="page-header-count">{progress.length} {'t\u00edtulo(s)'}</span>
            </div>
            {progress.length === 0 ? (
              <div className="empty-state">
                <div className="empty-icon">{'\u25b6'}</div>
                <h3>Nenhum progresso ainda</h3>
                <p>Comece a assistir um {'epis\u00f3dio'} para acompanhar aqui.</p>
              </div>
            ) : (
              <div className="watching-grid">
                {progress.map((entry, index) => (
                  <button
                    key={entry.allanimeId}
                    className={`watching-card${activeMedia && cleanTitle(activeMedia.name).toLowerCase().includes(entry.title.toLowerCase().slice(0, 12)) ? ' active' : ''}`}
                    style={{ '--card-i': index } as React.CSSProperties}
                    onClick={() => { setQuery(entry.title); setView('catalog') }}
                    title={entry.title}
                  >
                    <div className="watching-card-header">
                      <span className="watching-card-title">{entry.title}</span>
                      <span className="watching-ep-badge">{formatEpisodeCounter(entry)}</span>
                    </div>
                    <div className="watching-card-meta">
                      <span>{typeLabels[entry.mediaType as TypeFilter] ?? entry.mediaType}</span>
                      <span>{formatRemainingEpisodes(entry)}</span>
                      <span>{formatPercent(entry.progressPercent)} assistido</span>
                    </div>
                    {entry.progressPercent > 0 && (
                      <div className="ep-progress-wrap watching-progress">
                        <div className="ep-progress-bar" style={{ width: `${Math.max(entry.progressPercent, 2)}%` }} />
                      </div>
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {view === 'movies' && trimmedQuery.length < 2 && searchState === 'idle' && (
          <div className="movies-page">
            <div className="movie-hero-card">
              <div>
                <p className="movie-kicker">{'Se\u00e7\u00e3o de filmes'}</p>
                <h2>Filmes ficam em um fluxo separado do anime</h2>
                <p>
                  Aqui o app prioriza {'portugu\u00eas'} do Brasil. O FlixHQ continua dispon?vel apenas como
                  fonte extra, porque costuma entregar {'cat\u00e1logo'} em {'ingl\u00eas'}.
                </p>
              </div>
              <div className="movie-hero-actions">
                <button className={`btn btn-secondary${movieSourceMode === 'ptbr' ? ' movie-mode-active' : ''}`} type="button" onClick={() => setMovieSourceMode('ptbr')}>
                  Priorizar PT-BR
                </button>
                <button className={`btn btn-secondary${movieSourceMode === 'extra' ? ' movie-mode-active' : ''}`} type="button" onClick={() => setMovieSourceMode('extra')}>
                  Liberar FlixHQ
                </button>
              </div>
            </div>

            <div className="movie-info-grid">
              <div className="movie-info-card">
                <h3>Como essa {'se\u00e7\u00e3o'} funciona</h3>
                <p>O {'cat\u00e1logo'} abaixo vem do TMDb e serve para descoberta visual. Ele melhora capas, sinopses e destaque de filmes sem misturar isso com o fluxo de anime.</p>
                <p>Para assistir, clique em um {'t\u00edtulo'} do {'cat\u00e1logo'} e o app reaproveita a busca normal por nome.</p>
              </div>
              <div className="movie-info-card">
                <h3>Status atual</h3>
                <p>{movieSourceMode === 'ptbr' ? 'Modo principal ativo: a busca prioriza PT-BR e continua escondendo o FlixHQ.' : 'Modo extra ativo: o FlixHQ aparece como fallback e pode trazer resultados em ingl\u00eas.'}</p>
                <div className="movie-stats">
                  <span>{movieFavorites.length} favorito(s) de filme</span>
                  <span>{movieProgress.length} item(ns) com progresso</span>
                  <span>{movieCatalog.length} {'se\u00e7\u00e3o(\u00f5es)'} do TMDb</span>
                </div>
              </div>
            </div>

            {movieCatalogError ? (
              <div className="empty-state">
                <div className="empty-icon">TMDb</div>
                <h3>{'Cat\u00e1logo de filmes indispon\u00edvel'}</h3>
                <p>
                  {movieCatalogError.includes('TMDB_API_KEY')
                    ? 'Configure a vari\u00e1vel TMDB_API_KEY no Windows e abra o app de novo para carregar o cat\u00e1logo do TMDb.'
                    : movieCatalogError}
                </p>
              </div>
            ) : movieCatalog.length > 0 || movieCatalogLoading ? (
              <Catalog
                sections={movieCatalog}
                loading={movieCatalogLoading}
                onPlay={title => setQuery(title)}
              />
            ) : (
              <div className="empty-state">
                <div className="empty-icon">FILM</div>
                <h3>Nenhum destaque de filme chegou do TMDb</h3>
                <p>Tente abrir o app novamente em alguns minutos ou verificar a chave do TMDb.</p>
              </div>
            )}
          </div>
        )}

        {view === 'catalog' && trimmedQuery.length < 2 && searchState === 'idle' && (
          <Catalog
            sections={catalog}
            loading={catalogLoading}
            onPlay={title => setQuery(title)}
          />
        )}

        {results.length === 0 && searchState === 'ready' && (
          <div className="empty-state">
            <div className="empty-icon">?</div>
            <h3>{view === 'movies' ? 'Nenhum filme encontrado' : 'Nenhum titulo encontrado'}</h3>
            <p>
              {view === 'movies'
                ? 'Tente outro nome ou mude a fonte atual. Se quiser testar catálogo internacional, libere o FlixHQ nas fontes extras.'
                : 'Tente outro nome, mude a fonte ou remova filtros para ampliar a busca.'}
            </p>
          </div>
        )}

        {searchState !== 'loading' && results.length > 0 && view === 'movies' && visibleResults.length === 0 && (
          <div className="empty-state movie-empty-state">
            <div className="empty-icon">FILM</div>
            <h3>Nenhum filme PT-BR apareceu nas fontes principais</h3>
            <p>Se quiser testar catálogo internacional, ative o modo de fontes extras para liberar o FlixHQ.</p>
            <button className="btn btn-secondary movie-empty-action" type="button" onClick={() => setMovieSourceMode('extra')}>
              Liberar FlixHQ
            </button>
          </div>
        )}

        {searchState !== 'loading' && results.length > 0 && view !== 'movies' && visibleResults.length === 0 && (
          <div className="empty-state">
            <div className="empty-icon">GLOBE</div>
            <h3>Nenhum resultado nesse idioma</h3>
            <p>Tente mudar o filtro de idioma ou busque em outra fonte.</p>
          </div>
        )}

        {searchState !== 'loading' && visibleResults.length > 0 && (
          <div className="results-grid">
            {visibleResults.map((item, index) => {
              const favorite = isFavorite(item)

              return (
                <article
                  key={`${item.source}:${item.url}:${item.name}`}
                  className={`media-card${activeMedia?.url === item.url ? ' active' : ''}`}
                  style={{ '--card-i': index } as React.CSSProperties}
                  onClick={() => void handleOpenMedia(item)}
                  title={cleanTitle(item.name)}
                >
                  <div className="cover">
                    <button
                      className={`btn-fav${favorite ? ' active' : ''}`}
                      type="button"
                      onClick={event => {
                        event.stopPropagation()
                        void handleToggleFavorite(item)
                      }}
                      title={favorite ? 'Remover dos favoritos' : 'Adicionar aos favoritos'}
                      aria-label={favorite ? 'Remover dos favoritos' : 'Adicionar aos favoritos'}
                    >
                      <IconHeart active={favorite} />
                    </button>
                    {item.imageUrl
                      ? <img src={item.imageUrl} alt={item.name} loading="lazy" />
                      : <div className="cover-fallback"><IconFilm /></div>}
                    <div className="cover-overlay">
                      <span className="chip chip-type">{typeLabels[item.mediaType as TypeFilter] ?? item.mediaType}</span>
                      <span className="chip chip-source">{item.source}</span>
                      {item.watchHasPortuguese && <span className="chip chip-lang chip-lang-pt">PT-BR</span>}
                      {item.watchHasDub && <span className="chip chip-lang chip-lang-pt">DUB</span>}
                      {item.watchHasSub && <span className="chip chip-lang chip-lang-en">LEG</span>}
                      {!item.watchHasPortuguese && !item.watchHasDub && !item.watchHasSub && (() => { const t = extractLangTag(item.name); return t ? <span className={`chip chip-lang chip-lang-${t.variant}`}>{t.label}</span> : null })()}
                    </div>
                    {(item.score || item.description || item.genres?.length) ? (
                      <div className="card-meta-overlay">
                        {item.score ? <span className="cmo-score">★ {item.score.toFixed(1)}</span> : null}
                        {item.genres?.length ? (
                          <div className="cmo-genres">
                            {item.genres.slice(0, 3).map(g => (
                              <span key={g} className="catalog-genre-pill">{g}</span>
                            ))}
                          </div>
                        ) : null}
                        {item.description ? (
                          <p className="cmo-desc">
                            {item.description.length > 110 ? item.description.slice(0, 110) + '…' : item.description}
                          </p>
                        ) : null}
                      </div>
                    ) : null}
                  </div>
                  <div className="card-info">
                    <p className="card-title">{formatCardTitle(item)}</p>
                    <p className={`card-language${watchHasPortugueseSignal(item) ? ' card-language-pt' : ' card-language-en'}`}>
                      {formatLanguageSummary(item)}
                    </p>
                    {item.year && <p className="card-year">{item.year}</p>}
                    {(item.watchSource || item.downloadSource || item.dubSource || item.subSource || item.availableSources?.length) && (
                      <div className="card-recommendations">
                        {item.availableSources && item.availableSources.length > 1 && (
                          <span className="card-reco-chip">Fontes {item.availableSources.length}</span>
                        )}
                        {item.watchSource && <span className="card-reco-chip">Ver {formatSourceLabel(item.watchSource)}</span>}
                        {item.downloadSource && <span className="card-reco-chip">DL {formatSourceLabel(item.downloadSource)}</span>}
                        {item.dubSource && <span className="card-reco-chip">Dub {formatSourceLabel(item.dubSource)}</span>}
                        {item.subSource && <span className="card-reco-chip">Sub {formatSourceLabel(item.subSource)}</span>}
                      </div>
                    )}
                  </div>
                </article>
              )
            })}
          </div>
        )}
      </main>

      {/* Player */}
      <aside className="player-panel">
        <div
          ref={playerWrapRef}
          className="player-video-wrap"
          onDoubleClick={() => void handleToggleFullscreen()}
        >
          <video
            ref={videoRef}
            className="player-video"
            controls
            playsInline
            preload="metadata"
            poster={activeMedia?.imageUrl ?? ''}
            onTimeUpdate={handleVideoTimeUpdate}
            onPause={() => { void persistCurrentProgress(false) }}
            onEnded={() => { void handleVideoEnded() }}
          />
        </div>

        <div className="player-body">
          <div className="player-heading">
            <div>
              <p className="player-title">{activeMedia ? cleanTitle(activeMedia.name) : 'Player'}</p>
              <p className={`player-subtitle${playerState === 'error' ? ' player-subtitle-error' : ''}`}>{playerMessage}</p>
              {playerState === 'error' && activeMedia && (
                <button className="btn-retry" type="button" onClick={() => void startPlayback(episodeIndex)}>
                  Tentar novamente
                </button>
              )}
            </div>

            {activeMedia && (
              <div className="player-mini-actions">
                <button
                  className={`btn-icon${isActiveFavorite ? ' active' : ''}`}
                  type="button"
                  onClick={() => void handleToggleFavorite(activeMedia)}
                  title={isActiveFavorite ? 'Remover dos favoritos' : 'Adicionar aos favoritos'}
                >
                  <IconHeart active={isActiveFavorite} />
                </button>
                <button className="btn-icon" type="button" onClick={() => void handleToggleFullscreen()} title="Tela cheia">
                  <IconExpand active={isFullscreen} />
                </button>
              </div>
            )}
          </div>

          {activeProgress && activeProgress.duration > 0 && (
            <div className="player-progress-card">
              <div className="player-progress-meta">
                <span>Ep {activeProgress.episodeNumber}</span>
                <span>{formatPercent(activeProgress.progressPercent)} assistido</span>
              </div>
              <div className="ep-progress-wrap player-progress-wrap">
                <div className="ep-progress-bar" style={{ width: `${Math.max(activeProgress.progressPercent, 4)}%` }} />
              </div>
            </div>
          )}

          <div className="player-controls">
            <div className="control-group">
              <span className="control-label">
                Episodio{activeProgress ? ` - ${formatEpisodeCounter(activeProgress)}` : ''}
              </span>
              <div className="episode-list-wrap">
                {episodes.length === 0 ? (
                  <p className="episode-list-empty">—</p>
                ) : (
                  <div className="episode-list" ref={episodeListRef}>
                    {episodes.map((episode, index) => {
                      const epNum = getEpisodeNumber(episode, index)
                      const watched = activeProgress ? epNum < activeProgress.episodeNumber : false
                      const lastWatched = activeProgress ? epNum === activeProgress.episodeNumber : false
                      const isCurrent = index === episodeIndex
                      return (
                        <button
                          key={`${episode.url}-${index}`}
                          className={`ep-item${isCurrent ? ' current' : ''}${watched ? ' watched' : ''}${lastWatched ? ' last-watched' : ''}`}
                          onClick={() => setEpisodeIndex(index)}
                          title={episode.title || `Epis\u00f3dio ${epNum}`}
                        >
                          <span className="ep-item-num">Ep {epNum}</span>
                          {episode.title && <span className="ep-item-title">{episode.title}</span>}
                          {(watched || lastWatched) && <span className="ep-check">{'\u2713'}</span>}
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
            </div>

            <div className="control-row">
              <div className="control-group">
                <span className="control-label">
                  {'\u00c1udio'}
                  {activeMedia && dubSources.has(activeMedia.source.toLowerCase()) && (
                    <span className="control-label-hint" title="Fonte j? fornece apenas dublado">{'\u00b7'} fixo</span>
                  )}
                  {activeMedia && activeMedia.source.toLowerCase() === 'allanime' && (
                    <span className="control-label-hint control-label-hint-active" title="Muda entre sub e dub no AllAnime">{'\u00b7'} AllAnime</span>
                  )}
                </span>
                <select
                  className="control-select"
                  value={mode}
                  onChange={event => setMode(event.target.value as 'sub' | 'dub')}
                  disabled={activeMedia ? dubSources.has(activeMedia.source.toLowerCase()) : false}
                  title={activeMedia && dubSources.has(activeMedia.source.toLowerCase())
                    ? 'Esta fonte s? tem vers?o dublada'
                    : 'Selecione sub ou dub (funciona principalmente no AllAnime)'}
                >
                  <option value="sub">Legendado</option>
                  <option value="dub">Dublado</option>
                </select>
              </div>

              <div className="control-group">
                <span className="control-label">Qualidade</span>
                <select className="control-select" value={quality} onChange={event => setQuality(event.target.value)}>
                  {qualityOptions.map(option => <option key={option} value={option}>{option}</option>)}
                </select>
              </div>
            </div>

            <div className="play-actions">
              <button
                className={`btn btn-accent${playerState === 'loading' ? ' btn-loading' : ''}`}
                onClick={() => void handlePlay()}
                disabled={!canPlay}
              >
                {playerState !== 'loading' && <><IconPlay /> Assistir agora</>}
              </button>
              <button
                className="btn btn-secondary"
                onClick={() => canNext && setEpisodeIndex(index => index + 1)}
                disabled={!canNext}
              >
                <IconSkip /> Pr{'\u00f3'}ximo epis{'\u00f3'}dio
              </button>
            </div>

            <div className="utility-actions">
              <button className={`btn btn-secondary${isDownloading ? ' btn-loading' : ''}`} onClick={() => void handleDownload()} disabled={!canDownload}>
                {!isDownloading && <><IconDownload /> Baixar episodio</>}
              </button>
              <button className="btn btn-secondary" onClick={() => void handleToggleFullscreen()}>
                <IconExpand active={isFullscreen} /> Tela cheia
              </button>
            </div>

            <p className="keys-hint">&lt;- P | prox N -&gt; | F para tela cheia</p>
          </div>

          {/* Anime relacionados */}
          {activeMedia && (relatedLoading || related.length > 0) && (
            <div className="related-section">
              <p className="related-label">
                Também dessa série
                {relatedLoading && <span className="related-loading-dot" />}
              </p>
              {relatedLoading && related.length === 0 ? (
                <div className="related-skeletons">
                  {[0,1,2].map(i => <div key={i} className="related-card-skeleton" />)}
                </div>
              ) : (
                <div className="related-list">
                  {related.map(rel => (
                    <button
                      key={rel.malId}
                      className="related-card"
                      onClick={() => { setQuery(rel.name); setView('catalog') }}
                      title={`${rel.relation}: ${rel.name}`}
                    >
                      <div className="related-cover">
                        {rel.imageUrl
                          ? <img src={rel.imageUrl} alt={rel.name} loading="lazy" />
                          : <div className="related-cover-fallback"><IconFilm /></div>}
                      </div>
                      <div className="related-info">
                        <span className="related-relation">{rel.relation}</span>
                        <span className="related-name">{rel.name}</span>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </aside>

      {/* Toast stack */}
      {toasts.length > 0 && (
        <div className="toast-stack">
          {toasts.map(t => (
            <div key={t.id} className={`toast toast-${t.type}`}>
              {t.type === 'success' && <span className="toast-icon">✓</span>}
              {t.type === 'error' && <span className="toast-icon">✕</span>}
              {t.type === 'info' && <span className="toast-icon">i</span>}
              {t.message}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
















































