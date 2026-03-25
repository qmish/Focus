import { useEffect, useState } from 'react'
import { useAdminStore, type User } from '../store/adminStore'

export default function UsersPage() {
  const { users, pagination, error, loading, fetchUsers, updateUserRoles, banUser, unbanUser } = useAdminStore()
  const [page, setPage] = useState(1)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [selectedRoles, setSelectedRoles] = useState<string[]>([])
  const [showModal, setShowModal] = useState(false)

  useEffect(() => {
    fetchUsers(page)
  }, [page])

  const handleRoleChange = (userId: string, roles: string[]) => {
    void updateUserRoles(userId, roles)
    setShowModal(false)
  }

  const handleBan = (userId: string) => {
    if (confirm('Заблокировать пользователя?')) {
      void banUser(userId, 'Нарушение правил')
    }
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

  if (loading) {
    return <div className="loading">Загрузка...</div>
  }

  return (
    <div className="users-page">
      <h1>Пользователи</h1>
      {error && <p className="error">{error}</p>}

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
