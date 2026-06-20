import type { Metadata } from 'next'
import Link from 'next/link'
import './globals.css'

export const metadata: Metadata = {
  title: 'Copa World – Live Scores',
  description: 'Real-time football scores powered by Go + Kafka + Redis',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="pt-BR" className="dark">
      <body className="min-h-screen bg-background">
        <header className="sticky top-0 z-50 w-full border-b border-border bg-background/80 backdrop-blur-sm">
          <div className="mx-auto max-w-5xl px-4 sm:px-6">
            <div className="flex h-14 items-center justify-between">
              <Link href="/" className="flex items-center gap-2">
                <span className="text-lg font-bold tracking-tight text-foreground">
                  ⚽ Copa World
                </span>
              </Link>
              <div className="flex items-center gap-1">
                <span className="inline-flex items-center gap-1.5 rounded-full bg-red-600/15 px-3 py-1 text-xs font-semibold text-red-400 ring-1 ring-inset ring-red-600/25">
                  <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-red-500" />
                  AO VIVO
                </span>
              </div>
            </div>
          </div>
        </header>
        <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
          {children}
        </main>
      </body>
    </html>
  )
}
