import { Link } from 'react-router-dom'

export default function NotFoundPage() {
  return (
    <div className="not-found-page">
      <h1>404</h1>
      <p>Страница не найдена</p>
      <Link to="/rooms">Вернуться к комнатам</Link>
    </div>
  )
}
