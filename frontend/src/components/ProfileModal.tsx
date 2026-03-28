import { useState, useEffect } from 'react'
import { apiClient } from '../lib/apiClient'

interface ProfileData {
  id: string
  email: string
  name: string
  roles?: string[]
  department?: string
  directorate?: string
  position?: string
  phone?: string
  about_me?: string
  video_start_with_audio_muted?: boolean
  video_start_with_video_muted?: boolean
  video_display_name?: string
  video_default_language?: string
}

interface ProfileModalProps {
  open: boolean
  onClose: () => void
  user: ProfileData | null
  onSave: (updated: ProfileData) => void
}

export default function ProfileModal({ open, onClose, user, onSave }: ProfileModalProps) {
  const [form, setForm] = useState({
    name: '',
    directorate: '',
    department: '',
    position: '',
    phone: '',
    about_me: '',
    video_start_with_audio_muted: false,
    video_start_with_video_muted: false,
    video_display_name: '',
    video_default_language: 'ru',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (user && open) {
      setForm({
        name: user.name || '',
        directorate: user.directorate || '',
        department: user.department || '',
        position: user.position || '',
        phone: user.phone || '',
        about_me: user.about_me || '',
        video_start_with_audio_muted: user.video_start_with_audio_muted || false,
        video_start_with_video_muted: user.video_start_with_video_muted || false,
        video_display_name: user.video_display_name || '',
        video_default_language: user.video_default_language || 'ru',
      })
      setError('')
    }
  }, [user, open])

  const handleSave = async () => {
    setSaving(true)
    setError('')
    try {
      const updated = await apiClient.put<ProfileData>('/api/v1/auth/profile', form)
      onSave(updated)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка сохранения профиля')
    } finally {
      setSaving(false)
    }
  }

  if (!open) return null

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal profile-modal" role="dialog" aria-modal="true" onClick={e => e.stopPropagation()} style={{ maxWidth: 520, maxHeight: '90vh', overflow: 'auto' }}>
        <div className="modal-header">
          <h3>Настройки профиля</h3>
          <button className="icon-btn" onClick={onClose}>✕</button>
        </div>

        {error && <div style={{ color: '#ef4444', padding: '0 1.5rem', fontSize: '0.875rem' }}>{error}</div>}

        <div style={{ padding: '0 1.5rem 1rem' }}>
          <div className="profile-modal-section">
            <h4 style={{ margin: '0.75rem 0 0.5rem', fontSize: '0.95rem', color: '#94a3b8' }}>Личные данные</h4>

            <div className="form-group">
              <label>ФИО</label>
              <input type="text" value={form.name} onChange={e => setForm(p => ({ ...p, name: e.target.value }))} placeholder="Иванов Иван Иванович" />
            </div>

            <div className="form-group">
              <label>Почта</label>
              <input type="email" value={user?.email || ''} disabled style={{ opacity: 0.6 }} />
            </div>

            <div className="form-group">
              <label>Дирекция</label>
              <input type="text" value={form.directorate} onChange={e => setForm(p => ({ ...p, directorate: e.target.value }))} placeholder="IT-дирекция" />
            </div>

            <div className="form-group">
              <label>Отдел</label>
              <input type="text" value={form.department} onChange={e => setForm(p => ({ ...p, department: e.target.value }))} placeholder="Отдел разработки" />
            </div>

            <div className="form-group">
              <label>Должность</label>
              <input type="text" value={form.position} onChange={e => setForm(p => ({ ...p, position: e.target.value }))} placeholder="Ведущий разработчик" />
            </div>

            <div className="form-group">
              <label>Телефон</label>
              <input type="tel" value={form.phone} onChange={e => setForm(p => ({ ...p, phone: e.target.value }))} placeholder="+7 (999) 123-45-67" />
            </div>

            <div className="form-group">
              <label>О себе</label>
              <textarea value={form.about_me} onChange={e => setForm(p => ({ ...p, about_me: e.target.value }))} placeholder="Расскажите о себе..." rows={3} style={{ resize: 'vertical' }} />
            </div>
          </div>

          <div className="profile-modal-section">
            <h4 style={{ margin: '1rem 0 0.5rem', fontSize: '0.95rem', color: '#94a3b8' }}>Настройки видеоконференции</h4>

            <div className="form-group" style={{ flexDirection: 'row', alignItems: 'center', gap: '0.5rem' }}>
              <input type="checkbox" id="audioMuted" checked={form.video_start_with_audio_muted} onChange={e => setForm(p => ({ ...p, video_start_with_audio_muted: e.target.checked }))} />
              <label htmlFor="audioMuted" style={{ marginBottom: 0 }}>Начинать с выключенным микрофоном</label>
            </div>

            <div className="form-group" style={{ flexDirection: 'row', alignItems: 'center', gap: '0.5rem' }}>
              <input type="checkbox" id="videoMuted" checked={form.video_start_with_video_muted} onChange={e => setForm(p => ({ ...p, video_start_with_video_muted: e.target.checked }))} />
              <label htmlFor="videoMuted" style={{ marginBottom: 0 }}>Начинать с выключенной камерой</label>
            </div>

            <div className="form-group">
              <label>Отображаемое имя в конференции</label>
              <input type="text" value={form.video_display_name} onChange={e => setForm(p => ({ ...p, video_display_name: e.target.value }))} placeholder="Иван Иванов" />
            </div>

            <div className="form-group">
              <label>Язык по умолчанию</label>
              <select value={form.video_default_language} onChange={e => setForm(p => ({ ...p, video_default_language: e.target.value }))}>
                <option value="ru">Русский</option>
                <option value="en">English</option>
              </select>
            </div>
          </div>
        </div>

        <div className="modal-actions" style={{ padding: '0 1.5rem 1.5rem' }}>
          <button className="btn-secondary" onClick={onClose}>Отмена</button>
          <button className="btn-primary" onClick={handleSave} disabled={saving}>
            {saving ? 'Сохранение...' : 'Сохранить'}
          </button>
        </div>
      </div>
    </div>
  )
}
