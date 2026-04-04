import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { listServices } from '../api/client'

export function ServicesPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['services'],
    queryFn: listServices,
  })

  if (isLoading) return <div className="loading">Loading services...</div>
  if (error) return <div className="error">Failed to load services: {String(error)}</div>

  const services = data?.services ?? []

  return (
    <div className="page">
      <h2>Services</h2>
      {services.length === 0 ? (
        <p className="empty">No services registered yet.</p>
      ) : (
        <ul className="service-list">
          {services.map((svc) => (
            <li key={svc.id} className="service-item">
              <Link to={`/services/${svc.name}`} className="service-link">
                <span className="service-name">{svc.name}</span>
                <span className="service-arrow">&rarr;</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
