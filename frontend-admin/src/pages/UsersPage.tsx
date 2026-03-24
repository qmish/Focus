import { useEffect, useState } from 'react'
import { useAdminStore } from '../store/adminStore'

interface User {
  id: string
  email: string
  name: string
  roles: string[]
  is_active: boolean
  created_at: string
}

export default function UsersPage() {
  const { users, loading, fetchUsers, updateUserRoles, banUser, unbanUser } = useAdminStore()
  const [page, setPage] = useState(1)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [showModal, setShowModal] = useState(false)

  useEffect(() => {
    fetchUsers(page)
  }, [page])

  const handleRoleChange = (userId: string, roles: string[]) => {
    updateUserRoles(userId, roles)
    setShowModal(false)
  }

  const handleBan = (userId: string) => {
    if (confirm('Заблокировать пользователя?')) {
      banUser(userId, 'Нарушение правил')
    }
  }

  const handleUnban = (userId: string) => {
    unbanUser(userId)
  }

  if (loading) {
    return <div className="loading">Загрузка...</div>
  }

  return (
    <div className="users-page">
      <h1>Пользователи</h1>

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
                  <button onClick={() => { setSelectedUser(user); setShowModal(true); }}>
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
        <span>Страница {page}</span>
        <button onClick={() => setPage(p => p + 1)}>Вперёд →</button>
      </div>

      {showModal && selectedUser && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h3>Роли пользователя</h3>
            <p>{selectedUser.name} ({selectedUser.email})</p>
            
            <div className="role-selection">
              <label>
                <input type="checkbox" defaultChecked={selectedUser.roles.includes('user')} />
                Пользователь
              </label>
              <label>
                <input type="checkbox" defaultChecked={selectedUser.roles.includes('moderator')} />
                Модератор
              </label>
              <label>
                <input type="checkbox" defaultChecked={selectedUser.roles.includes('admin')} />
                Администратор
              </label>
            </div>

            <div className="modal-actions">
              <button onClick={() => setShowModal(false)}>Отмена</button>
              <button onClick={() => {
                const roles = Array.from(document.querySelectorAll('input[type=checkbox]:checked'))
                  .map(el => (el.parentElement as HTMLElement).textContent?.trim() || '')
                  .map(r => r === 'Пользователь' ? 'user' : r === 'Модератор' ? 'moderator' : 'admin')
                handleRoleChange(selectedUser.id, roles)
              }}>
                Сохранить
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
