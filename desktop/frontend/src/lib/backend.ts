export type MediaType = 'all' | 'anime' | 'movie' | 'tv'

export type MediaAlternative = {
  name: string
  url: string
  source: string
  mediaType: string
}

export type MediaResult = {
  name: string
  url: string
  imageUrl: string
  source: string
  mediaType: 'anime' | 'movie' | 'tv'
  year?: string
  score?: number
  description?: string
  genres?: string[]
  canonicalTitle?: string
  groupKey?: string
  seasonNumber?: number
  availableSources?: string[]
  watchSource?: string
  downloadSource?: string
  dubSource?: string
  subSource?: string
  alternatives?: MediaAlternative[]
  hasPortuguese?: boolean
  hasEnglish?: boolean
  hasDub?: boolean
  hasSub?: boolean
  watchHasPortuguese?: boolean
  watchHasEnglish?: boolean
  watchHasDub?: boolean
  watchHasSub?: boolean
}

export type MediaRequest = {
  name: string
  url: string
  source: string
  mediaType: MediaType | 'anime' | 'movie' | 'tv'
  groupKey?: string
  alternatives?: MediaAlternative[]
  hasPortuguese?: boolean
  hasEnglish?: boolean
  hasDub?: boolean
  hasSub?: boolean
  watchHasPortuguese?: boolean
  watchHasEnglish?: boolean
  watchHasDub?: boolean
  watchHasSub?: boolean
}

export type EpisodeResult = {
  number: string
  num: number
  title: string
  url: string
}

export type EpisodesResponse = {
  name: string
  source: string
  mediaType: string
  count: number
  episodes: EpisodeResult[]
  resolvedSource?: string
  resolvedUrl?: string
  note?: string
}

export type SubtitleResult = {
  url: string
  proxyUrl: string
  language: string
  label: string
}

export type StreamRequest = {
  media: MediaRequest
  episodeUrl: string
  episodeNumber: string
  mode: 'sub' | 'dub'
  quality: string
}

export type StreamResponse = {
  streamUrl: string
  proxyUrl: string
  contentType: string
  subtitles?: SubtitleResult[]
  note?: string
  resolvedSource?: string
  resolvedUrl?: string
  resolvedEpisodeUrl?: string
}

export type HistoryEntry = {
  name: string
}

export type WatchProgressEntry = {
  allanimeId: string
  title: string
  episodeNumber: number
  playbackTime: number
  duration: number
  progressPercent: number
  totalEpisodes: number
  remainingEpisodes: number
  mediaType: string
  lastUpdated: string
}

export type FavoriteEntry = {
  title: string
  imageUrl: string
  url: string
  source: string
  mediaType: string
  addedAt: string
}

export type SearchCoversEvent = {
  query: string
  source: string
  mediaType: string
  results: MediaResult[]
}

export type RelatedAnime = {
  malId: number
  name: string
  relation: string
  imageUrl: string
}

export type UpdateWatchProgressRequest = {
  allanimeId: string
  title: string
  episodeNumber: number
  playbackTime: number
  duration: number
  mediaType: string
}

export type DownloadEpisodeRequest = {
  media: MediaRequest
  episodeUrl: string
  episodeNumber: string
  mode: 'sub' | 'dub'
  quality: string
}

export type DownloadEpisodeResponse = {
  filePath: string
  message: string
}

export type CatalogItem = {
  id: number
  title: string
  coverImage: string
  bannerImage: string
  score: number
  genres: string[]
  episodes: number
  description: string
  status: string
}

export type CatalogSection = {
  label: string
  items: CatalogItem[]
}

type WailsApp = {
  SearchMedia: (query: string, source: string, mediaType: string) => Promise<MediaResult[]>
  GetRelatedAnime: (title: string) => Promise<RelatedAnime[]>
  GetEpisodes: (request: MediaRequest) => Promise<EpisodesResponse>
  GetStream: (request: StreamRequest) => Promise<StreamResponse>
  GetProxyBaseURL: () => Promise<string>
  GetSearchHistory: () => Promise<HistoryEntry[]>
  GetWatchProgress: () => Promise<WatchProgressEntry[]>
  GetCatalog: () => Promise<CatalogSection[]>
  GetCatalogByGenre: (genre: string) => Promise<CatalogSection[]>
  GetGenres: () => Promise<string[]>
  GetMovieCatalog: () => Promise<CatalogSection[]>
  GetFavorites: () => Promise<FavoriteEntry[]>
  AddFavorite: (entry: FavoriteEntry) => Promise<void>
  RemoveFavorite: (title: string) => Promise<void>
  UpdateWatchProgress: (request: UpdateWatchProgressRequest) => Promise<void>
  DownloadEpisode: (request: DownloadEpisodeRequest) => Promise<DownloadEpisodeResponse>
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: WailsApp
      }
    }
    runtime?: {
      WindowMinimise: () => void
      WindowToggleMaximise: () => void
      Quit: () => void
    }
  }
}

const getApp = (): WailsApp => {
  const app = window.go?.main?.App
  if (!app) {
    throw new Error('Wails runtime indisponivel. Rode via `wails dev` ou build do desktop.')
  }
  return app
}

export async function searchMedia(query: string, source: string, mediaType: string) {
  return getApp().SearchMedia(query, source, mediaType)
}

export async function getEpisodes(request: MediaRequest) {
  return getApp().GetEpisodes(request)
}

export async function getStream(request: StreamRequest) {
  return getApp().GetStream(request)
}

export async function getProxyBaseURL() {
  return getApp().GetProxyBaseURL()
}

export async function getSearchHistory(): Promise<HistoryEntry[]> {
  return getApp().GetSearchHistory()
}

export async function getWatchProgress(): Promise<WatchProgressEntry[]> {
  return getApp().GetWatchProgress()
}

export async function getCatalog(): Promise<CatalogSection[]> {
  return getApp().GetCatalog()
}

export async function getCatalogByGenre(genre: string): Promise<CatalogSection[]> {
  return getApp().GetCatalogByGenre(genre)
}

export async function getGenres(): Promise<string[]> {
  return getApp().GetGenres()
}

export async function getMovieCatalog(): Promise<CatalogSection[]> {
  return getApp().GetMovieCatalog()
}

export async function getFavorites(): Promise<FavoriteEntry[]> {
  return getApp().GetFavorites()
}

export async function addFavorite(entry: FavoriteEntry): Promise<void> {
  return getApp().AddFavorite(entry)
}

export async function removeFavorite(title: string): Promise<void> {
  return getApp().RemoveFavorite(title)
}

export async function updateWatchProgress(request: UpdateWatchProgressRequest): Promise<void> {
  return getApp().UpdateWatchProgress(request)
}

export async function downloadEpisode(request: DownloadEpisodeRequest): Promise<DownloadEpisodeResponse> {
  return getApp().DownloadEpisode(request)
}

export async function getRelatedAnime(title: string): Promise<RelatedAnime[]> {
  return getApp().GetRelatedAnime(title)
}


