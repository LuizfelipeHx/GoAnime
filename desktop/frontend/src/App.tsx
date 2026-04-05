import { useEffect, useMemo, useRef, useState } from 'react'
import Hls from 'hls.js'
import { Catalog } from './Catalog'
import Settings from './Settings'
import Stats from './Stats'
import AnimeNotes from './AnimeNotes'
import {
  BrowserOpenURL,
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
  getCatalogByGenre,
  getGenres,
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
  getBotStatus,
  getNyaaReleases,
  clearNewReleases,
  getAIRecommendations,
  refreshRecommendations,
  getCuratedReleases,
  refreshCuratedReleases,
  getSettings,
  saveSettings,
  getSeasonCalendar,
  getWatchedEpisodes,
  setEpisodeWatched,
  getSkipTimes,
  getCustomLists,
  getListNames,
  addToList,
  removeFromList,
  moveToList,
  getWatchStats,
  getPlayQueue,
  addToQueue,
  removeFromQueue,
  clearQueue,
  getAnimeNote,
  saveAnimeNote,
  getAniListSyncStatus,
  startAniListAuth,
  disconnectAniList,
  syncToAniList,
  getAnimeDetails,
  type AppSettings,
  type AnimeLibraryEntry,
  type BotStatus,
  type CalendarDay,
  type NyaaRelease,
  type AIRecommendation,
  type CuratedRelease,
  type CatalogSection,
  type EpisodeResult,
  type FavoriteEntry,
  type HistoryEntry,
  type MediaRequest,
  type MediaResult,
  type RelatedAnime,
  type SearchCoversEvent,
  type SkipTimesResult,
  type WatchProgressEntry,
  type ListEntry,
  type WatchStats,
  type QueueEntry,
  type AnimeNote,
  type AniListSyncStatus,
} from './lib/backend'

type SourceFilter = 'all' | 'allanime' | 'animefire' | 'flixhq' | 'animesonlinecc' | 'anroll' | 'bakashi'
type TypeFilter = 'all' | 'anime' | 'movie' | 'tv'
type LangFilter = 'all' | 'pt' | 'en'
type LoadState = 'idle' | 'loading' | 'ready' | 'error'
type ViewMode = 'catalog' | 'movies' | 'favorites' | 'watching' | 'history' | 'bots' | 'settings' | 'calendar' | 'stats' | 'lists' | 'queue'
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

