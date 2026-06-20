'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

interface MatchState {
  match_id: string
  team_a: string
  team_b: string
  team_a_code: string
  team_b_code: string
  score_a: number
  score_b: number
  status: string
  minute: number
  competition_title: string
  competition_stage: string
  last_event: string
  last_updated: string
}

const EVENT_EMOJI: Record<string, string> = {
  GOAL: '⚽',
  RED_CARD: '🟥',
  YELLOW_CARD: '🟨',
  VAR_DECISION: '📺',
  CORNER_KICK: '🏁',
  PENALTY: '🎯',
  SUBSTITUTION: '🔄',
  MATCH_STARTED: '🏟️',
  MATCH_ENDED: '🏁',
}

const FLAG: Record<string, string> = {
  BRA: '🇧🇷', MAR: '🇲🇦', GER: '🇩🇪', CUR: '🇨🇼', ARG: '🇦🇷', POL: '🇵🇱',
}

function StatusBadge({ status }: { status: string }) {
  if (status === 'LIVE') return <Badge variant="live" className="gap-1.5"><span className="h-1.5 w-1.5 animate-pulse rounded-full bg-white" />LIVE</Badge>
  if (status === 'ENDED') return <Badge variant="ended">ENCERRADO</Badge>
  return <Badge variant="scheduled">AGENDADO</Badge>
}

export default function LiveMatch({ matchId }: { matchId: string }) {
  const [state, setState] = useState<MatchState | null>(null)
  const [connected, setConnected] = useState(false)
  const [events, setEvents] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)

  const webServerUrl =
    typeof window !== 'undefined'
      ? (process.env.NEXT_PUBLIC_WEB_SERVER_URL || 'http://localhost:8080')
      : 'http://localhost:8080'

  useEffect(() => {
    const es = new EventSource(`${webServerUrl}/matches/${matchId}/stream`)

    es.onopen = () => {
      setConnected(true)
      setError(null)
    }

    es.onmessage = (e) => {
      try {
        const newState: MatchState = JSON.parse(e.data)
        setState(newState)
        if (newState.last_event) {
          const emoji = EVENT_EMOJI[newState.last_event] ?? '📋'
          const msg = `${emoji} ${newState.last_event.replace(/_/g, ' ')} — ${newState.minute}'`
          setEvents((prev) => [msg, ...prev].slice(0, 20))
        }
      } catch {
        // ignore parse errors from heartbeat comments
      }
    }

    es.onerror = () => {
      setConnected(false)
      setError('Connection lost — reconnecting...')
    }

    return () => {
      es.close()
      setConnected(false)
    }
  }, [matchId, webServerUrl])

  if (error && !state) {
    return (
      <Card>
        <CardContent className="py-12 text-center">
          <p className="text-destructive text-sm">{error}</p>
        </CardContent>
      </Card>
    )
  }

  if (!state) {
    return (
      <Card>
        <CardContent className="py-12 text-center">
          <div className="flex items-center justify-center gap-2 text-muted-foreground text-sm">
            <span className="h-2 w-2 animate-pulse rounded-full bg-primary" />
            Conectando ao stream ao vivo...
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      {/* Scoreboard hero */}
      <Card>
        <CardContent className="py-8">
          {/* Competition */}
          <p className="mb-6 text-center text-xs font-medium uppercase tracking-widest text-muted-foreground">
            {state.competition_title} · {state.competition_stage?.replace(/_/g, ' ')}
          </p>

          {/* Teams + Score */}
          <div className="flex items-center justify-center gap-6">
            {/* Team A */}
            <div className="flex flex-1 flex-col items-end gap-1">
              <span className="text-2xl">{FLAG[state.team_a_code] ?? '🏳️'}</span>
              <span className="text-lg font-bold leading-tight text-foreground">{state.team_a}</span>
              <span className="text-xs text-muted-foreground">{state.team_a_code}</span>
            </div>

            {/* Score box */}
            <div className="flex flex-col items-center gap-2">
              <div className="rounded-xl border border-border bg-muted px-6 py-3">
                <span className="text-4xl font-black tracking-widest tabular-nums text-foreground">
                  {state.score_a} : {state.score_b}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <StatusBadge status={state.status} />
                {state.status === 'LIVE' && (
                  <span className="text-xs text-muted-foreground">{state.minute}&apos;</span>
                )}
              </div>
            </div>

            {/* Team B */}
            <div className="flex flex-1 flex-col items-start gap-1">
              <span className="text-2xl">{FLAG[state.team_b_code] ?? '🏳️'}</span>
              <span className="text-lg font-bold leading-tight text-foreground">{state.team_b}</span>
              <span className="text-xs text-muted-foreground">{state.team_b_code}</span>
            </div>
          </div>

          {/* Connection indicator */}
          <div className="mt-6 flex justify-center">
            <span className={cn(
              'inline-flex items-center gap-1.5 text-xs',
              connected ? 'text-emerald-400' : 'text-destructive'
            )}>
              <span className={cn(
                'h-1.5 w-1.5 rounded-full',
                connected ? 'bg-emerald-500 animate-pulse' : 'bg-destructive'
              )} />
              {connected ? 'CONECTADO' : 'RECONECTANDO...'}
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Last event highlight */}
      {state.last_event && (
        <Card className="border-primary/20 bg-primary/5">
          <CardContent className="flex items-center gap-3 py-4">
            <span className="text-2xl">{EVENT_EMOJI[state.last_event] ?? '📋'}</span>
            <div>
              <p className="text-sm font-semibold text-foreground">
                {state.last_event.replace(/_/g, ' ')}
              </p>
              <p className="text-xs text-muted-foreground">
                {new Date(state.last_updated).toLocaleTimeString('pt-BR')}
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Event feed */}
      {events.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
              Feed ao vivo
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-0">
            <div className="space-y-1">
              {events.map((e, i) => (
                <div
                  key={i}
                  className={cn(
                    'rounded-lg px-3 py-2 text-sm transition-colors',
                    i === 0
                      ? 'bg-accent text-foreground font-medium'
                      : 'text-muted-foreground'
                  )}
                >
                  {e}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
