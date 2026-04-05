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
  totalEpisodes?: number
  anilistId?: number
  malId?: number
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

export type NyaaRelease = {
  title: string
  link: string
  infoHash?: string
  size: string
  date: string
  seeders: number
  category?: string
  isNew: boolean
}

export type AIRecommendation = {
  title: string
  reason: string
  genres?: string
  score?: string
}

export type CuratedRelease = {
  release: NyaaRelease
  quality: string // "Excelente" | "Bom" | "Regular"
  summary: string
}

export type BotStatus = {
  aiOnline: boolean
  aiModel?: string
  releasesCount: number
  newReleases: number
  lastCheck?: string
  recsAvailable: boolean
  recsCount: number
  curatedCount: number
}

export interface AppSettings {
  downloadFolder: string
  defaultMode: string
  defaultQuality: string
  autoplayNext: boolean
  notificationsEnabled: boolean
  playbackSpeed: number
}

export interface CalendarDay {
  day: string
  entries: CalendarEntry[]
}

export interface CalendarEntry {
  title: string
  imageUrl: string
  episode: number
  totalEpisodes: number
  airingAt: number
  format: string
}

export interface SkipTimesResult {
  opStart: number
  opEnd: number
  edStart: number
  edEnd: number
  found: boolean
}

export interface ListEntry {
  name: string
  url: string
  imageUrl: string
  source: string
  listName: string
}

export interface WatchStats {
  totalAnime: number
  totalEpisodes: number
  totalMinutes: number
  completedAnime: number
  topGenres: string[]
  currentStreak: number
  longestStreak: number
  recentActivity: ActivityDay[]
}

export interface ActivityDay {
  date: string
  episodes: number
  minutes: number
}

export interface QueueEntry {
  mediaName: string
  url: string
  source: string
  mediaType: string
  episodeUrl: string
  episodeNumber: string
  imageUrl: string
}

export interface AnimeNote {
  title: string
  note: string
  rating: number
  updatedAt: string
}

export interface AniListProfile {
  id: number
  name: string
  avatar: string
  siteUrl: string
}

export interface AniListSyncStatus {
  connected: boolean
  profile?: AniListProfile
  lastSync?: string
  tokenStored: boolean
}

export interface AnimeLibraryEntry {
  anilistId: number
  malId: number
  title: string
  titleRomaji: string
  titleEnglish: string
  coverImage: string
  bannerImage: string
  genres: string[]
  description: string
  totalEpisodes: number
  score: number
  status: string
  format: string
  year: number
  sources: SourceMapping[]
  lastUpdated: string
}

