const searchInput = document.getElementById("searchInput");
const sourceSelect = document.getElementById("sourceSelect");
const typeGroup = document.getElementById("typeGroup");
const statusText = document.getElementById("statusText");
const resultsGrid = document.getElementById("resultsGrid");
const cardTemplate = document.getElementById("cardTemplate");

const watchPanel = document.getElementById("watchPanel");
const watchTitle = document.getElementById("watchTitle");
const watchMeta = document.getElementById("watchMeta");
const episodeSelect = document.getElementById("episodeSelect");
const modeSelect = document.getElementById("modeSelect");
const qualitySelect = document.getElementById("qualitySelect");
const playButton = document.getElementById("playButton");
const videoPlayer = document.getElementById("videoPlayer");
const playerStatus = document.getElementById("playerStatus");

let selectedType = "all";
let debounceTimer = null;
let activeMedia = null;
let activeEpisodes = [];
let hlsInstance = null;

function setStatus(text) {
  statusText.textContent = text;
}

function setPlayerStatus(text) {
  playerStatus.textContent = text;
}

function normalizedSource(sourceLabel) {
  const label = (sourceLabel || "").toLowerCase();
  if (label.includes("allanime")) return "allanime";
  if (label.includes("animefire")) return "animefire";
  if (label.includes("flixhq")) return "flixhq";
  if (label.includes("animesonlinecc")) return "animesonlinecc";
  return "all";
}

function humanType(type) {
  if (type === "movie") return "Filme";
  if (type === "tv") return "Serie";
  return "Anime";
}

