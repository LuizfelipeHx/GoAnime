export namespace main {
	
	export class AIRecommendation {
	    title: string;
	    reason: string;
	    genres?: string;
	    score?: string;
	
	    static createFrom(source: any = {}) {
	        return new AIRecommendation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.reason = source["reason"];
	        this.genres = source["genres"];
	        this.score = source["score"];
	    }
	}
	export class ActivityDay {
	    date: string;
	    episodes: number;
	    minutes: number;
	
	    static createFrom(source: any = {}) {
	        return new ActivityDay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.episodes = source["episodes"];
	        this.minutes = source["minutes"];
	    }
	}
	export class AniListProfile {
	    id: number;
	    name: string;
	    avatar: string;
	    siteUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new AniListProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.avatar = source["avatar"];
	        this.siteUrl = source["siteUrl"];
	    }
	}
	export class AniListSyncStatus {
	    connected: boolean;
	    profile?: AniListProfile;
	    lastSync?: string;
	    tokenStored: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AniListSyncStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.profile = this.convertValues(source["profile"], AniListProfile);
	        this.lastSync = source["lastSync"];
	        this.tokenStored = source["tokenStored"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SourceMapping {
	    source: string;
	    url: string;
	    name: string;
	    mediaType: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.url = source["url"];
	        this.name = source["name"];
	        this.mediaType = source["mediaType"];
	    }
	}
	export class AnimeLibraryEntry {
	    anilistId: number;
	    malId?: number;
	    title: string;
	    titleRomaji?: string;
	    titleEnglish?: string;
	    coverImage?: string;
	    bannerImage?: string;
	    genres?: string[];
	    description?: string;
	    totalEpisodes?: number;
	    score?: number;
	    status?: string;
	    format?: string;
	    year?: number;
	    sources: SourceMapping[];
	    lastUpdated: string;
	
	    static createFrom(source: any = {}) {
	        return new AnimeLibraryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anilistId = source["anilistId"];
	        this.malId = source["malId"];
	        this.title = source["title"];
	        this.titleRomaji = source["titleRomaji"];
	        this.titleEnglish = source["titleEnglish"];
	        this.coverImage = source["coverImage"];
	        this.bannerImage = source["bannerImage"];
	        this.genres = source["genres"];
	        this.description = source["description"];
	        this.totalEpisodes = source["totalEpisodes"];
	        this.score = source["score"];
	        this.status = source["status"];
	        this.format = source["format"];
	        this.year = source["year"];
	        this.sources = this.convertValues(source["sources"], SourceMapping);
	        this.lastUpdated = source["lastUpdated"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AnimeNote {
	    title: string;
	    note: string;
	    rating: number;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AnimeNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.note = source["note"];
	        this.rating = source["rating"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class AppSettings {
	    downloadFolder: string;
	    defaultMode: string;
	    defaultQuality: string;
	    autoplayNext: boolean;
	    notificationsEnabled: boolean;
	    playbackSpeed: number;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.downloadFolder = source["downloadFolder"];
	        this.defaultMode = source["defaultMode"];
	        this.defaultQuality = source["defaultQuality"];
	        this.autoplayNext = source["autoplayNext"];
	        this.notificationsEnabled = source["notificationsEnabled"];
	        this.playbackSpeed = source["playbackSpeed"];
	    }
	}
	export class BotStatus {
	    aiOnline: boolean;
	    aiModel?: string;
	    releasesCount: number;
	    newReleases: number;
	    lastCheck?: string;
	    recsAvailable: boolean;
	    recsCount: number;
	    curatedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new BotStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.aiOnline = source["aiOnline"];
	        this.aiModel = source["aiModel"];
	        this.releasesCount = source["releasesCount"];
	        this.newReleases = source["newReleases"];
	        this.lastCheck = source["lastCheck"];
	        this.recsAvailable = source["recsAvailable"];
	        this.recsCount = source["recsCount"];
	        this.curatedCount = source["curatedCount"];
	    }
	}
	export class CalendarEntry {
	    title: string;
	    imageUrl: string;
	    episode: number;
	    totalEpisodes: number;
	    airingAt: number;
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new CalendarEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.imageUrl = source["imageUrl"];
	        this.episode = source["episode"];
	        this.totalEpisodes = source["totalEpisodes"];
	        this.airingAt = source["airingAt"];
	        this.format = source["format"];
	    }
	}
	export class CalendarDay {
	    day: string;
	    entries: CalendarEntry[];
	
	    static createFrom(source: any = {}) {
	        return new CalendarDay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.day = source["day"];
	        this.entries = this.convertValues(source["entries"], CalendarEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CatalogItem {
	    id: number;
	    title: string;
	    coverImage: string;
	    bannerImage: string;
	    score: number;
	    genres: string[];
	    episodes: number;
	    description: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new CatalogItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.coverImage = source["coverImage"];
	        this.bannerImage = source["bannerImage"];
	        this.score = source["score"];
	        this.genres = source["genres"];
	        this.episodes = source["episodes"];
	        this.description = source["description"];
	        this.status = source["status"];
	    }
	}
	export class CatalogSection {
	    label: string;
	    items: CatalogItem[];
	
	    static createFrom(source: any = {}) {
	        return new CatalogSection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.items = this.convertValues(source["items"], CatalogItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NyaaRelease {
	    title: string;
	    link: string;
	    infoHash?: string;
	    size: string;
	    // Go type: time
	    date: any;
	    seeders: number;
	    category?: string;
	    isNew: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NyaaRelease(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.link = source["link"];
	        this.infoHash = source["infoHash"];
	        this.size = source["size"];
	        this.date = this.convertValues(source["date"], null);
	        this.seeders = source["seeders"];
	        this.category = source["category"];
	        this.isNew = source["isNew"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CuratedRelease {
	    release: NyaaRelease;
	    quality: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new CuratedRelease(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.release = this.convertValues(source["release"], NyaaRelease);
	        this.quality = source["quality"];
	        this.summary = source["summary"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MediaAlternative {
	    name: string;
	    url: string;
	    source: string;
	    mediaType: string;
	
	    static createFrom(source: any = {}) {
	        return new MediaAlternative(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.source = source["source"];
	        this.mediaType = source["mediaType"];
	    }
	}
	export class MediaRequest {
	    name: string;
	    url: string;
	    source: string;
	    mediaType: string;
	    groupKey?: string;
	    alternatives?: MediaAlternative[];
	
	    static createFrom(source: any = {}) {
	        return new MediaRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.source = source["source"];
	        this.mediaType = source["mediaType"];
	        this.groupKey = source["groupKey"];
	        this.alternatives = this.convertValues(source["alternatives"], MediaAlternative);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DownloadEpisodeRequest {
	    media: MediaRequest;
	    episodeUrl: string;
	    episodeNumber: string;
	    mode: string;
	    quality: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadEpisodeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.media = this.convertValues(source["media"], MediaRequest);
	        this.episodeUrl = source["episodeUrl"];
	        this.episodeNumber = source["episodeNumber"];
	        this.mode = source["mode"];
	        this.quality = source["quality"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DownloadEpisodeResponse {
	    filePath: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadEpisodeResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.message = source["message"];
	    }
	}
	export class EpisodeResult {
	    number: string;
	    num: number;
	    title: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new EpisodeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.num = source["num"];
	        this.title = source["title"];
	        this.url = source["url"];
	    }
	}
	export class EpisodesResponse {
	    name: string;
	    source: string;
	    mediaType: string;
	    count: number;
	    episodes: EpisodeResult[];
	    resolvedSource?: string;
	    resolvedUrl?: string;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new EpisodesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.source = source["source"];
	        this.mediaType = source["mediaType"];
	        this.count = source["count"];
	        this.episodes = this.convertValues(source["episodes"], EpisodeResult);
	        this.resolvedSource = source["resolvedSource"];
	        this.resolvedUrl = source["resolvedUrl"];
	        this.note = source["note"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FavoriteEntry {
	    title: string;
	    imageUrl: string;
	    url: string;
	    source: string;
	    mediaType: string;
	    addedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.imageUrl = source["imageUrl"];
	        this.url = source["url"];
	        this.source = source["source"];
	        this.mediaType = source["mediaType"];
	        this.addedAt = source["addedAt"];
	    }
	}
	export class HistoryEntry {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class ListEntry {
	    anilistId?: number;
	    name: string;
	    url: string;
	    imageUrl: string;
	    source: string;
	    listName: string;
	
	    static createFrom(source: any = {}) {
	        return new ListEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.anilistId = source["anilistId"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.imageUrl = source["imageUrl"];
	        this.source = source["source"];
	        this.listName = source["listName"];
	    }
	}
	
	
	export class MediaResult {
	    name: string;
	    url: string;
	    imageUrl: string;
	    source: string;
	    mediaType: string;
	    year: string;
	    score?: number;
	    description?: string;
	    genres?: string[];
	    totalEpisodes?: number;
	    anilistId?: number;
	    malId?: number;
	    canonicalTitle?: string;
	    groupKey?: string;
	    seasonNumber?: number;
	    availableSources?: string[];
	    watchSource?: string;
	    downloadSource?: string;
	    dubSource?: string;
	    subSource?: string;
	    alternatives?: MediaAlternative[];
	    hasPortuguese?: boolean;
	    hasEnglish?: boolean;
	    hasDub?: boolean;
	    hasSub?: boolean;
	    watchHasPortuguese?: boolean;
	    watchHasEnglish?: boolean;
	    watchHasDub?: boolean;
	    watchHasSub?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MediaResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.imageUrl = source["imageUrl"];
	        this.source = source["source"];
	        this.mediaType = source["mediaType"];
	        this.year = source["year"];
	        this.score = source["score"];
	        this.description = source["description"];
	        this.genres = source["genres"];
	        this.totalEpisodes = source["totalEpisodes"];
	        this.anilistId = source["anilistId"];
	        this.malId = source["malId"];
	        this.canonicalTitle = source["canonicalTitle"];
	        this.groupKey = source["groupKey"];
	        this.seasonNumber = source["seasonNumber"];
	        this.availableSources = source["availableSources"];
	        this.watchSource = source["watchSource"];
	        this.downloadSource = source["downloadSource"];
	        this.dubSource = source["dubSource"];
	        this.subSource = source["subSource"];
	        this.alternatives = this.convertValues(source["alternatives"], MediaAlternative);
	        this.hasPortuguese = source["hasPortuguese"];
	        this.hasEnglish = source["hasEnglish"];
	        this.hasDub = source["hasDub"];
	        this.hasSub = source["hasSub"];
	        this.watchHasPortuguese = source["watchHasPortuguese"];
	        this.watchHasEnglish = source["watchHasEnglish"];
	        this.watchHasDub = source["watchHasDub"];
	        this.watchHasSub = source["watchHasSub"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class QueueEntry {
	    mediaName: string;
	    url: string;
	    source: string;
	    mediaType: string;
	    episodeUrl: string;
	    episodeNumber: string;
	    imageUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new QueueEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mediaName = source["mediaName"];
	        this.url = source["url"];
	        this.source = source["source"];
	        this.mediaType = source["mediaType"];
	        this.episodeUrl = source["episodeUrl"];
	        this.episodeNumber = source["episodeNumber"];
	        this.imageUrl = source["imageUrl"];
	    }
	}
	export class RelatedAnime {
	    malId: number;
	    name: string;
	    relation: string;
	    imageUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new RelatedAnime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.malId = source["malId"];
	        this.name = source["name"];
	        this.relation = source["relation"];
	        this.imageUrl = source["imageUrl"];
	    }
	}
	export class SkipTimesResult {
	    opStart: number;
	    opEnd: number;
	    edStart: number;
	    edEnd: number;
	    found: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SkipTimesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.opStart = source["opStart"];
	        this.opEnd = source["opEnd"];
	        this.edStart = source["edStart"];
	        this.edEnd = source["edEnd"];
	        this.found = source["found"];
	    }
	}
	
	export class StreamRequest {
	    media: MediaRequest;
	    episodeUrl: string;
	    episodeNumber: string;
	    mode: string;
	    quality: string;
	
	    static createFrom(source: any = {}) {
	        return new StreamRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.media = this.convertValues(source["media"], MediaRequest);
	        this.episodeUrl = source["episodeUrl"];
	        this.episodeNumber = source["episodeNumber"];
	        this.mode = source["mode"];
	        this.quality = source["quality"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SubtitleResult {
	    url: string;
	    proxyUrl: string;
	    language: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new SubtitleResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.proxyUrl = source["proxyUrl"];
	        this.language = source["language"];
	        this.label = source["label"];
	    }
	}
	export class StreamResponse {
	    streamUrl: string;
	    proxyUrl: string;
	    contentType: string;
	    subtitles?: SubtitleResult[];
	    note?: string;
	    resolvedSource?: string;
	    resolvedUrl?: string;
	    resolvedEpisodeUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new StreamResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.streamUrl = source["streamUrl"];
	        this.proxyUrl = source["proxyUrl"];
	        this.contentType = source["contentType"];
	        this.subtitles = this.convertValues(source["subtitles"], SubtitleResult);
	        this.note = source["note"];
	        this.resolvedSource = source["resolvedSource"];
	        this.resolvedUrl = source["resolvedUrl"];
	        this.resolvedEpisodeUrl = source["resolvedEpisodeUrl"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class UpdateWatchProgressRequest {
	    allanimeId: string;
	    title: string;
	    episodeNumber: number;
	    playbackTime: number;
	    duration: number;
	    mediaType: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateWatchProgressRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allanimeId = source["allanimeId"];
	        this.title = source["title"];
	        this.episodeNumber = source["episodeNumber"];
	        this.playbackTime = source["playbackTime"];
	        this.duration = source["duration"];
	        this.mediaType = source["mediaType"];
	    }
	}
	export class WatchProgressEntry {
	    allanimeId: string;
	    title: string;
	    episodeNumber: number;
	    playbackTime: number;
	    duration: number;
	    progressPercent: number;
	    totalEpisodes: number;
	    remainingEpisodes: number;
	    mediaType: string;
	    lastUpdated: string;
	
	    static createFrom(source: any = {}) {
	        return new WatchProgressEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allanimeId = source["allanimeId"];
	        this.title = source["title"];
	        this.episodeNumber = source["episodeNumber"];
	        this.playbackTime = source["playbackTime"];
	        this.duration = source["duration"];
	        this.progressPercent = source["progressPercent"];
	        this.totalEpisodes = source["totalEpisodes"];
	        this.remainingEpisodes = source["remainingEpisodes"];
	        this.mediaType = source["mediaType"];
	        this.lastUpdated = source["lastUpdated"];
	    }
	}
	export class WatchStats {
	    totalAnime: number;
	    totalEpisodes: number;
	    totalMinutes: number;
	    completedAnime: number;
	    topGenres: string[];
	    currentStreak: number;
	    longestStreak: number;
	    recentActivity: ActivityDay[];
	
	    static createFrom(source: any = {}) {
	        return new WatchStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalAnime = source["totalAnime"];
	        this.totalEpisodes = source["totalEpisodes"];
	        this.totalMinutes = source["totalMinutes"];
	        this.completedAnime = source["completedAnime"];
	        this.topGenres = source["topGenres"];
	        this.currentStreak = source["currentStreak"];
	        this.longestStreak = source["longestStreak"];
	        this.recentActivity = this.convertValues(source["recentActivity"], ActivityDay);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

