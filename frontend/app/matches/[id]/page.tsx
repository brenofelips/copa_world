import Link from 'next/link'
import LiveMatch from '@/components/LiveMatch'
import { Button } from '@/components/ui/button'

interface Props {
  params: { id: string }
}

export default function MatchPage({ params }: Props) {
  const matchId = decodeURIComponent(params.id)

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2">
        <Link href="/">
          <Button variant="ghost" size="sm" className="h-8 gap-1.5 text-muted-foreground hover:text-foreground">
            ← Partidas
          </Button>
        </Link>
        <span className="text-muted-foreground/40">/</span>
        <span className="text-sm text-muted-foreground truncate max-w-[200px] sm:max-w-none">
          {matchId}
        </span>
      </div>

      {/* Live component */}
      <LiveMatch matchId={matchId} />

      {/* Stats link */}
      <div>
        <a
          href={`${process.env.NEXT_PUBLIC_WEB_SERVER_URL || 'http://localhost:8080'}/matches/${matchId}/statistic`}
          target="_blank"
          rel="noreferrer"
        >
          <Button variant="outline" size="sm" className="gap-2 text-muted-foreground">
            Ver estatísticas (JSON) ↗
          </Button>
        </a>
      </div>
    </div>
  )
}
