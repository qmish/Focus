import { useEffect, useState } from 'react'
import { useAdminStore, type User } from '../store/adminStore'
import ConfirmDialog from '../components/ConfirmDialog'

export default function UsersPage() {
  const {
    users,
    invites,
    pagination,
    error,
    loading,
    fetchUsers,
    fetchInvites,
    createUser,
    patchUser,
    deleteUser,
    updateUserRoles,
    banUser,
    unbanUser,
    createInvite,
    resendInvite,
  } = useAdminStore()
  const [page, setPage] = useState(1)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [selectedRoles, setSelectedRoles] = useState<string[]>([])
  const [showModal, setShowModal] = useState(false)
  const [newName, setNewName] = useState('')
  const [newEmail, setNewEmail] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [inviteEmail, setInviteEmail] = useState('')
  const [lastInviteUrl, setLastInviteUrl] = useState<string | null>(null)
  const [confirmDialog, setConfirmDialog] = useState<{
    open: boolean
    title: string
    message: string
    confirmLabel?: string
    inputMode?: boolean
    inputValue: string
    action: string
    userId: string
  }>({ open: false, title: '', message: '', inputValue: '', action: '', userId: '' })

  useEffect(() => {
    void fetchUsers(page)
    void fetchInvites()
  }, [page])

  const handleRoleChange = (userId: string, roles: string[]) => {
    void updateUserRoles(userId, roles)
    setShowModal(false)
  }

  const handleBan = (userId: string) => {
    setConfirmDialog({
      open: true,
      title: 'Блокировка',
      message: 'Заблокировать пользователя?',
      confirmLabel: 'Далее',
      action: 'ban-confirm',
      userId,
      inputMode: false,
      inputValue: '',
    })
  }

  const handleUnban = (userId: string) => {
    void unbanUser(userId)
  }

  const openRolesModal = (user: User) => {
    setSelectedUser(user)
    setSelectedRoles(user.roles)
    setShowModal(true)
  }

  const toggleRole = (role: string) => {
    setSelectedRoles((prev) => (
      prev.includes(role) ? prev.filter((item) => item !== role) : [...prev, role]
    ))
  }

  const handleCreateUser = () => {
    if (!newName.trim() || !newEmail.trim()) return
    void createUser({
      name: newName.trim(),
      email: newEmail.trim(),
      password: newPassword.trim() || undefined,
      roles: ['user'],
      is_active: true,
    })
    setNewName('')
    setNewEmail('')
    setNewPassword('')
  }

  const handleEditUser = (user: User) => {
    setConfirmDialog({
      open: true,
      title: 'Изменить имя',
      message: 'Введите новое имя пользователя:',
      confirmLabel: 'Сохранить',
      action: 'edit',
      userId: user.id,
      inputMode: true,
      inputValue: user.name,
    })
  }

  const handleDeleteUser = (userId: string) => {
    setConfirmDialog({
      open: true,
      title: 'Деактивация',
      message: 'Деактивировать пользователя?',
      confirmLabel: 'Деактивировать',
      action: 'delete',
      userId,
      inputMode: false,
      inputValue: '',
    })
  }

  const handleDialogConfirm = () => {
    const { action, userId, inputValue } = confirmDialog
    setConfirmDialog(prev => ({ ...prev, open: false }))

    if (action === 'ban-confirm') {
      setConfirmDialog({
        open: true,
        title: 'Причина блокировки',
        message: 'Укажите причину блокировки:',
        confirmLabel: 'Заблокировать',
        action: 'ban-reason',
        userId,
        inputMode: true,
        inputValue: 'Нарушение правил',
      })
    } else if (action === 'ban-reason') {
      void banUser(userId, inputValue || 'Нарушение правил', 0)
    } else if (action === 'edit') {
      if (inputValue.trim()) {
        void patchUser(userId, { name: inputValue.trim() })
      }
    } else if (action === 'delete') {
      void deleteUser(userId)
    }
  }

  const handleDialogCancel = () => {
    setConfirmDialog(prev => ({ ...prev, open: false }))
  }

  const handleCreateInvite = async () => {
    if (!inviteEmail.trim()) return
    const inviteUrl = await createInvite({ email: inviteEmail.trim(), roles: ['user'], expires_in_hours: 72 })
    setLastInviteUrl(inviteUrl)
    setInviteEmail('')
  }

  const handleResendInvite = async (inviteId: string) => {
    const inviteUrl = await resendInvite(inviteId)
    setLastInviteUrl(inviteUrl)
  }

  if (loading) {
    return <div className="loading">Загрузка...</div>
  }

  return (
    <div className="users-page">
      <h1>Пользователи</h1>
      {error && <p className="error">{error}</p>}

      <div className="settings-section">
        <h2>Создать пользователя</h2>
        <div className="form-group">
          <label>Имя</label>
          <input value={newName} onChange={(e) => setNewName(e.target.value)} aria-label="Имя" />
        </div>
        <div className="form-group">
          <label>Email</label>
          <input value={newEmail} onChange={(e) => setNewEmail(e.target.value)} aria-label="Email" />
        </div>
        <div className="form-group">
          <label>Пароль (опционально)</label>
          <input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} aria-label="Пароль" />
        </div>
        <button className="primary" onClick={handleCreateUser}>Создать</button>
      </div>

      <div className="users-table">
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Имя</th>
              <th>Email</th>
              <th>Роли</th>
              <th>Статус</th>
              <th>Действия</th>
            </tr>
          </thead>
          <tbody>
            {users.map(user => (
              <tr key={user.id}>
                <td>{user.id.slice(0, 8)}...</td>
                <td>{user.name}</td>
                <td>{user.email}</td>
                <td>
                  {user.roles.map(role => (
                    <span key={role} className="role-badge">{role}</span>
                  ))}
                </td>
                <td>
                  <span className={`status-badge ${user.is_active ? 'active' : 'inactive'}`}>
                    {user.is_active ? 'Активен' : 'Заблокирован'}
                  </span>
                </td>
                <td>
                  <button onClick={() => openRolesModal(user)}>
                    Роли
                  </button>
                  <button onClick={() => handleEditUser(user)}>
                    Изменить
                  </button>
                  <button onClick={() => handleDeleteUser(user.id)} className="danger">
                    Удалить
                  </button>
                  {user.is_active ? (
                    <button onClick={() => handleBan(user.id)} className="danger">
                      Заблокировать
                    </button>
                  ) : (
                    <button onClick={() => handleUnban(user.id)} className="success">
                      Разблокировать
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="pagination">
        <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}>
          ← Назад
        </button>
        <span>Страница {pagination.page} из {pagination.total_pages}</span>
        <button
          onClick={() => setPage(p => p + 1)}
          disabled={pagination.page >= pagination.total_pages}
        >
          Вперёд →
        </button>
      </div>

      <div className="settings-section">
        <h2>Инвайты</h2>
        <div className="form-group">
          <label>Email для инвайта</label>
          <input value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} aria-label="Email для инвайта" />
        </div>
        <button className="primary" onClick={() => void handleCreateInvite()}>
          Отправить инвайт
        </button>
        {lastInviteUrl && (
          <p style={{ marginTop: 12 }}>
            Ссылка инвайта: <a href={lastInviteUrl} target="_blank" rel="noreferrer">{lastInviteUrl}</a>
          </p>
        )}
        <div style={{ marginTop: 16 }}>
          {invites.map((invite) => (
            <div key={invite.id} style={{ display: 'flex', gap: 8, marginBottom: 8, alignItems: 'center' }}>
              <span>{invite.email}</span>
              <span className="status-badge">{invite.status}</span>
              <button onClick={() => void handleResendInvite(invite.id)}>Повторно отправить</button>
            </div>
          ))}
        </div>
      </div>

      <ConfirmDialog
        open={confirmDialog.open}
        title={confirmDialog.title}
        message={confirmDialog.message}
        confirmLabel={confirmDialog.confirmLabel}
        onConfirm={handleDialogConfirm}
        onCancel={handleDialogCancel}
        inputMode={confirmDialog.inputMode}
        inputValue={confirmDialog.inputValue}
        onInputChange={(val) => setConfirmDialog(prev => ({ ...prev, inputValue: val }))}
      />

      {showModal && selectedUser && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h3>Роли пользователя</h3>
            <p>{selectedUser.name} ({selectedUser.email})</p>
            
            <div className="role-selection">
              <label>
                <input
                  type="checkbox"
                  checked={selectedRoles.includes('user')}
                  onChange={() => toggleRole('user')}
                />
                Пользователь
              </label>
              <label>
                <input
                  type="checkbox"
                  checked={selectedRoles.includes('moderator')}
                  onChange={() => toggleRole('moderator')}
                />
                Модератор
              </label>
              <label>
                <input
                  type="checkbox"
                  checked={selectedRoles.includes('admin')}
                  onChange={() => toggleRole('admin')}
                />
                Администратор
              </label>
            </div>

            <div className="modal-actions">
              <button onClick={() => setShowModal(false)}>Отмена</button>
              <button onClick={() => handleRoleChange(selectedUser.id, selectedRoles)}>
                Сохранить
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
