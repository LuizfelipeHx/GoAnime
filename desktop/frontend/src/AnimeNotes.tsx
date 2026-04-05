import { useState } from 'react'
import type { AnimeNote } from './lib/backend'

interface AnimeNotesProps {
  title: string
  note: AnimeNote | null
  onSave: (note: AnimeNote) => void
  onClose: () => void
}

export default function AnimeNotes({ title, note, onSave, onClose }: AnimeNotesProps) {
  const [rating, setRating] = useState(note?.rating ?? 0)
  const [text, setText] = useState(note?.note ?? '')

  const handleSave = () => {
    onSave({
      title,
      note: text,
      rating,
      updatedAt: new Date().toISOString(),
    })
  }

  return (
    <div className="notes-overlay" onClick={onClose}>
      <div className="notes-modal" onClick={(e) => e.stopPropagation()}>
        <h3 className="notes-title">{title}</h3>

        <div className="notes-rating">
          {Array.from({ length: 10 }, (_, i) => i + 1).map((star) => (
            <button
              key={star}
              className={`notes-star ${star <= rating ? 'notes-star--filled' : ''}`}
              onClick={() => setRating(star === rating ? 0 : star)}
              type="button"
              title={`${star}/10`}
            >
              &#9733;
            </button>
          ))}
          <span className="notes-rating-label">
            {rating > 0 ? `${rating}/10` : 'Sem nota'}
          </span>
        </div>

        <textarea
          className="notes-textarea"
          placeholder="Escreva suas anotacoes sobre este anime..."
          value={text}
          onChange={(e) => setText(e.target.value)}
          rows={6}
        />

        <div className="notes-actions">
          <button className="notes-btn notes-btn--save" onClick={handleSave}>
            Salvar
          </button>
          <button className="notes-btn notes-btn--cancel" onClick={onClose}>
            Cancelar
          </button>
        </div>
      </div>
    </div>
  )
}