function buildCliCommand(item) {
  const source = normalizedSource(item.source);
  let cmd = `goanime --source ${source}`;

  if (item.mediaType === "movie") {
    cmd += " --type movie";
  } else if (item.mediaType === "tv") {
    cmd += " --type tv";
  }

  cmd += ` \"${item.name.replace(/\"/g, "'")}\"`;
  return cmd;
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

function clearCards() {
  resultsGrid.innerHTML = "";
}

function clearEpisodeOptions() {
  episodeSelect.innerHTML = "";
}

function fillEpisodeOptions(episodes) {
  clearEpisodeOptions();
  episodes.forEach((ep, idx) => {
    const option = document.createElement("option");
    option.value = String(idx);
    const fallback = ep.num > 0 ? `Episodio ${ep.num}` : "Episodio";
    option.textContent = ep.title ? `${fallback} - ${ep.title}` : fallback;
    episodeSelect.appendChild(option);
  });
}

function destroyHls() {
  if (hlsInstance) {
    hlsInstance.destroy();
    hlsInstance = null;
  }
}

function clearSubtitles() {
  videoPlayer.querySelectorAll("track").forEach((node) => node.remove());
}

function applySubtitles(subtitles) {
  clearSubtitles();
  if (!Array.isArray(subtitles)) return;

  subtitles.forEach((sub) => {
    if (!sub || (!sub.proxyUrl && !sub.url)) return;
    const track = document.createElement("track");
    track.kind = "subtitles";
    track.label = sub.label || sub.language || "Sub";
    track.srclang = (sub.language || "en").slice(0, 2).toLowerCase();
    track.src = sub.proxyUrl || sub.url;
    videoPlayer.appendChild(track);
  });
}

function isM3U8(url, contentType) {
  const u = (url || "").toLowerCase();
  const ct = (contentType || "").toLowerCase();
  return u.includes(".m3u8") || ct.includes("mpegurl");
}

async function attachVideo(url, contentType) {
  destroyHls();
  clearSubtitles();

  videoPlayer.pause();
  videoPlayer.removeAttribute("src");
  videoPlayer.load();

  if (!url) {
    setPlayerStatus("URL de stream vazia.");
    return;
  }

  const m3u8 = isM3U8(url, contentType);

  if (m3u8 && videoPlayer.canPlayType("application/vnd.apple.mpegurl")) {
    videoPlayer.src = url;
  } else if (m3u8 && window.Hls && window.Hls.isSupported()) {
    hlsInstance = new window.Hls({
      maxBufferLength: 30,
      enableWorker: true,
      lowLatencyMode: false,
    });
    hlsInstance.loadSource(url);
    hlsInstance.attachMedia(videoPlayer);
    hlsInstance.on(window.Hls.Events.ERROR, (_event, data) => {
      if (data?.fatal) {
        setPlayerStatus(`Falha HLS: ${data.type || "erro"}`);
      }
    });
  } else {
    videoPlayer.src = url;
  }

  try {
    await videoPlayer.play();
    setPlayerStatus("Reproducao iniciada.");
  } catch {
    setPlayerStatus("Stream carregado. Clique em Play no video.");
  }
}

async function apiGet(endpoint, params) {
  const query = new URLSearchParams(params);
  const response = await fetch(`${endpoint}?${query.toString()}`);
  let data = null;
  try {
    data = await response.json();
  } catch {
    data = null;
  }

  if (!response.ok) {
    const message = data?.error || `Erro ${response.status}`;
    throw new Error(message);
  }

  return data;
}

async function openMediaInPlayer(item) {
  activeMedia = item;
  activeEpisodes = [];
  watchPanel.classList.remove("hidden");
  watchTitle.textContent = item.name;
  watchMeta.textContent = `Carregando episodios de ${item.source}...`;
  setPlayerStatus("Consultando episodios...");

  try {
    const data = await apiGet("/api/episodes", {
      media_url: item.url,
      source: item.source,
      media_type: item.mediaType,
      name: item.name,
    });

    activeEpisodes = data.episodes || [];
    fillEpisodeOptions(activeEpisodes);

    if (!activeEpisodes.length) {
      setPlayerStatus("Nenhum episodio encontrado para esse titulo.");
      watchMeta.textContent = "Sem episodios disponiveis";
      return;
    }

    watchMeta.textContent = `${activeEpisodes.length} episodio(s) em ${item.source}`;
    setPlayerStatus("Selecione episodio e clique em Assistir agora.");

    if (item.mediaType === "movie") {
      modeSelect.value = "sub";
    }
  } catch (error) {
    clearEpisodeOptions();
    setPlayerStatus(error.message || "Falha ao carregar episodios.");
    watchMeta.textContent = "Erro ao carregar dados";
  }
}

async function playSelectedEpisode() {
  if (!activeMedia) {
    setPlayerStatus("Escolha um titulo primeiro.");
    return;
  }

  const idx = Number(episodeSelect.value || 0);
  const selected = activeEpisodes[idx];
  if (!selected) {
    setPlayerStatus("Selecione um episodio valido.");
    return;
  }

  setPlayerStatus("Carregando stream...");

  try {
    const streamData = await apiGet("/api/stream", {
      media_url: activeMedia.url,
      source: activeMedia.source,
      media_type: activeMedia.mediaType,
      name: activeMedia.name,
      episode_url: selected.url,
      episode_number: selected.num || selected.number || "1",
      mode: modeSelect.value,
      quality: qualitySelect.value,
    });

    const videoURL = streamData.proxyUrl || streamData.streamUrl;
    await attachVideo(videoURL, streamData.contentType);
    applySubtitles(streamData.subtitles || []);

    if (streamData.note) {
      setPlayerStatus(streamData.note);
    }
  } catch (error) {
    setPlayerStatus(error.message || "Erro ao carregar stream.");
  }
}

function renderCards(items) {
  clearCards();

  if (!items.length) {
    setStatus("Nada encontrado com esse filtro.");
    return;
  }

  const fragment = document.createDocumentFragment();

  items.forEach((item, index) => {
    const node = cardTemplate.content.cloneNode(true);
    const card = node.querySelector(".media-card");
    const img = node.querySelector("img");
    const fallback = node.querySelector(".poster-fallback");
    const chipType = node.querySelector(".chip-type");
    const chipSource = node.querySelector(".chip-source");
    const title = node.querySelector(".title");
    const meta = node.querySelector(".meta");
    const playBtn = node.querySelector(".play-btn");
    const copyBtn = node.querySelector(".copy-btn");
    const openLink = node.querySelector(".open-link");

    card.style.animationDelay = `${Math.min(index * 0.03, 0.3)}s`;
    chipType.textContent = humanType(item.mediaType);
    chipSource.textContent = item.source || "Fonte desconhecida";
    title.textContent = item.name;

    const year = item.year ? `Ano ${item.year}` : "Ano nao informado";
    meta.textContent = `${year}`;

    openLink.href = item.url;

    playBtn.addEventListener("click", () => openMediaInPlayer(item));

    const cliCommand = buildCliCommand(item);
    copyBtn.addEventListener("click", async () => {
      const ok = await copyText(cliCommand);
      copyBtn.textContent = ok ? "Copiado" : "Falhou";
      setTimeout(() => {
        copyBtn.textContent = "Copiar comando";
      }, 1300);
    });

    if (item.imageUrl) {
      img.src = item.imageUrl;
      img.alt = item.name;
      img.onload = () => {
        img.classList.add("ready");
        fallback.style.display = "none";
      };
      img.onerror = () => {
        img.removeAttribute("src");
      };
    }

    fragment.appendChild(node);
  });

  resultsGrid.appendChild(fragment);
  setStatus(`${items.length} resultado(s) carregado(s).`);
}

async function runSearch() {
  const query = searchInput.value.trim();
  if (query.length < 2) {
    clearCards();
    setStatus("Digite pelo menos 2 caracteres para buscar.");
    return;
  }

  const source = sourceSelect.value;
  const params = new URLSearchParams({ q: query, source, type: selectedType });

  setStatus("Buscando...");
  try {
    const response = await fetch(`/api/search?${params.toString()}`);
    const data = await response.json();

    if (!response.ok) {
      clearCards();
      setStatus(data.error || "Erro ao buscar resultados.");
      return;
    }

    renderCards(data.results || []);
  } catch {
    clearCards();
    setStatus("Falha de rede ao consultar a API local.");
  }
}

function scheduleSearch() {
  if (debounceTimer) {
    clearTimeout(debounceTimer);
  }
  debounceTimer = setTimeout(runSearch, 320);
}

searchInput.addEventListener("input", scheduleSearch);
sourceSelect.addEventListener("change", runSearch);
playButton.addEventListener("click", playSelectedEpisode);

typeGroup.addEventListener("click", (event) => {
  const button = event.target.closest("button[data-type]");
  if (!button) return;

  selectedType = button.dataset.type || "all";

  typeGroup.querySelectorAll(".pill").forEach((pill) => {
    pill.classList.remove("active");
  });
  button.classList.add("active");

  runSearch();
});
