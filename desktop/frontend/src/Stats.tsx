import type { WatchStats } from './lib/backend'

interface StatsProps {
  stats: WatchStats
}

export default function Stats({ stats }: StatsProps) {
  const hours = Math.floor(stats.totalMinutes / 60)

  const maxEpisodes = stats.recentActivity.reduce(
    (max, day) => Math.max(max, day.episodes),
    1
  )

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr)
    return `${d.getDate()}/${d.getMonth() + 1}`
  }

  return (
    <div className="stats-page">
      <div className="page-header">
        <h2>Estatisticas</h2>
      </div>

      <div className="stats-grid">
        <div className="stats-card">
          <span className="stats-card-value">{stats.totalAnime}</span>
          <span className="stats-card-label">Total Anime</span>
        </div>
        <div className="stats-card">
          <span className="stats-card-value">{stats.totalEpisodes}</span>
          <span className="stats-card-label">Total Episodios</span>
        </div>
        <div className="stats-card">
          <span className="stats-card-value">{hours}</span>
          <span className="stats-card-label">Horas Assistidas</span>
        </div>
        <div className="stats-card">
          <span className="stats-card-value">{stats.completedAnime}</span>
          <span className="stats-card-label">Animes Concluidos</span>
        </div>
      </div>

      <div className="stats-chart">
        {stats.recentActivity.slice(-14).map((day, i) => {
          const pct = maxEpisodes > 0 ? (day.episodes / maxEpisodes) * 100 : 0
          return (
            <div className="stats-bar-wrap" key={i}>
              <div
                className="stats-bar"
                style={{ height: `${pct}%` }}
                title={`${day.episodes} ep - ${day.minutes} min`}
              />
              <span className="stats-bar-date">{formatDate(day.date)}</span>
            </div>
          )
        })}
      </div>

      <div className="stats-bottom">
        <div className="stats-streak">
          <span className="stats-streak-icon">&#128293;</span>
          <div className="stats-streak-info">
            <span className="stats-streak-value">
              {stats.currentStreak} dias
            </span>
            <span className="stats-streak-label">Sequencia atual</span>
          </div>
        </div>
        <div className="stats-streak">
          <span className="stats-streak-icon">&#127942;</span>
          <div className="stats-streak-info">
            <span className="stats-streak-value">
              {stats.longestStreak} dias
            </span>
            <span className="stats-streak-label">Maior sequencia</span>
          </div>
        </div>
      </div>

      {stats.topGenres.length > 0 && (
        <div className="stats-genres">
          {stats.topGenres.map((genre) => (
            <span className="stats-genre-pill" key={genre}>
              {genre}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
