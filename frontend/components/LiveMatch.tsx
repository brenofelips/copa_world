'use client'

import { useEffect, useState } from 'react'

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

const STATUS_COLOR: Record<string, string> = {
  LIVE: '#e63946',
  ENDED: '#2a9d8f',
  SCHEDULED: '#457b9d',
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
      <div style={{ color: '#e63946', padding: 32, textAlign: 'center' }}>
        {error}
      </div>
    )
  }

  if (!state) {
    return (
      <div style={{ color: '#888', padding: 32, textAlign: 'center' }}>
        Connecting to live stream...
      </div>
    )
  }

  const statusColor = STATUS_COLOR[state.status] ?? '#888'

  return (
    <div style={{ fontFamily: 'inherit' }}>
      {/* Match Header */}
      <div style={{
        background: '#1a1a2e',
        border: '1px solid #333',
        borderRadius: 12,
        padding: 32,
        textAlign: 'center',
        marginBottom: 24,
      }}>
        <div style={{ color: '#888', fontSize: '0.85rem', marginBottom: 8 }}>
          {state.competition_title} · {state.competition_stage?.replace(/_/g, ' ')}
        </div>

        {/* Scoreboard */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 32,
          margin: '24px 0',
        }}>
          <div style={{ flex: 1, textAlign: 'right' }}>
            <div style={{ fontSize: '1.4rem', fontWeight: 700 }}>{state.team_a}</div>
            <div style={{ color: '#888', fontSize: '0.85rem' }}>{state.team_a_code}</div>
          </div>
          <div style={{
            background: '#0d0d1a',
            border: '2px solid #333',
            borderRadius: 8,
            padding: '12px 24px',
            minWidth: 120,
          }}>
            <div style={{ fontSize: '2.5rem', fontWeight: 900, letterSpacing: 4 }}>
              {state.score_a} : {state.score_b}
            </div>
            <div style={{ color: statusColor, fontWeight: 700, fontSize: '0.8rem', marginTop: 4 }}>
              {state.status === 'LIVE' ? `${state.minute}'` : state.status}
            </div>
          </div>
          <div style={{ flex: 1, textAlign: 'left' }}>
            <div style={{ fontSize: '1.4rem', fontWeight: 700 }}>{state.team_b}</div>
            <div style={{ color: '#888', fontSize: '0.85rem' }}>{state.team_b_code}</div>
          </div>
        </div>

        {/* Status badge */}
        <div style={{ display: 'flex', justifyContent: 'center', gap: 12, alignItems: 'center' }}>
          <span style={{
            background: statusColor,
            color: '#fff',
            borderRadius: 4,
            padding: '3px 10px',
            fontSize: '0.75rem',
            fontWeight: 700,
            letterSpacing: 1,
          }}>
            {state.status}
          </span>
          <span style={{ color: connected ? '#2a9d8f' : '#e63946', fontSize: '0.75rem' }}>
            {connected ? '● CONNECTED' : '○ RECONNECTING'}
          </span>
        </div>
      </div>

      {/* Last event highlight */}
      {state.last_event && (
        <div style={{
          background: '#12121f',
          border: '1px solid #333',
          borderRadius: 8,
          padding: '12px 20px',
          marginBottom: 16,
          display: 'flex',
          alignItems: 'center',
          gap: 10,
        }}>
          <span style={{ fontSize: '1.4rem' }}>{EVENT_EMOJI[state.last_event] ?? '📋'}</span>
          <div>
            <div style={{ fontWeight: 600 }}>{state.last_event.replace(/_/g, ' ')}</div>
            <div style={{ color: '#888', fontSize: '0.8rem' }}>
              {new Date(state.last_updated).toLocaleTimeString()}
            </div>
          </div>
        </div>
      )}

      {/* Event feed */}
      {events.length > 0 && (
        <div style={{
          background: '#12121f',
          border: '1px solid #222',
          borderRadius: 8,
          padding: 16,
        }}>
          <h2 style={{ color: '#888', fontSize: '0.75rem', fontWeight: 600, letterSpacing: 1, marginBottom: 12 }}>
            LIVE FEED
          </h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {events.map((e, i) => (
              <div key={i} style={{
                padding: '8px 12px',
                background: i === 0 ? '#1a1a2e' : 'transparent',
                borderRadius: 4,
                fontSize: '0.9rem',
                color: i === 0 ? '#fff' : '#aaa',
                transition: 'all 0.3s',
              }}>
                {e}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
