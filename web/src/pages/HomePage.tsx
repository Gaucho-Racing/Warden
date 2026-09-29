import { Link } from "react-router-dom"

import { PageContainer } from "@/components/PageContainer"
import { SkinFrame } from "@/components/SkinFrame"
import { StatTile } from "@/components/StatTile"
import { TopBlocksChart } from "@/components/TopBlocksChart"
import { PixelGrassBlock } from "@/components/icons/pixel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/lib/auth"
import { compact } from "@/lib/format"
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
  const hours = Math.floor(s.playtime_minutes / 60)
  const firstName = user?.first_name || user?.username || "there"

  return (
    <PageContainer>
      <div className="mb-6 flex flex-wrap items-center gap-4">
        <SkinFrame src={account.data.avatar_url} alt={s.username} className="size-16" />
        <div className="min-w-0 flex-1">
          <h1 className="font-pixel text-3xl leading-none">{s.username}</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Welcome back, {firstName}. Last seen{" "}
            {new Date(s.last_seen).toLocaleString(undefined, {
              month: "short",
              day: "numeric",
              hour: "numeric",
              minute: "2-digit",
            })}
            .
          </p>
        </div>
        {s.source === "mock" && (
          <Badge variant="secondary" className="shrink-0">
            Mock data
          </Badge>
        )}
      </div>

      {/* Hero figure — the one number the page leads with. */}
      <Card className="mb-4">
        <CardContent className="flex flex-wrap items-end justify-between gap-4 p-6">
          <div>
            <div className="text-xs text-muted-foreground">Total playtime</div>
            <div className="mt-2 flex items-baseline gap-2">
              <span className="font-pixel text-6xl leading-none">{compact(hours)}</span>
              <span className="text-sm text-muted-foreground">hours</span>
            </div>
          </div>
          <div className="text-right text-xs text-muted-foreground">
            <div>
              {s.join_count.toLocaleString()} session{s.join_count === 1 ? "" : "s"}
            </div>
            <div className="mt-1">
              Since{" "}
              {new Date(s.first_seen).toLocaleDateString(undefined, {
                month: "short",
                year: "numeric",
              })}
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatTile
          label="Blocks mined"
          value={compact(s.blocks_mined)}
          delta={s.last_7_days?.blocks_mined}
        />
        <StatTile
          label="Mob kills"
          value={compact(s.mob_kills)}
          delta={s.last_7_days?.mob_kills}
        />
        <StatTile
          label="Deaths"
          value={compact(s.deaths)}
          delta={s.last_7_days?.deaths}
          upIsGood={false}
        />
        <StatTile
          label="Distance travelled"
          value={compact(Math.round(s.distance_meters / 1000))}
          unit="km"
        />
      </div>

      <Card className="mt-4">
        <CardContent className="p-6">
          <h2 className="mb-4 font-pixel text-xl">Most mined blocks</h2>
          <TopBlocksChart blocks={s.top_blocks} />
        </CardContent>
      </Card>
    </PageContainer>
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
