import { Routes, Route, Navigate } from 'react-router-dom'
import { ServicesPage } from './pages/ServicesPage'
import { ServiceDetailPage } from './pages/ServiceDetailPage'

function App() {
  return (
    <div className="app">
      <header className="header">
        <h1>Protocol Registry</h1>
      </header>
      <main className="main">
        <Routes>
          <Route path="/" element={<Navigate to="/services" replace />} />
          <Route path="/services" element={<ServicesPage />} />
          <Route path="/services/:name" element={<ServiceDetailPage />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
