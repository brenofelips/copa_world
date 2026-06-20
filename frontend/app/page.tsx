'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

interface MatchState {
  match_id: string
  team_a: string
  team_b: string
  team_a_code: string
  team_b_code: string
  team_a_logo: string
  team_b_logo: string
  score_a: number
  score_b: number
  status: string
  competition_title: string
  competition_stage: string
}

const POLL_INTERVAL = 30_000

function MatchCard({ m }: { m: MatchState }) {
  return (
    <Link key={m.match_id} href={`/matches/${m.match_id}`}>
      <Card className="group cursor-pointer transition-all duration-200 hover:border-primary/40 hover:bg-accent">
        <CardContent className="p-0">
          <div className="flex items-center justify-between px-5 py-4">
            <div className="hidden min-w-[140px] sm:block">
              <p className="text-xs font-medium text-muted-foreground">{m.competition_title}</p>
              <p className="mt-0.5 text-xs text-muted-foreground/60">{m.competition_stage?.replace(/_/g, ' ')}</p>
            </div>

            <div className="flex flex-1 items-center justify-center gap-4 sm:flex-none">
              <div className="flex items-center gap-2">
                {m.team_a_logo && <img src={m.team_a_logo} alt={m.team_a} className="h-7 w-7 object-contain" />}
                <span className="text-base font-semibold text-foreground">{m.team_a}</span>
              </div>
              {m.status === 'SCHEDULED' ? (
                <span className="text-xs font-bold text-muted-foreground">vs</span>
              ) : (
                <span className={`font-black tabular-nums text-foreground ${m.status === 'LIVE' ? 'text-lg' : 'text-base text-muted-foreground'}`}>
                  {m.score_a} : {m.score_b}
                </span>
              )}
              <div className="flex items-center gap-2">
                <span className="text-base font-semibold text-foreground">{m.team_b}</span>
                {m.team_b_logo && <img src={m.team_b_logo} alt={m.team_b} className="h-7 w-7 object-contain" />}
              </div>
            </div>

            <div className="flex min-w-[80px] justify-end">
              {m.status === 'LIVE' && (
                <Badge variant="live" className="gap-1.5">
                  <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-white" />
                  LIVE
                </Badge>
              )}
              {m.status === 'ENDED' && (
                <Badge variant="ended">ENCERRADO</Badge>
              )}
              {m.status === 'SCHEDULED' && (
                <Badge variant="scheduled">AGENDADO</Badge>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}

function Section({ title, matches }: { title: string; matches: MatchState[] }) {
  if (matches.length === 0) return null
  return (
    <div className="space-y-2">
      <h2 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">{title}</h2>
      <div className="grid gap-3">
        {matches.map((m) => <MatchCard key={m.match_id} m={m} />)}
      </div>
    </div>
  )
}

export default function HomePage() {
  const [matches, setMatches] = useState<MatchState[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const webServerUrl = process.env.NEXT_PUBLIC_WEB_SERVER_URL || 'http://localhost:8080'

  useEffect(() => {
    let cancelled = false

    async function fetchMatches() {
      try {
        const res = await fetch(`${webServerUrl}/matches`)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const data: MatchState[] = await res.json()
        if (!cancelled) {
          setMatches(data)
          setError(null)
        }
      } catch (e) {
        if (!cancelled) setError('Não foi possível carregar as partidas.')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    fetchMatches()
    const timer = setInterval(fetchMatches, POLL_INTERVAL)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [webServerUrl])

  const live = matches.filter((m) => m.status === 'LIVE')
  const scheduled = matches.filter((m) => m.status === 'SCHEDULED')
  const ended = matches.filter((m) => m.status === 'ENDED')

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
          Partidas
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Placar em tempo real via SSE · Go + Kafka + Redis
        </p>
      </div>

      {loading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <span className="h-2 w-2 animate-pulse rounded-full bg-primary" />
          Carregando partidas...
        </div>
      )}

      {error && !loading && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      {!loading && !error && matches.length === 0 && (
        <p className="text-sm text-muted-foreground">Nenhuma partida disponível no momento.</p>
      )}

      {!loading && !error && matches.length > 0 && (
        <div className="space-y-8">
          <Section title="Ao vivo" matches={live} />
          <Section title="Em breve" matches={scheduled} />
          <Section title="Encerradas" matches={ended} />
        </div>
      )}

      <p className="text-center text-xs text-muted-foreground/50">
        Data Provider → Nginx LB → Ingestion API → Kafka → Score Service + Event Service → Redis + PostgreSQL → Web Server → SSE
      </p>
    </div>
  )
}
