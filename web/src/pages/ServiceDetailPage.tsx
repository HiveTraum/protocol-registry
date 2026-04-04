import { useQuery } from '@tanstack/react-query'
import { useParams, Link } from 'react-router-dom'
import { getGrpcView } from '../api/client'
import { GrpcServiceView } from '../components/GrpcServiceView'

export function ServiceDetailPage() {
  const { name } = useParams<{ name: string }>()

  const { data, isLoading, error } = useQuery({
    queryKey: ['grpc-view', name],
    queryFn: () => getGrpcView(name!),
    enabled: !!name,
  })

  if (isLoading) return <div className="loading">Loading gRPC view...</div>
  if (error) return <div className="error">Failed to load service: {String(error)}</div>
  if (!data) return null

  return (
    <div className="page">
      <Link to="/services" className="back-link">&larr; All services</Link>
      <h2>{data.serviceName}</h2>

      {data.services.length === 0 ? (
        <p className="empty">No gRPC services found.</p>
      ) : (
        data.services.map((svc) => (
          <GrpcServiceView key={svc.name} service={svc} />
        ))
      )}
    </div>
  )
}
