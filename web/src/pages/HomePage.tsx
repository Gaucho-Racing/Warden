import { Link } from "react-router-dom"

import { PageContainer } from "@/components/PageContainer"
import { NoStatsYet, PlayerStatsView } from "@/components/PlayerStatsView"
import { PixelGrassBlock } from "@/components/icons/pixel"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/lib/auth"
import { MINECRAFT_SERVER_ADDRESS } from "@/lib/links"
import { errorMessage, useMyAccount, useMyStats } from "@/lib/warden"

export default function HomePage() {
  const { user } = useAuth()
  const account = useMyAccount()
  const stats = useMyStats()

  if (account.isLoading || stats.isLoading) {
    return (
      <PageContainer>
        <Skeleton className="h-28 w-full" />
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
      </PageContainer>
    )
  }

  if (!account.data) return <GetStarted />

  if (stats.isError) {
    return (
      <PageContainer>
        <Card>
          <CardContent className="p-6 text-sm text-destructive">
            {errorMessage(stats.error, "Could not load your stats.")}
          </CardContent>
        </Card>
      </PageContainer>
    )
  }
  if (!stats.data) return <GetStarted />

  const s = stats.data
  if (s.source === "none") {
    return (
      <NoStatsYet>
        {s.username} is linked, but the server hasn&apos;t reported any statistics. They appear
        after your next session on {MINECRAFT_SERVER_ADDRESS}.
      </NoStatsYet>
    )
  }
  const firstName = user?.first_name || user?.username || "there"

  return (
    <PlayerStatsView
      stats={s}
      avatarUrl={account.data.avatar_url}
      greeting={`Welcome back, ${firstName}. `}
    />
  )
}

function GetStarted() {
  const steps = [
    {
      title: "Join the server",
      body: (
        <>
          Open Minecraft Java Edition and connect to{" "}
          <span className="mc-bevel-in bg-input px-2 py-0.5 font-mono text-xs">
            {MINECRAFT_SERVER_ADDRESS}
          </span>
          .
        </>
      ),
    },
    {
      title: "Click the link in chat",
      body: "Warden posts a one-time link as soon as you join. It expires after 15 minutes — rejoin for a fresh one if you miss it.",
    },
    {
      title: "Confirm it's you",
      body: "You're already signed in here, so it's a single click. Your permissions apply the next time you join.",
    },
  ]

  return (
    <PageContainer>
      <div className="mx-auto max-w-2xl">
        <div className="mb-8 flex flex-col items-center gap-4 text-center">
          <PixelGrassBlock className="size-16" />
          <div>
            <h1 className="font-pixel text-3xl leading-none">Link your Minecraft account</h1>
            <p className="mt-3 text-sm text-muted-foreground">
              Your Sentinel account decides what you can do on the server. Link them once and your
              permissions follow your Sentinel groups automatically.
            </p>
          </div>
        </div>

        <div className="space-y-3">
          {steps.map((step, i) => (
            <Card key={step.title}>
              <CardContent className="flex gap-4 p-5">
                <div className="mc-bevel flex size-9 shrink-0 items-center justify-center bg-primary font-pixel text-lg text-primary-foreground">
                  {i + 1}
                </div>
                <div className="min-w-0">
                  <div className="font-pixel text-lg leading-none">{step.title}</div>
                  <p className="mt-2 text-sm text-muted-foreground">{step.body}</p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          Already linked on another account?{" "}
          <Button asChild variant="link" className="h-auto p-0 text-xs">
            <Link to="/account">Manage it here</Link>
          </Button>
        </p>
      </div>
    </PageContainer>
  )
}
