interface Props {
  consumers: string[]
}

export function ConsumerBadges({ consumers }: Props) {
  if (!consumers || consumers.length === 0) return null

  return (
    <span className="consumer-badges">
      {consumers.map((c) => (
        <span key={c} className="consumer-badge">{c}</span>
      ))}
    </span>
  )
}
