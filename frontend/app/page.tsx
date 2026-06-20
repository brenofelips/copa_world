import Link from 'next/link'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

const DEMO_MATCHES = [
  { id: 'BRA-MAR-13-06-2026', teamA: 'Brazil', codeA: 'BRA', teamB: 'Morocco', codeB: 'MAR', competition: 'FIFA World Cup 2026', stage: 'Group Stage', status: 'LIVE' as const },
  { id: 'GER-CUR-14-06-2026', teamA: 'Germany', codeA: 'GER', teamB: 'Curaçao', codeB: 'CUR', competition: 'FIFA World Cup 2026', stage: 'Group Stage', status: 'LIVE' as const },
  { id: 'ARG-POL-15-06-2026', teamA: 'Argentina', codeA: 'ARG', teamB: 'Poland', codeB: 'POL', competition: 'FIFA World Cup 2026', stage: 'Group Stage', status: 'LIVE' as const },
]

const FLAG: Record<string, string> = {
  BRA: '🇧🇷', MAR: '🇲🇦', GER: '🇩🇪', CUR: '🇨🇼', ARG: '🇦🇷', POL: '🇵🇱',
}

export default function HomePage() {
  return (
    <div className="space-y-8">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
          Partidas ao vivo
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Placar em tempo real via SSE · Go + Kafka + Redis
        </p>
      </div>

      {/* Match cards */}
      <div className="grid gap-3">
        {DEMO_MATCHES.map((m) => (
          <Link key={m.id} href={`/matches/${m.id}`}>
            <Card className="group cursor-pointer transition-all duration-200 hover:border-primary/40 hover:bg-accent">
              <CardContent className="p-0">
                <div className="flex items-center justify-between px-5 py-4">
                  {/* Competition */}
                  <div className="hidden min-w-[140px] sm:block">
                    <p className="text-xs font-medium text-muted-foreground">{m.competition}</p>
                    <p className="mt-0.5 text-xs text-muted-foreground/60">{m.stage}</p>
                  </div>

                  {/* Teams */}
                  <div className="flex flex-1 items-center justify-center gap-4 sm:flex-none">
                    <div className="flex items-center gap-2 text-right">
                      <span className="text-base font-semibold text-foreground">{m.teamA}</span>
                      <span className="text-xl">{FLAG[m.codeA] ?? '🏳️'}</span>
                    </div>
                    <span className="text-xs font-bold text-muted-foreground">vs</span>
                    <div className="flex items-center gap-2">
                      <span className="text-xl">{FLAG[m.codeB] ?? '🏳️'}</span>
                      <span className="text-base font-semibold text-foreground">{m.teamB}</span>
                    </div>
                  </div>

                  {/* Status badge */}
                  <div className="flex min-w-[80px] justify-end">
                    {m.status === 'LIVE' && (
                      <Badge variant="live" className="gap-1.5">
                        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-white" />
                        LIVE
                      </Badge>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>

      {/* Footer note */}
      <p className="text-center text-xs text-muted-foreground/50">
        Data Provider → Nginx LB → Ingestion API → Kafka → Score Service + Event Service → Redis + PostgreSQL → Web Server → SSE
      </p>
    </div>
  )
}
