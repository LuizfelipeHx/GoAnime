export namespace main {
	
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

}

