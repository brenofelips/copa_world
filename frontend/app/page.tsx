import Link from 'next/link'

// Demo matches — in production this would come from GET /matches (a list endpoint)
const DEMO_MATCHES = [
  { id: 'BRA-MAR-13-06-2026', teamA: 'Brazil', teamB: 'Morocco', competition: 'FIFA World Cup 2026' },
  { id: 'GER-CUR-14-06-2026', teamA: 'Germany', teamB: 'Curaçao',  competition: 'FIFA World Cup 2026' },
  { id: 'ARG-POL-15-06-2026', teamA: 'Argentina', teamB: 'Poland', competition: 'FIFA World Cup 2026' },
]

export default function HomePage() {
  return (
    <main className="container">
      <h1>Copa World – Live Scores</h1>
      <div style={{ display: 'grid', gap: 12 }}>
        {DEMO_MATCHES.map((m) => (
          <Link key={m.id} href={`/matches/${m.id}`}>
            <div style={{
              background: '#1a1a2e',
              border: '1px solid #333',
              borderRadius: 8,
              padding: '20px 24px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              cursor: 'pointer',
              transition: 'border-color 0.2s',
            }}>
              <div>
                <div style={{ fontWeight: 700, fontSize: '1.1rem' }}>
                  {m.teamA} vs {m.teamB}
                </div>
                <div style={{ color: '#888', fontSize: '0.85rem', marginTop: 4 }}>
                  {m.competition}
                </div>
              </div>
              <div style={{
                background: '#e63946',
                color: '#fff',
                borderRadius: 4,
                padding: '4px 10px',
                fontSize: '0.75rem',
                fontWeight: 700,
                letterSpacing: 1,
              }}>
                LIVE
              </div>
            </div>
          </Link>
        ))}
      </div>
      <p style={{ marginTop: 32, color: '#555', fontSize: '0.85rem' }}>
        Architecture: Data Provider → Nginx LB → Ingestion API → Kafka → Score Service + Event Service → Redis + PostgreSQL → Web Server → SSE
      </p>
    </main>
  )
}
