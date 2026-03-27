import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { AdminUiProvider } from './providers/AdminUiProvider'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AdminUiProvider>
      <App />
    </AdminUiProvider>
  </React.StrictMode>,
)
