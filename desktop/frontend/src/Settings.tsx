import { useState } from 'react'
import type { AppSettings } from './lib/backend'

interface SettingsProps {
  settings: AppSettings
  onSave: (s: AppSettings) => void
}

export default function Settings({ settings, onSave }: SettingsProps) {
  const [form, setForm] = useState<AppSettings>({ ...settings })

  const update = <K extends keyof AppSettings>(key: K, value: AppSettings[K]) => {
    setForm(prev => ({ ...prev, [key]: value }))
  }

  return (
    <div className="settings-page">
      <div className="page-header">
        <h2>Configurações</h2>
      </div>

      <div className="settings-section">
        <h3 className="settings-section-title">Geral</h3>

        <div className="settings-row">
          <label className="settings-label">Pasta de download</label>
          <input
            className="settings-input"
            type="text"
            value={form.downloadFolder}
            onChange={e => update('downloadFolder', e.target.value)}
            placeholder="/home/user/Downloads"
          />
        </div>

        <div className="settings-row">
          <label className="settings-label">Modo de áudio padrão</label>
          <select
            className="settings-select"
            value={form.defaultMode}
            onChange={e => update('defaultMode', e.target.value)}
          >
            <option value="sub">Legendado</option>
            <option value="dub">Dublado</option>
          </select>
        </div>

        <div className="settings-row">
          <label className="settings-label">Qualidade padrão</label>
          <select
            className="settings-select"
            value={form.defaultQuality}
            onChange={e => update('defaultQuality', e.target.value)}
          >
            <option value="best">Melhor disponível</option>
            <option value="1080p">1080p</option>
            <option value="720p">720p</option>
            <option value="480p">480p</option>
            <option value="worst">Menor qualidade</option>
          </select>
        </div>
      </div>

      <div className="settings-section">
        <h3 className="settings-section-title">Reprodução</h3>

        <div className="settings-row">
          <label className="settings-label">Reproduzir próximo automaticamente</label>
          <label className="settings-toggle">
            <input
              type="checkbox"
              checked={form.autoplayNext}
              onChange={e => update('autoplayNext', e.target.checked)}
            />
            <span className="settings-toggle-slider" />
          </label>
        </div>

        <div className="settings-row">
          <label className="settings-label">Notificações de novos lançamentos</label>
          <label className="settings-toggle">
            <input
              type="checkbox"
              checked={form.notificationsEnabled}
              onChange={e => update('notificationsEnabled', e.target.checked)}
            />
            <span className="settings-toggle-slider" />
          </label>
        </div>

        <div className="settings-row">
          <label className="settings-label">Velocidade de reprodução padrão</label>
          <select
            className="settings-select"
            value={form.playbackSpeed}
            onChange={e => update('playbackSpeed', Number(e.target.value))}
          >
            <option value={0.5}>0.5x</option>
            <option value={0.75}>0.75x</option>
            <option value={1}>1x</option>
            <option value={1.25}>1.25x</option>
            <option value={1.5}>1.5x</option>
            <option value={2}>2x</option>
          </select>
        </div>
      </div>

      <button className="settings-save-btn" onClick={() => onSave(form)}>
        Salvar
      </button>
    </div>
  )
}