function normalizeSearchText(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
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
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  const [mode, setMode] = useState<'sub' | 'dub'>(() => readPref('pref-mode', 'sub') as 'sub' | 'dub')
  const [quality, setQuality] = useState(() => readPref('pref-quality', 'best'))

  const [playerMessage, setPlayerMessage] = useState('Selecione um titulo para começar.')
  const [playerState, setPlayerState] = useState<LoadState>('idle')
  const [iframeUrl, setIframeUrl] = useState<string | null>(null)
  const [isMaximised, setIsMaximised] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [isDownloading, setIsDownloading] = useState(false)
  const [view, setView] = useState<ViewMode>('catalog')
  const [movieSourceMode, setMovieSourceMode] = useState<MovieSourceMode>('ptbr')
  const [toasts, setToasts] = useState<Toast[]>([])
  const [related, setRelated] = useState<RelatedAnime[]>([])
  const [relatedLoading, setRelatedLoading] = useState(false)
  const [genres, setGenres] = useState<string[]>([])
  const [genreOpen, setGenreOpen] = useState(false)
  const [selectedGenre, setSelectedGenre] = useState('')
  const [genreSections, setGenreSections] = useState<CatalogSection[]>([])
  const [genreLoading, setGenreLoading] = useState(false)
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [botStatus, setBotStatus] = useState<BotStatus>({ aiOnline: false, releasesCount: 0, newReleases: 0, recsAvailable: false, recsCount: 0, curatedCount: 0 })
  const [nyaaReleases, setNyaaReleases] = useState<NyaaRelease[]>([])
  const [aiRecs, setAiRecs] = useState<AIRecommendation[]>([])
  const [curatedReleases, setCuratedReleases] = useState<CuratedRelease[]>([])
  const [botsLoading, setBotsLoading] = useState(false)

  // Episode filter
  const [episodeFilter, setEpisodeFilter] = useState('')

  // Picture-in-Picture
  const [isPip, setIsPip] = useState(false)

  // Playback speed
  const [playbackSpeed, setPlaybackSpeed] = useState(1.0)

  // Settings
  const [settings, setSettings] = useState<AppSettings | null>(null)

  // Season calendar
  const [seasonCalendar, setSeasonCalendar] = useState<CalendarDay[]>([])

  // Watched episode marks
  const [watchedMarks, setWatchedMarks] = useState<Set<number>>(new Set())

  // Skip intro/ending
  const [skipTimes, setSkipTimes] = useState<SkipTimesResult | null>(null)
  const [showSkipIntro, setShowSkipIntro] = useState(false)
  const [showSkipEnding, setShowSkipEnding] = useState(false)

  // Custom lists
  const [customLists, setCustomLists] = useState<Record<string, ListEntry[]>>({})
  const [listNames, setListNames] = useState<string[]>([])
  const [selectedList, setSelectedList] = useState('Favoritos')

  // Stats
  const [watchStats, setWatchStats] = useState<WatchStats | null>(null)

  // Play queue
  const [playQueue, setPlayQueue] = useState<QueueEntry[]>([])

  // Notes
  const [showNotes, setShowNotes] = useState(false)
  const [activeNote, setActiveNote] = useState<AnimeNote | null>(null)

  // AniList sync
  const [anilistStatus, setAnilistStatus] = useState<AniListSyncStatus | null>(null)

  // List action dropdown (for moving between lists)
  const [listDropdownFor, setListDropdownFor] = useState<string | null>(null)

  // Anime detail view
  const [animeDetail, setAnimeDetail] = useState<AnimeLibraryEntry | null>(null)
  const [animeDetailLoading, setAnimeDetailLoading] = useState(false)
  const animeDetailMediaRef = useRef<MediaResult | null>(null)

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

  const filteredEpisodes = useMemo(() => {
    if (!episodeFilter.trim()) return episodes.map((ep, i) => ({ ep, originalIndex: i }))
    const q = episodeFilter.toLowerCase()
    return episodes.map((ep, i) => ({ ep, originalIndex: i }))
      .filter(({ ep }) => ep.number.includes(q) || ep.title?.toLowerCase().includes(q) || String(ep.num).includes(q))
  }, [episodes, episodeFilter])

  const movieFavorites = useMemo(() => favorites.filter(item => item.mediaType === 'movie'), [favorites])
  const movieProgress = useMemo(() => progress.filter(item => item.mediaType === 'movie'), [progress])
  const canNext = episodes.length > 0 && episodeIndex < episodes.length - 1
  const currentEpisode = episodes[episodeIndex] ?? null
  const activeProgress = useMemo(() => findProgressEntry(activeMedia, progress), [activeMedia, progress])
  const favoriteKeys = useMemo(() => new Set(favorites.map(item => favoriteKey(item.source, item.url))), [favorites])
  const hasSidebarContent = favorites.length > 0 || progress.length > 0 || history.length > 0
  const isActiveFavorite = activeMedia ? favoriteKeys.has(favoriteKey(activeMedia.source, activeMedia.url)) : false
  const searchPlaceholder = view === 'movies' ? 'Buscar filme...' : 'Buscar anime ou série...'

  const suggestions = useMemo(() => {
    if (!query || query.length < 1 || searchState === 'loading') return []
    const q = query.toLowerCase()
    const fromHistory = history
      .filter(h => h.name.toLowerCase().includes(q))
      .map(h => ({ text: h.name, type: 'history' as const }))
    const fromResults = results
      .filter(r => cleanTitle(r.name).toLowerCase().includes(q))
      .slice(0, 3)
      .map(r => ({ text: cleanTitle(r.name), type: 'result' as const }))
    const seen = new Set<string>()
    const merged: { text: string; type: 'history' | 'result' }[] = []
    for (const item of [...fromHistory, ...fromResults]) {
      const key = item.text.toLowerCase()
      if (!seen.has(key) && key !== q) {
        seen.add(key)
        merged.push(item)
      }
      if (merged.length >= 6) break
    }
    return merged
  }, [query, history, results, searchState])

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
    getGenres().then(setGenres).catch(() => {})

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

    getCustomLists().then(setCustomLists).catch(() => {})
    getListNames().then(setListNames).catch(() => {})
    getAniListSyncStatus().then(setAnilistStatus).catch(() => {})
  }, [])

  useEffect(() => {
    if (!sourceOptions.some(item => item.value === source)) {
      setSource(sourceOptions[0]?.value ?? 'all')
    }
  }, [sourceOptions, source])

  useEffect(() => {
    if (trimmedQuery.length < 2) {
      setResults([])
      setSearchState('idle')
      setSearchMessage('')
      return
    }

    // Close anime detail when starting a new search
    setAnimeDetail(null)
    animeDetailMediaRef.current = null

    if (view === 'favorites' || view === 'watching' || view === 'history' || view === 'bots' || view === 'settings' || view === 'calendar' || view === 'stats' || view === 'lists' || view === 'queue') {
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
        setSearchMessage(data.length ? `${data.length} resultado(s) encontrado(s)` : 'Nenhum resultado encontrado.')
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

  // Listen for streaming partial search results
  useEffect(() => {
    const unsubscribe = EventsOn('search:partial', (partialResults: MediaResult[]) => {
      if (!partialResults?.length || searchState !== 'loading') return
      setResults(current => {
        const existing = new Set(current.map(r => `${r.source}|${r.url}`))
        const newItems = partialResults.filter(r => !existing.has(`${r.source}|${r.url}`))
        if (newItems.length === 0) return current
        return [...current, ...newItems]
      })
      setSearchState('loading') // Keep loading state (more may come)
      setSearchMessage(prev => {
        const match = prev.match(/\d+/)
        const prevCount = match ? parseInt(match[0], 10) : 0
        return `${prevCount + partialResults.length} resultado(s) e buscando...`
      })
    })
    return () => unsubscribe()
  }, [searchState])

  useEffect(() => () => { hlsRef.current?.destroy() }, [])

  // Fetch bot status periodically
  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const status = await getBotStatus()
        setBotStatus(status)
      } catch {}
    }
    fetchStatus()
    const interval = setInterval(fetchStatus, 30000) // every 30s
    return () => clearInterval(interval)
  }, [])

  // Listen for new releases event
  useEffect(() => {
    const unsub = EventsOn('bot:newReleases', (data: { count: number }) => {
      setBotStatus(prev => ({ ...prev, newReleases: prev.newReleases + (data?.count || 0) }))
    })
    return () => unsub()
  }, [])

  // PiP event listeners
  useEffect(() => {
    const video = videoRef.current
    if (!video) return
    const onEnterPip = () => setIsPip(true)
    const onLeavePip = () => setIsPip(false)
    video.addEventListener('enterpictureinpicture', onEnterPip)
    video.addEventListener('leavepictureinpicture', onLeavePip)
    return () => {
      video.removeEventListener('enterpictureinpicture', onEnterPip)
      video.removeEventListener('leavepictureinpicture', onLeavePip)
    }
  }, [])

  // Load settings on mount
  useEffect(() => {
    getSettings().then(setSettings).catch(() => {})
  }, [])

  // Load calendar when view changes to 'calendar'
  useEffect(() => {
    if (view !== 'calendar') return
    getSeasonCalendar().then(setSeasonCalendar).catch(() => setSeasonCalendar([]))
  }, [view])

  // Load stats when view changes to stats
  useEffect(() => {
    if (view === 'stats') {
      getWatchStats().then(setWatchStats).catch(() => {})
    }
  }, [view])

  // Load queue when view changes to queue
  useEffect(() => {
    if (view === 'queue') {
      getPlayQueue().then(setPlayQueue).catch(() => {})
    }
  }, [view])

  // Load note when activeMedia changes
  useEffect(() => {
    if (activeMedia) {
      const title = cleanTitle(activeMedia.name)
      getAnimeNote(title).then(setActiveNote).catch(() => setActiveNote(null))
    }
  }, [activeMedia])

  // Load watched marks when activeMedia changes
  useEffect(() => {
    if (!activeMedia) { setWatchedMarks(new Set()); return }
    const key = activeMedia.groupKey || normalizeSearchText(cleanTitle(activeMedia.name))
    if (!key) return
    getWatchedEpisodes(key)
      .then(nums => setWatchedMarks(new Set(nums)))
      .catch(() => setWatchedMarks(new Set()))
  }, [activeMedia])

  // Load bot data when switching to bots view
  useEffect(() => {
    if (view !== 'bots') return
    const load = async () => {
      setBotsLoading(true)
      try {
        const [releases, recs, curated] = await Promise.all([
          getNyaaReleases(),
          getAIRecommendations(),
          getCuratedReleases(),
        ])
        setNyaaReleases(releases || [])
        setAiRecs(recs || [])
        setCuratedReleases(curated || [])
        await clearNewReleases()
        setBotStatus(prev => ({ ...prev, newReleases: 0 }))
      } catch {}
      setBotsLoading(false)
    }
    load()
  }, [view])

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const tag = (event.target as HTMLElement).tagName
      if (tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA') return

      // Space to toggle play/pause
      if (event.key === ' ' || event.code === 'Space') {
        event.preventDefault()
        const video = videoRef.current
        if (video) {
          if (video.paused) {
            video.play().catch(() => {})
          } else {
            video.pause()
          }
        }
      }

      // Arrow keys for seeking (when not in episode navigation)
      if (event.key === 'ArrowRight' && event.shiftKey) {
        event.preventDefault()
        const video = videoRef.current
        if (video) video.currentTime = Math.min(video.duration, video.currentTime + 10)
      }
      if (event.key === 'ArrowLeft' && event.shiftKey) {
        event.preventDefault()
        const video = videoRef.current
        if (video) video.currentTime = Math.max(0, video.currentTime - 10)
      }

      // M to toggle mute
      if (event.key.toLowerCase() === 'm') {
        const video = videoRef.current
        if (video) video.muted = !video.muted
      }

      if (((event.key === 'ArrowRight' && !event.shiftKey) || event.key.toLowerCase() === 'n') && canNext) {
        setEpisodeIndex(index => index + 1)
      }
      if (((event.key === 'ArrowLeft' && !event.shiftKey) || event.key.toLowerCase() === 'p') && episodeIndex > 0) {
        setEpisodeIndex(index => index - 1)
      }
      if (event.key.toLowerCase() === 'f') {
        void handleToggleFullscreen()
      }

      // PiP toggle
      if (event.key.toLowerCase() === 'i') {
        void handleTogglePip()
      }

      // Speed control: ] increase, [ decrease
      if (event.key === ']') {
        const idx = speedSteps.indexOf(playbackSpeed)
        if (idx < speedSteps.length - 1) {
          const next = speedSteps[idx + 1]
          setPlaybackSpeed(next)
          if (videoRef.current) videoRef.current.playbackRate = next
        }
      }
      if (event.key === '[') {
        const idx = speedSteps.indexOf(playbackSpeed)
        if (idx > 0) {
          const prev = speedSteps[idx - 1]
          setPlaybackSpeed(prev)
          if (videoRef.current) videoRef.current.playbackRate = prev
        }
      }
    }

    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [canNext, episodeIndex, isFullscreen, playbackSpeed])

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

    // Skip intro/ending button visibility
    if (skipTimes && skipTimes.found) {
      const t = video.currentTime
      setShowSkipIntro(t >= skipTimes.opStart && t < skipTimes.opEnd)
      setShowSkipEnding(t >= skipTimes.edStart && t < skipTimes.edEnd)
    }

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

  const handleAddToList = async (listName: string) => {
    if (!activeMedia) return
    const entry: ListEntry = {
      name: cleanTitle(activeMedia.name),
      url: activeMedia.url,
      imageUrl: activeMedia.imageUrl || '',
      source: activeMedia.source,
      listName,
    }
    await addToList(listName, entry)
    const updated = await getCustomLists()
    setCustomLists(updated)
    showToast(`Adicionado a "${listName}"`, 'success')
    setListDropdownFor(null)
  }

  const handleRemoveFromList = async (listName: string, name: string) => {
    await removeFromList(listName, name)
    const updated = await getCustomLists()
    setCustomLists(updated)
  }

  const handleMoveToList = async (fromList: string, toList: string, name: string) => {
    await moveToList(fromList, toList, name)
    const updated = await getCustomLists()
    setCustomLists(updated)
    showToast(`Movido para "${toList}"`, 'success')
  }

  const handleAddToQueue = async () => {
    if (!activeMedia || !currentEpisode) return
    const entry: QueueEntry = {
      mediaName: cleanTitle(activeMedia.name),
      url: activeMedia.url,
      source: activeMedia.source,
      mediaType: activeMedia.mediaType || 'anime',
      episodeUrl: currentEpisode.url,
      episodeNumber: currentEpisode.num ? String(currentEpisode.num) : currentEpisode.number,
      imageUrl: activeMedia.imageUrl || '',
    }
    await addToQueue(entry)
    const updated = await getPlayQueue()
    setPlayQueue(updated)
    showToast('Adicionado \u00e0 fila', 'success')
  }

  const handleSaveNote = async (note: AnimeNote) => {
    await saveAnimeNote(note)
    setActiveNote(note)
    setShowNotes(false)
    showToast('Nota salva!', 'success')
  }

  const handleAniListAuth = async () => {
    try {
      const url = await startAniListAuth()
      window.open(url, '_blank')
      showToast('Autorize no navegador e volte aqui', 'info')
      // Poll for status update
      setTimeout(async () => {
        const status = await getAniListSyncStatus()
        setAnilistStatus(status)
      }, 10000)
    } catch {
      showToast('Erro ao iniciar autentica\u00e7\u00e3o', 'error')
    }
  }

  const handleTogglePip = async () => {
    const video = videoRef.current
    if (!video) return
    try {
      if (document.pictureInPictureElement) {
        await document.exitPictureInPicture()
      } else if (document.pictureInPictureEnabled) {
        await video.requestPictureInPicture()
      }
    } catch {}
  }

  const speedSteps = [0.5, 0.75, 1, 1.25, 1.5, 2]

  const dubSources = new Set(['animefire', 'animesonlinecc'])

  const handleCardClick = async (item: MediaResult) => {
    if (item.anilistId && item.anilistId > 0) {
      setAnimeDetailLoading(true)
      setAnimeDetail(null)
      try {
        const detail = await getAnimeDetails(item.anilistId)
        setAnimeDetail(detail)
      } catch {
        // Fallback: build a minimal detail from the search result
        setAnimeDetail({
          anilistId: item.anilistId || 0,
          malId: item.malId || 0,
          title: cleanTitle(item.name),
          titleRomaji: item.canonicalTitle || '',
          titleEnglish: '',
          coverImage: item.imageUrl || '',
          bannerImage: '',
          genres: item.genres || [],
          description: item.description || '',
          totalEpisodes: item.totalEpisodes || 0,
          score: item.score || 0,
          status: '',
          format: '',
          year: item.year ? parseInt(item.year, 10) || 0 : 0,
          sources: (item.alternatives || []).map(a => ({
            source: a.source,
            url: a.url,
            name: a.name,
            mediaType: a.mediaType,
          })),
          lastUpdated: '',
        })
      } finally {
        setAnimeDetailLoading(false)
      }
      // Store the original media result for the "Assistir" action
      animeDetailMediaRef.current = item
    } else {
      // No anilistId - fallback to direct episode loading
      void handleOpenMedia(item)
    }
  }

  const handleDetailWatch = () => {
    const item = animeDetailMediaRef.current
    if (item) {
      setAnimeDetail(null)
      animeDetailMediaRef.current = null
      void handleOpenMedia(item)
    }
  }

  const handleDetailClose = () => {
    setAnimeDetail(null)
    animeDetailMediaRef.current = null
  }

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
    setEpisodeFilter('')
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
    setIframeUrl(null) // clear iframe if switching to video
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
      if (playbackSpeed !== 1) video.playbackRate = playbackSpeed
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

      // Iframe-only embeds (e.g. Blogger Video): render in iframe instead of <video>
      if (stream.contentType === 'iframe') {
        setIframeUrl(stream.streamUrl)
        setPlayerMessage('Reprodução via player externo.')
        setPlayerState('ready')
      } else {
        setIframeUrl(null)
        await loadVideo(
          stream.proxyUrl || stream.streamUrl,
          stream.contentType || 'video/*',
          stream.subtitles,
          resumeAt,
        )
      }
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

  const handleClosePlayer = () => {
    // Stop video/hls playback
    hlsRef.current?.destroy()
    hlsRef.current = null
    if (videoRef.current) {
      videoRef.current.pause()
      videoRef.current.removeAttribute('src')
      videoRef.current.load()
    }
    setIframeUrl(null)
    setActiveMedia(null)
    setEpisodes([])
    setEpisodeIndex(0)
    setPlayerState('idle')
    setPlayerMessage('Selecione um título para começar.')
    setRelated([])
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

  const genreLabels: Record<string, string> = {
    'Action': 'A\u00e7\u00e3o', 'Adventure': 'Aventura', 'Comedy': 'Com\u00e9dia',
    'Drama': 'Drama', 'Ecchi': 'Ecchi', 'Fantasy': 'Fantasia',
    'Horror': 'Horror', 'Mahou Shoujo': 'Mahou Shoujo', 'Mecha': 'Mecha',
    'Music': 'Musical', 'Mystery': 'Mist\u00e9rio', 'Psychological': 'Psicol\u00f3gico',
    'Romance': 'Romance', 'Sci-Fi': 'Fic\u00e7\u00e3o Cient\u00edfica',
    'Slice of Life': 'Slice of Life', 'Sports': 'Esporte',
    'Supernatural': 'Sobrenatural', 'Thriller': 'Suspense',
  }

  function handleGenreSelect(genre: string) {
    if (genre === selectedGenre) {
      setSelectedGenre('')
      setGenreSections([])
      return
    }
    setSelectedGenre(genre)
    setGenreLoading(true)
    setView('catalog')
    setQuery('')
    getCatalogByGenre(genre)
      .then(setGenreSections)
      .catch(() => setGenreSections([]))
      .finally(() => setGenreLoading(false))
  }

  const catalogDisplaySections = selectedGenre ? genreSections : catalog
  const catalogDisplayLoading = selectedGenre ? genreLoading : catalogLoading

  return (
    <div className={`app${activeMedia === null ? ' app--no-player' : ''}${sidebarCollapsed ? ' app--sidebar-collapsed' : ''}`}>
      {/* Sidebar */}
      <aside className={`sidebar${sidebarCollapsed ? ' sidebar--collapsed' : ''}`}>
        <div className="logo">
          <div className="logo-icon">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="white"><polygon points="5 3 19 12 5 21 5 3" /></svg>
          </div>
          <div className="logo-text">
            <span className="logo-go">Go</span><span className="logo-anime">Anime</span>
          </div>
          <span className="logo-badge">Desktop</span>
          <button className="sidebar-toggle-btn" onClick={() => setSidebarCollapsed(!sidebarCollapsed)} title={sidebarCollapsed ? 'Expandir menu' : 'Recolher menu'} type="button">
            {sidebarCollapsed ? '\u25B6' : '\u25C0'}
          </button>
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
            <button
              className={`sidebar-nav-btn${view === 'history' ? ' active' : ''}`}
              onClick={() => { setView('history'); setQuery('') }}
            >
              <IconClock />
              <span>Hist{'\u00F3'}rico</span>
              {history.length > 0 && <span className="sidebar-count">{history.length}</span>}
            </button>
            <button className={`sidebar-nav-btn${view === 'bots' ? ' active' : ''}`} onClick={() => { setView('bots'); setQuery('') }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/></svg>
              <span>Bots IA</span>
              {botStatus.newReleases > 0 && <span className="sidebar-count bot-badge">{botStatus.newReleases}</span>}
            </button>
            <button className={`sidebar-nav-btn${view === 'stats' ? ' active' : ''}`} onClick={() => { setView('stats'); setQuery('') }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 20V10M12 20V4M6 20v-6"/></svg>
              <span>Estat{'\u00ed'}sticas</span>
            </button>
            <button className={`sidebar-nav-btn${view === 'lists' ? ' active' : ''}`} onClick={() => { setView('lists'); setQuery('') }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
              <span>Listas</span>
              {Object.values(customLists).reduce((a, b) => a + b.length, 0) > 0 && <span className="sidebar-count">{Object.values(customLists).reduce((a, b) => a + b.length, 0)}</span>}
            </button>
            <button className={`sidebar-nav-btn${view === 'queue' ? ' active' : ''}`} onClick={() => { setView('queue'); setQuery('') }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="16 3 21 3 21 8"/><line x1="4" y1="20" x2="21" y2="3"/><polyline points="21 16 21 21 16 21"/><line x1="15" y1="15" x2="21" y2="21"/><line x1="4" y1="4" x2="9" y2="9"/></svg>
              <span>Fila</span>
              {playQueue.length > 0 && <span className="sidebar-count">{playQueue.length}</span>}
            </button>
          </div>

          {genres.length > 0 && (
            <>
              <button className="sidebar-nav-btn" onClick={() => setGenreOpen(!genreOpen)}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
                <span>G{'\u00ea'}nero</span>
                <span className={`sidebar-chevron${genreOpen ? ' open' : ''}`}>{'\u25b8'}</span>
              </button>
              {genreOpen && (
                <div className="sidebar-genre-list">
                  {genres.map(g => (
                    <button
                      key={g}
                      className={`sidebar-genre-item${selectedGenre === g ? ' active' : ''}`}
                      onClick={() => handleGenreSelect(g)}
                    >
                      {genreLabels[g] ?? g}
                    </button>
                  ))}
                </div>
              )}
            </>
          )}

          <button className={`sidebar-nav-btn${view === 'calendar' ? ' active' : ''}`} onClick={() => { setView('calendar'); setQuery('') }}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
            <span>Calend{'\u00e1'}rio</span>
          </button>

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
        <div className="sidebar-footer">
          <button className={`sidebar-nav-btn${view === 'settings' ? ' active' : ''}`} onClick={() => { setView('settings'); setQuery('') }}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
            <span>Configura{'\u00e7\u00f5'}es</span>
          </button>
          <div className={`ai-status ${botStatus.aiOnline ? 'ai-online' : 'ai-offline'}`}>
            <span className="ai-status-dot" />
            <span className="ai-status-text">{botStatus.aiOnline ? `IA: ${botStatus.aiModel || 'Online'}` : 'IA Offline'}</span>
          </div>
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
              onChange={event => { setQuery(event.target.value); setShowSuggestions(true) }}
              onFocus={() => setShowSuggestions(true)}
              onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
              placeholder={searchPlaceholder}
              autoFocus
            />
            {searchState === 'loading' && (
              <span className="search-spinner" />
            )}
            {query && (
              <button className="search-clear" type="button" onClick={() => setQuery('')} title="Limpar busca" aria-label="Limpar busca">
                <IconClear />
              </button>
            )}
            {showSuggestions && suggestions.length > 0 && (
              <div className="search-autocomplete">
                {suggestions.map((s, i) => (
                  <button
                    key={i}
                    className={`autocomplete-item autocomplete-item--${s.type}`}
                    onMouseDown={e => {
                      e.preventDefault()
                      setQuery(s.text)
                      setShowSuggestions(false)
                    }}
                  >
                    {s.type === 'history' ? <IconClock /> : <IconSearch />}
                    <span>{s.text}</span>
                  </button>
                ))}
              </div>
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

          {view === 'movies' && (
            <div className="source-select">
              <select value={source} onChange={event => setSource(event.target.value as SourceFilter)}>
                {sourceOptions.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}
              </select>
            </div>
          )}
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
          <div className="search-list">
            <div className="search-list-loading">Buscando...</div>
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

        {view === 'history' && searchState === 'idle' && trimmedQuery.length < 2 && (
          <div className="history-view">
            <div className="page-header">
              <h2>Hist{'\u00F3'}rico</h2>
            </div>
            {history.length === 0 && progress.length === 0 ? (
              <div className="empty-state">
                <div className="empty-icon">{'\u23F2'}</div>
                <h3>Nenhuma busca recente</h3>
                <p>Suas buscas e {'epis\u00F3dios'} assistidos aparecer{'\u00E3o'} aqui.</p>
              </div>
            ) : (
              <>
                {history.length > 0 && (
                  <>
                    <h3 className="section-heading">Hist{'\u00F3'}rico de Busca</h3>
                    <div className="history-grid">
                      {history.map(entry => (
                        <button key={entry.name} className="history-card" onClick={() => { setQuery(entry.name); setView('catalog') }}>
                          <IconClock />
                          <span className="history-card-text">{entry.name}</span>
                        </button>
                      ))}
                    </div>
                  </>
                )}
                {progress.length > 0 && (
                  <>
                    <h3 className="section-heading" style={{ marginTop: 24 }}>{'\u00DA'}ltimos Assistidos</h3>
                    <div className="watching-grid">
                      {progress.slice(0, 12).map(item => (
                        <button
                          key={item.allanimeId}
                          className="watching-card"
                          onClick={() => { setQuery(item.title); setView('catalog') }}
                        >
                          <div className="watching-card-header">
                            <span className="watching-card-title">{item.title}</span>
                            <span className="watching-ep-badge">Ep {item.episodeNumber}</span>
                          </div>
                          <div className="watching-card-meta">
                            <span>{formatPercent(item.progressPercent)} assistido</span>
                          </div>
                          {item.progressPercent > 0 && (
                            <div className="ep-progress-wrap watching-progress">
                              <div className="ep-progress-bar" style={{ width: `${Math.min(100, item.progressPercent)}%` }} />
                            </div>
                          )}
                        </button>
                      ))}
                    </div>
                  </>
                )}
              </>
            )}
          </div>
        )}

        {view === 'bots' && searchState === 'idle' && trimmedQuery.length < 2 && (
          <div className="bots-view">
            <div className="page-header" style={{ justifyContent: 'space-between', marginBottom: 16 }}>
              <h2>Bots IA</h2>
              <div className="bots-status-bar">
                <span className={`ai-indicator ${botStatus.aiOnline ? 'online' : 'offline'}`}>
                  {botStatus.aiOnline ? `\u2713 IA Online (${botStatus.aiModel || 'local'})` : '\u2717 IA Offline \u2014 Inicie o LM Studio'}
                </span>
              </div>
            </div>

            {/* Bot 1: Releases PT-BR */}
            <section className="bot-section">
              <h3 className="section-heading">{'\uD83D\uDD14'} Lan{'\u00E7'}amentos PT-BR (Nyaa)</h3>
              {nyaaReleases.length === 0 ? (
                <p className="empty-state-text">Nenhum lan{'\u00E7'}amento encontrado ainda. Verificando a cada 30 min...</p>
              ) : (
                <div className="releases-list">
                  {nyaaReleases.slice(0, 20).map((r, i) => (
                    <button key={i} className={`release-card ${r.isNew ? 'release-new' : ''}`} onClick={() => BrowserOpenURL(r.link)}>
                      <div className="release-info">
                        <span className="release-title">{r.title}</span>
                        <div className="release-meta">
                          <span>{r.size}</span>
                          <span>{'\u2191'} {r.seeders}</span>
                          <span>{r.date ? new Date(r.date).toLocaleDateString('pt-BR') : ''}</span>
                        </div>
                      </div>
                      {r.isNew && <span className="release-badge">NOVO</span>}
                    </button>
                  ))}
                </div>
              )}
            </section>

            {/* Bot 2: Recommendations */}
            <section className="bot-section">
              <div className="section-header-row">
                <h3 className="section-heading">{'\u2728'} Recomendados para Voc{'\u00EA'}</h3>
                {botStatus.aiOnline && (
                  <button className="btn-refresh" onClick={async () => {
                    setBotsLoading(true)
                    const recs = await refreshRecommendations()
                    setAiRecs(recs || [])
                    setBotsLoading(false)
                  }}>
                    {botsLoading ? 'Gerando...' : 'Atualizar'}
                  </button>
                )}
              </div>
              {!botStatus.aiOnline ? (
                <p className="empty-state-text">Inicie o LM Studio para receber recomenda{'\u00E7\u00F5'}es personalizadas.</p>
              ) : aiRecs.length === 0 ? (
                <p className="empty-state-text">{botsLoading ? 'Gerando recomenda\u00E7\u00F5es...' : 'Assista mais animes para receber recomenda\u00E7\u00F5es.'}</p>
              ) : (
                <div className="recs-grid">
                  {aiRecs.map((rec, i) => (
                    <div key={i} className="rec-card" onClick={() => { setQuery(rec.title); setView('catalog') }}>
                      <div className="rec-rank">#{i + 1}</div>
                      <div className="rec-info">
                        <span className="rec-title">{rec.title}</span>
                        <span className="rec-reason">{rec.reason}</span>
                        {rec.genres && <span className="rec-genres">{rec.genres}</span>}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </section>

            {/* Bot 3: Curated */}
            <section className="bot-section">
              <div className="section-header-row">
                <h3 className="section-heading">{'\uD83C\uDFAF'} Curadoria PT-BR</h3>
                {botStatus.aiOnline && (
                  <button className="btn-refresh" onClick={async () => {
                    setBotsLoading(true)
                    const c = await refreshCuratedReleases()
                    setCuratedReleases(c || [])
                    setBotsLoading(false)
                  }}>
                    {botsLoading ? 'Avaliando...' : 'Atualizar'}
                  </button>
                )}
              </div>
              {!botStatus.aiOnline ? (
                <p className="empty-state-text">Inicie o LM Studio para curadoria inteligente.</p>
              ) : curatedReleases.length === 0 ? (
                <p className="empty-state-text">{botsLoading ? 'Avaliando lan\u00E7amentos...' : 'Nenhum lan\u00E7amento avaliado.'}</p>
              ) : (
                <div className="releases-list">
                  {curatedReleases.map((c, i) => (
                    <button key={i} className={`release-card curated-${c.quality.toLowerCase()}`} onClick={() => BrowserOpenURL(c.release.link)}>
                      <div className="release-info">
                        <span className="release-title">{c.release.title}</span>
                        <div className="release-meta">
                          <span className={`quality-badge quality-${c.quality.toLowerCase()}`}>{c.quality}</span>
                          <span>{c.summary}</span>
                          <span>{c.release.size}</span>
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </section>
          </div>
        )}

        {view === 'settings' && (
          <>
            {settings && (
              <Settings settings={settings} onSave={async (s) => { await saveSettings(s); setSettings(s); showToast('Configura\u00e7\u00f5es salvas!', 'success') }} />
            )}
            <div className="sync-section" style={{ maxWidth: '600px', margin: '0 auto', padding: '0 24px 24px' }}>
              <h3 style={{ color: 'var(--text)', marginBottom: '12px' }}>Sincroniza\u00e7\u00e3o AniList</h3>
              {anilistStatus?.connected ? (
                <div>
                  {anilistStatus.profile && (
                    <div className="sync-profile">
                      {anilistStatus.profile.avatar && <img src={anilistStatus.profile.avatar} alt="" style={{ width: 40, height: 40, borderRadius: '50%' }} />}
                      <span style={{ color: 'var(--text)' }}>{anilistStatus.profile.name}</span>
                    </div>
                  )}
                  <p className="sync-status" style={{ color: '#4ade80' }}>Conectado</p>
                  {anilistStatus.lastSync && <p style={{ color: 'var(--text-muted)', fontSize: '12px' }}>{'\u00DA'}ltimo sync: {anilistStatus.lastSync}</p>}
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
                    <button className="sync-btn" onClick={async () => { await syncToAniList(); showToast('Sincronizado!', 'success'); const s = await getAniListSyncStatus(); setAnilistStatus(s) }}>Sincronizar agora</button>
                    <button className="sync-btn" style={{ background: '#dc2626' }} onClick={async () => { await disconnectAniList(); setAnilistStatus({ connected: false, tokenStored: false }) }}>Desconectar</button>
                  </div>
                </div>
              ) : (
                <div>
                  <p className="sync-status" style={{ color: 'var(--text-muted)' }}>N\u00e3o conectado</p>
                  <button className="sync-btn" onClick={handleAniListAuth} style={{ marginTop: '8px' }}>Conectar AniList</button>
                </div>
              )}
            </div>
          </>
        )}

        {view === 'calendar' && (
          <div className="calendar-view">
            <h2>Calend{'\u00e1'}rio da Temporada</h2>
            {seasonCalendar.map(day => (
              <div key={day.day} className="calendar-day">
                <h3>{day.day}</h3>
                <div className="calendar-grid">
                  {day.entries.map((entry, i) => (
                    <div key={i} className="calendar-card" onClick={() => { setQuery(entry.title) }}>
                      <img src={entry.imageUrl} alt={entry.title} />
                      <div className="calendar-card-info">
                        <p className="calendar-card-title">{entry.title}</p>
                        <p className="calendar-card-ep">Ep {entry.episode}{entry.totalEpisodes ? ` / ${entry.totalEpisodes}` : ''}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}

        {view === 'stats' && watchStats && (
          <Stats stats={watchStats} />
        )}

        {view === 'lists' && (
          <div className="custom-lists-view" style={{ padding: '24px' }}>
            <h2 style={{ color: 'var(--text)', marginBottom: '16px' }}>Minhas Listas</h2>
            <div className="list-selector">
              {listNames.map(name => (
                <button key={name} className={`list-tab${selectedList === name ? ' active' : ''}`} onClick={() => setSelectedList(name)}>
                  {name} ({(customLists[name] || []).length})
                </button>
              ))}
            </div>
            <div className="catalog-grid" style={{ marginTop: '16px' }}>
              {(customLists[selectedList] || []).map((entry, i) => (
                <div key={i} className="catalog-card" onClick={() => { setQuery(entry.name) }}>
                  {entry.imageUrl && <img className="catalog-card-img" src={entry.imageUrl} alt={entry.name} loading="lazy" />}
                  <div className="catalog-card-body">
                    <p className="catalog-card-title">{entry.name}</p>
                    <div style={{ display: 'flex', gap: '4px', marginTop: '4px' }}>
                      <button className="btn-small" onClick={(e) => { e.stopPropagation(); handleRemoveFromList(selectedList, entry.name) }} title="Remover">Remover</button>
                      <select className="settings-select" style={{ fontSize: '11px', padding: '2px 4px' }} value="" onChange={(e) => { if (e.target.value) handleMoveToList(selectedList, e.target.value, entry.name) }} onClick={e => e.stopPropagation()}>
                        <option value="">Mover para...</option>
                        {listNames.filter(n => n !== selectedList).map(n => <option key={n} value={n}>{n}</option>)}
                      </select>
                    </div>
                  </div>
                </div>
              ))}
              {(customLists[selectedList] || []).length === 0 && (
                <p style={{ color: 'var(--text-muted)', gridColumn: '1/-1', textAlign: 'center', padding: '40px' }}>Lista vazia</p>
              )}
            </div>
          </div>
        )}

        {view === 'queue' && (
          <div className="queue-panel" style={{ padding: '24px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <h2 style={{ color: 'var(--text)' }}>Fila de Reprodu\u00e7\u00e3o</h2>
              {playQueue.length > 0 && (
                <button className="btn-small" onClick={async () => { await clearQueue(); setPlayQueue([]) }}>Limpar fila</button>
              )}
            </div>
            {playQueue.length === 0 ? (
              <p className="queue-empty">Nenhum epis\u00f3dio na fila. Adicione epis\u00f3dios usando o bot\u00e3o na \u00e1rea do player.</p>
            ) : (
              playQueue.map((item, i) => (
                <div key={i} className="queue-item">
                  {item.imageUrl && <img src={item.imageUrl} alt={item.mediaName} style={{ width: '60px', height: '36px', objectFit: 'cover', borderRadius: '4px' }} />}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <p style={{ color: 'var(--text)', fontSize: '13px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{item.mediaName}</p>
                    <p style={{ color: 'var(--text-muted)', fontSize: '11px' }}>Ep {item.episodeNumber}</p>
                  </div>
                  <button className="btn-small" onClick={async () => { await removeFromQueue(i); const q = await getPlayQueue(); setPlayQueue(q) }}>X</button>
                </div>
              ))
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
            sections={catalogDisplaySections}
            loading={catalogDisplayLoading}
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

        {animeDetailLoading && !animeDetail && (
          <div className="anime-detail" style={{ padding: '40px 24px', textAlign: 'center' }}>
            <div className="status-bar" style={{ display: 'inline-flex' }}>
              <span className="status-dot loading" />
              <span>Carregando detalhes...</span>
            </div>
          </div>
        )}

        {animeDetail && (
          <div className="anime-detail">
            {animeDetail.bannerImage ? (
              <img className="anime-detail-banner" src={animeDetail.bannerImage} alt="" />
            ) : (
              <div className="anime-detail-banner" />
            )}
            <div className="anime-detail-header">
              {animeDetail.coverImage ? (
                <img className="anime-detail-cover" src={animeDetail.coverImage} alt={animeDetail.title} />
              ) : (
                <div className="anime-detail-cover" style={{ background: 'var(--surface-2)', display: 'grid', placeItems: 'center' }}>
                  <IconFilm />
                </div>
              )}
              <div className="anime-detail-info">
                <h2 className="anime-detail-title">{animeDetail.title}</h2>
                {animeDetail.titleRomaji && animeDetail.titleRomaji !== animeDetail.title && (
                  <p style={{ fontSize: '13px', color: 'var(--text-soft)', marginBottom: '8px' }}>{animeDetail.titleRomaji}</p>
                )}
                <div className="anime-detail-meta">
                  {animeDetail.score > 0 && <span className="anime-detail-score">{'\u2605'} {animeDetail.score.toFixed(1)}</span>}
                  {animeDetail.year > 0 && <span>{animeDetail.year}</span>}
                  {animeDetail.format && <span>{animeDetail.format}</span>}
                  {animeDetail.status && <span>{animeDetail.status === 'RELEASING' ? 'Em exibi\u00e7\u00e3o' : animeDetail.status === 'FINISHED' ? 'Finalizado' : animeDetail.status === 'NOT_YET_RELEASED' ? 'A lan\u00e7ar' : animeDetail.status}</span>}
                </div>
                {animeDetail.genres?.length > 0 && (
                  <div className="anime-detail-genres">
                    {animeDetail.genres.map(g => (
                      <span key={g} className="anime-detail-genre">{genreLabels[g] ?? g}</span>
                    ))}
                  </div>
                )}
                {animeDetail.totalEpisodes > 0 && (
                  <p style={{ fontSize: '13px', color: 'var(--text-soft)' }}>
                    {animeDetail.totalEpisodes} {animeDetail.totalEpisodes === 1 ? 'epis\u00f3dio' : 'epis\u00f3dios'}
                    {animeDetail.status === 'RELEASING' ? ' \u00b7 Em exibi\u00e7\u00e3o' : ''}
                  </p>
                )}
              </div>
            </div>
            {animeDetail.description && (
              <p className="anime-detail-synopsis">
                {animeDetail.description.replace(/<br\s*\/?>/gi, ' ').replace(/<[^>]+>/g, '')}
              </p>
            )}
            <div className="anime-detail-actions">
              <button className="anime-detail-btn anime-detail-btn-primary" onClick={handleDetailWatch} disabled={animeDetailLoading}>
                <IconPlay /> Assistir
              </button>
              <button className="anime-detail-btn anime-detail-btn-secondary" onClick={handleDetailClose}>
                Voltar
              </button>
            </div>
            {animeDetail.sources && animeDetail.sources.length > 0 && (
              <div className="anime-detail-sources">
                <strong>Fontes dispon{'\u00ed'}veis:</strong>{' '}
                {animeDetail.sources.map(s => s.source).join(' \u00b7 ')}
              </div>
            )}
            {!animeDetail.sources?.length && animeDetailMediaRef.current?.availableSources?.length ? (
              <div className="anime-detail-sources">
                <strong>Fontes dispon{'\u00ed'}veis:</strong>{' '}
                {animeDetailMediaRef.current.availableSources.join(' \u00b7 ')}
              </div>
            ) : null}
          </div>
        )}

        {searchState !== 'loading' && visibleResults.length > 0 && !animeDetail && (
          <div className="search-list">
            {visibleResults.map((item) => {
              const langTag = extractLangTag(item.name)
              const title = cleanTitle(item.name)
              return (
                <div
                  key={`${item.source}:${item.url}`}
                  className={`search-list-item${activeMedia?.url === item.url ? ' active' : ''}`}
                  onClick={() => void handleOpenMedia(item)}
                  title={`${title} — ${item.source}`}
                >
                  <span className={`search-list-tag tag-${langTag?.variant ?? 'source'}`}>
                    [{langTag?.label ?? item.source}]
                  </span>
                  <span className="search-list-title">{title}</span>
                  {item.score ? <span className="search-list-score">{item.score.toFixed(1)}</span> : null}
                </div>
              )
            })}
          </div>
        )}
      </main>

      {/* Player */}
      <aside className="player-panel">
        <button className="panel-close-btn" onClick={handleClosePlayer} title="Fechar player" type="button">&times;</button>
        <div
          ref={playerWrapRef}
          className="player-video-wrap"
          onDoubleClick={() => void handleToggleFullscreen()}
        >
          {iframeUrl ? (
            <iframe
              src={iframeUrl}
              className="player-video"
              style={{ border: 'none', width: '100%', height: '100%' }}
              allowFullScreen
              allow="autoplay; encrypted-media"
            />
          ) : (
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
          )}
          {showSkipIntro && <button className="skip-btn skip-intro" onClick={() => { if (videoRef.current && skipTimes) videoRef.current.currentTime = skipTimes.opEnd }}>Pular Intro</button>}
          {showSkipEnding && <button className="skip-btn skip-ending" onClick={() => { if (videoRef.current && skipTimes) videoRef.current.currentTime = skipTimes.edEnd }}>Pular Encerramento</button>}
        </div>

        <div className="player-body">
          <div className="player-heading">
            <div>
              <p className="player-title">{activeMedia ? cleanTitle(activeMedia.name) : 'Player'}</p>
              <p className={`player-subtitle${playerState === 'error' ? ' player-subtitle-error' : ''}`}>{playerMessage}</p>
              {playerState === 'error' && activeMedia && (
                <div style={{ display: 'flex', gap: '8px', marginTop: '6px' }}>
                  <button className="btn-retry" type="button" onClick={() => void startPlayback(episodeIndex)}>
                    Tentar novamente
                  </button>
                  {activeMedia.url && activeMedia.url.includes('bakashi') && episodes[episodeIndex]?.url && (
                    <button className="btn-retry" type="button" style={{ borderColor: 'rgba(108,99,255,0.4)', color: 'var(--accent-2)', background: 'rgba(108,99,255,0.1)' }} onClick={() => BrowserOpenURL(episodes[episodeIndex].url)}>
                      Abrir no navegador
                    </button>
                  )}
                </div>
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
                <div style={{ position: 'relative', display: 'inline-block' }}>
                  <button className="btn-small" onClick={() => setListDropdownFor(listDropdownFor ? null : (activeMedia ? cleanTitle(activeMedia.name) : null))} title="Adicionar a lista">
                    +Lista
                  </button>
                  {listDropdownFor && (
                    <div className="list-move-dropdown" style={{ position: 'absolute', top: '100%', left: 0, zIndex: 100 }}>
                      {listNames.map(name => (
                        <button key={name} className="list-move-option" onClick={() => handleAddToList(name)}>{name}</button>
                      ))}
                    </div>
                  )}
                </div>
                <button className="btn-small" onClick={() => setShowNotes(true)} disabled={!activeMedia} title="Notas" style={{ marginLeft: '4px' }}>
                  Notas{activeNote && activeNote.rating > 0 ? ` (${activeNote.rating}/10)` : ''}
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
                  <p className="episode-list-empty">{'\u2014'}</p>
                ) : (
                  <>
                    <input
                      className="episode-filter-input"
                      type="text"
                      placeholder="Filtrar episódios..."
                      value={episodeFilter}
                      onChange={e => setEpisodeFilter(e.target.value)}
                    />
                    <div className="episode-list" ref={episodeListRef}>
                      {filteredEpisodes.map(({ ep: episode, originalIndex }) => {
                        const epNum = getEpisodeNumber(episode, originalIndex)
                        const watched = activeProgress ? epNum < activeProgress.episodeNumber : false
                        const lastWatched = activeProgress ? epNum === activeProgress.episodeNumber : false
                        const isCurrent = originalIndex === episodeIndex
                        return (
                          <button
                            key={`${episode.url}-${originalIndex}`}
                            className={`ep-item${isCurrent ? ' current' : ''}${watched ? ' watched' : ''}${lastWatched ? ' last-watched' : ''}`}
                            onClick={() => setEpisodeIndex(originalIndex)}
                            title={episode.title || `Episódio ${epNum}`}
                          >
                            <input
                              type="checkbox"
                              className="watched-check"
                              checked={watchedMarks.has(epNum)}
                              onChange={async (e) => {
                                e.stopPropagation()
                                const checked = e.target.checked
                                const key = activeMedia ? (activeMedia.groupKey || normalizeSearchText(cleanTitle(activeMedia.name))) : ''
                                if (key) {
                                  await setEpisodeWatched(key, epNum, checked)
                                  setWatchedMarks(prev => { const next = new Set(prev); checked ? next.add(epNum) : next.delete(epNum); return next })
                                }
                              }}
                              onClick={e => e.stopPropagation()}
                            />
                            <span className="ep-item-num">Ep {epNum}</span>
                            {episode.title && <span className="ep-item-title">{episode.title}</span>}
                            {(watched || lastWatched) && <span className="ep-check">{'\u2713'}</span>}
                          </button>
                        )
                      })}
                    </div>
                  </>
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

              <div className="control-group">
                <span className="control-label">Velocidade</span>
                <select className="speed-select control-select" value={playbackSpeed} onChange={e => { const s = parseFloat(e.target.value); setPlaybackSpeed(s); if (videoRef.current) videoRef.current.playbackRate = s }}>
                  <option value={0.5}>0.5x</option>
                  <option value={0.75}>0.75x</option>
                  <option value={1}>1x</option>
                  <option value={1.25}>1.25x</option>
                  <option value={1.5}>1.5x</option>
                  <option value={2}>2x</option>
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
              <button className="btn btn-secondary" onClick={handleAddToQueue} disabled={!activeMedia || !currentEpisode}>
                + Fila
              </button>
              <button className="btn btn-secondary" onClick={() => void handleToggleFullscreen()}>
                <IconExpand active={isFullscreen} /> Tela cheia
              </button>
              {!iframeUrl && (
                <button className={`btn btn-secondary${isPip ? ' active' : ''}`} onClick={() => void handleTogglePip()} title="Picture-in-Picture (I)">
                  PiP
                </button>
              )}
            </div>

            <p className="keys-hint">Espa{'\u00E7'}o: pausar | Shift+{'\u2190'}{'\u2192'}: avan{'\u00E7'}ar 10s | N/P: epis{'\u00F3'}dios | F: tela cheia | M: mudo | I: PiP | []: velocidade</p>
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

      {/* Notes modal */}
      {showNotes && activeMedia && (
        <AnimeNotes
          title={cleanTitle(activeMedia.name)}
          note={activeNote}
          onSave={handleSaveNote}
          onClose={() => setShowNotes(false)}
        />
      )}

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
















































