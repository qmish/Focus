import { useAuthStore } from '../store/authStore'

export default function ProfilePage() {
  const { user } = useAuthStore()

  return (
    <div className="profile-page">
      <h2>Профиль</h2>
      
      <div className="profile-card">
        <div className="profile-avatar">
          {user?.name?.charAt(0) || 'U'}
        </div>
        
        <div className="profile-info">
          <div className="profile-field">
            <label>Имя</label>
            <p>{user?.name || 'Не указано'}</p>
          </div>
          
          <div className="profile-field">
            <label>Email</label>
            <p>{user?.email || 'Не указано'}</p>
          </div>
          
          <div className="profile-field">
            <label>ID</label>
            <p className="profile-id">{user?.id}</p>
          </div>
          
          {user?.roles && user.roles.length > 0 && (
            <div className="profile-field">
              <label>Роли</label>
              <div className="profile-roles">
                {user.roles.map(role => (
                  <span key={role} className="role-badge">{role}</span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
