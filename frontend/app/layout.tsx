import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Copa World – Live Scores',
  description: 'Real-time football scores powered by Go + Kafka + Redis',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
