import Link from 'next/link'
import LiveMatch from '@/components/LiveMatch'

interface Props {
  params: { id: string }
}

export default function MatchPage({ params }: Props) {
  const matchId = decodeURIComponent(params.id)

  return (
    <main className="container">
      <div style={{ marginBottom: 20 }}>
        <Link href="/" style={{ color: '#888', fontSize: '0.9rem' }}>
          ← Back to matches
        </Link>
      </div>
      <h1 style={{ fontSize: '1.2rem', marginBottom: 24, color: '#888' }}>
        Match: <span style={{ color: '#fff' }}>{matchId}</span>
      </h1>
      <LiveMatch matchId={matchId} />
      <div style={{ marginTop: 32, display: 'flex', gap: 12 }}>
        <a
          href={`${process.env.NEXT_PUBLIC_WEB_SERVER_URL || 'http://localhost:8080'}/matches/${matchId}/statistic`}
          target="_blank"
          rel="noreferrer"
          style={{
            background: '#1a1a2e',
            border: '1px solid #333',
            borderRadius: 6,
            padding: '8px 16px',
            fontSize: '0.85rem',
            color: '#ccc',
          }}
        >
          View Statistics (JSON)
        </a>
      </div>
    </main>
  )
}