export interface SourceMapping {
  source: string
  url: string
  name: string
  mediaType: string
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
  GetBotStatus: () => Promise<BotStatus>
  GetNyaaReleases: () => Promise<NyaaRelease[]>
  ClearNewReleases: () => Promise<void>
  GetAIRecommendations: () => Promise<AIRecommendation[]>
  RefreshRecommendations: () => Promise<AIRecommendation[]>
  GetCuratedReleases: () => Promise<CuratedRelease[]>
  RefreshCuratedReleases: () => Promise<CuratedRelease[]>
  GetSettings: () => Promise<AppSettings>
  SaveSettings: (settings: AppSettings) => Promise<void>
  GetWatchedEpisodes: (groupKey: string) => Promise<number[]>
  SetEpisodeWatched: (groupKey: string, episodeNum: number, watched: boolean) => Promise<void>
  SetAllWatchedUpTo: (groupKey: string, upToEpisode: number) => Promise<void>
  GetSeasonCalendar: () => Promise<CalendarDay[]>
  GetSkipTimes: (malID: number, episodeNum: number) => Promise<SkipTimesResult>
  GetCustomLists: () => Promise<Record<string, ListEntry[]>>
  GetListNames: () => Promise<string[]>
  AddToList: (listName: string, entry: ListEntry) => Promise<void>
  RemoveFromList: (listName: string, name: string) => Promise<void>
  MoveToList: (fromList: string, toList: string, name: string) => Promise<void>
  GetWatchStats: () => Promise<WatchStats>
  GetPlayQueue: () => Promise<QueueEntry[]>
  AddToQueue: (entry: QueueEntry) => Promise<void>
  RemoveFromQueue: (index: number) => Promise<void>
  ClearQueue: () => Promise<void>
  ReorderQueue: (fromIndex: number, toIndex: number) => Promise<void>
  GetAnimeNote: (title: string) => Promise<AnimeNote | null>
  SaveAnimeNote: (note: AnimeNote) => Promise<void>
  GetAllNotes: () => Promise<AnimeNote[]>
  DeleteAnimeNote: (title: string) => Promise<void>
  GetAniListSyncStatus: () => Promise<AniListSyncStatus>
  StartAniListAuth: () => Promise<string>
  DisconnectAniList: () => Promise<void>
  SyncToAniList: () => Promise<void>
  GetAnimeDetails: (anilistId: number) => Promise<AnimeLibraryEntry>
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

export async function getBotStatus(): Promise<BotStatus> {
  return getApp().GetBotStatus()
}

export async function getNyaaReleases(): Promise<NyaaRelease[]> {
  return getApp().GetNyaaReleases()
}

export async function clearNewReleases(): Promise<void> {
  return getApp().ClearNewReleases()
}

export async function getAIRecommendations(): Promise<AIRecommendation[]> {
  return getApp().GetAIRecommendations()
}

export async function refreshRecommendations(): Promise<AIRecommendation[]> {
  return getApp().RefreshRecommendations()
}

export async function getCuratedReleases(): Promise<CuratedRelease[]> {
  return getApp().GetCuratedReleases()
}

export async function refreshCuratedReleases(): Promise<CuratedRelease[]> {
  return getApp().RefreshCuratedReleases()
}

export async function getSettings(): Promise<AppSettings> {
  return getApp().GetSettings()
}

export async function saveSettings(settings: AppSettings): Promise<void> {
  return getApp().SaveSettings(settings)
}

export async function getWatchedEpisodes(groupKey: string): Promise<number[]> {
  return getApp().GetWatchedEpisodes(groupKey)
}

export async function setEpisodeWatched(groupKey: string, episodeNum: number, watched: boolean): Promise<void> {
  return getApp().SetEpisodeWatched(groupKey, episodeNum, watched)
}

export async function setAllWatchedUpTo(groupKey: string, upToEpisode: number): Promise<void> {
  return getApp().SetAllWatchedUpTo(groupKey, upToEpisode)
}

export async function getSeasonCalendar(): Promise<CalendarDay[]> {
  return getApp().GetSeasonCalendar()
}

export async function getSkipTimes(malID: number, episodeNum: number): Promise<SkipTimesResult> {
  return getApp().GetSkipTimes(malID, episodeNum)
}

export async function getCustomLists(): Promise<Record<string, ListEntry[]>> {
  return getApp().GetCustomLists()
}

export async function getListNames(): Promise<string[]> {
  return getApp().GetListNames()
}

export async function addToList(listName: string, entry: ListEntry): Promise<void> {
  return getApp().AddToList(listName, entry)
}

export async function removeFromList(listName: string, name: string): Promise<void> {
  return getApp().RemoveFromList(listName, name)
}

export async function moveToList(fromList: string, toList: string, name: string): Promise<void> {
  return getApp().MoveToList(fromList, toList, name)
}

export async function getWatchStats(): Promise<WatchStats> {
  return getApp().GetWatchStats()
}

export async function getPlayQueue(): Promise<QueueEntry[]> {
  return getApp().GetPlayQueue()
}

export async function addToQueue(entry: QueueEntry): Promise<void> {
  return getApp().AddToQueue(entry)
}

export async function removeFromQueue(index: number): Promise<void> {
  return getApp().RemoveFromQueue(index)
}

export async function clearQueue(): Promise<void> {
  return getApp().ClearQueue()
}

export async function reorderQueue(fromIndex: number, toIndex: number): Promise<void> {
  return getApp().ReorderQueue(fromIndex, toIndex)
}

export async function getAnimeNote(title: string): Promise<AnimeNote | null> {
  return getApp().GetAnimeNote(title)
}

export async function saveAnimeNote(note: AnimeNote): Promise<void> {
  return getApp().SaveAnimeNote(note)
}

export async function getAllNotes(): Promise<AnimeNote[]> {
  return getApp().GetAllNotes()
}

export async function deleteAnimeNote(title: string): Promise<void> {
  return getApp().DeleteAnimeNote(title)
}

export async function getAniListSyncStatus(): Promise<AniListSyncStatus> {
  return getApp().GetAniListSyncStatus()
}

export async function startAniListAuth(): Promise<string> {
  return getApp().StartAniListAuth()
}

export async function disconnectAniList(): Promise<void> {
  return getApp().DisconnectAniList()
}

export async function syncToAniList(): Promise<void> {
  return getApp().SyncToAniList()
}

export async function getAnimeDetails(anilistId: number): Promise<AnimeLibraryEntry> {
  return getApp().GetAnimeDetails(anilistId)
}
