import type { GrpcService, GrpcMethod, GrpcMessage } from '../api/client'
import { ConsumerBadges } from './ConsumerBadges'

interface Props {
  service: GrpcService
}

export function GrpcServiceView({ service }: Props) {
  return (
    <div className="grpc-service">
      <h3 className="grpc-service-name">{service.name}</h3>
      <div className="methods">
        {service.methods.map((method) => (
          <MethodView key={method.name} method={method} />
        ))}
      </div>
    </div>
  )
}

function MethodView({ method }: { method: GrpcMethod }) {
  return (
    <div className="method">
      <div className="method-header">
        <span className="method-badge">rpc</span>
        <span className="method-name">{method.name}</span>
        <ConsumerBadges consumers={method.consumers} />
      </div>
      <div className="method-body">
        <MessageView label="Request" message={method.input} />
        <MessageView label="Response" message={method.output} />
      </div>
    </div>
  )
}

function MessageView({ label, message }: { label: string; message: GrpcMessage }) {
  return (
    <div className="message">
      <div className="message-header">
        <span className="message-label">{label}:</span>
        <span className="message-name">{message.name}</span>
      </div>
      {message.fields.length > 0 && (
        <table className="fields-table">
          <thead>
            <tr>
              <th>#</th>
              <th>Field</th>
              <th>Type</th>
              <th>Consumers</th>
            </tr>
          </thead>
          <tbody>
            {message.fields.map((field) => (
              <tr key={field.number}>
                <td className="field-number">{field.number}</td>
                <td className="field-name">{field.name}</td>
                <td className="field-type">{field.type}</td>
                <td><ConsumerBadges consumers={field.consumers} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
