import type { ReactNode } from "react"

import { PageContainer } from "@/components/PageContainer"
import { SkinFrame } from "@/components/SkinFrame"
import { StatTile } from "@/components/StatTile"
import { TopCountsChart } from "@/components/TopCountsChart"
import { PixelGrassBlock } from "@/components/icons/pixel"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { compact } from "@/lib/format"
import type { PlayerStats } from "@/lib/warden"

/**
 * One player's stat page, shared by Home (your own) and the player details
 * page (anyone's). greeting is prepended to the "Last seen" line; backLink
 * sits above the header.
 */
export function PlayerStatsView({
  stats: s,
  avatarUrl,
  greeting,
  backLink,
}: {
  stats: PlayerStats
  avatarUrl: string
  greeting?: ReactNode
  backLink?: ReactNode
}) {
  const hours = Math.floor(s.playtime_minutes / 60)

  return (
    <PageContainer>
      {backLink}
      <div className="mb-6 flex flex-wrap items-center gap-4">
        <SkinFrame src={avatarUrl} alt={s.username} className="size-16" />
        <div className="min-w-0 flex-1">
          <h1 className="font-pixel text-3xl leading-none">{s.username}</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {greeting}Last seen{" "}
            {new Date(s.last_seen).toLocaleString(undefined, {
              month: "short",
              day: "numeric",
              hour: "numeric",
              minute: "2-digit",
            })}
            .
          </p>
        </div>
        <Badge variant="secondary" className="shrink-0">
          {s.sessions.toLocaleString()} session{s.sessions === 1 ? "" : "s"}
        </Badge>
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
        <StatTile label="Mob kills" value={compact(s.mob_kills)} delta={s.last_7_days?.mob_kills} />
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
          delta={
            s.last_7_days ? Math.round(s.last_7_days.distance_meters / 1000) : undefined
          }
        />
        <StatTile label="Items crafted" value={compact(s.items_crafted)} />
        <StatTile label="Jumps" value={compact(s.jumps)} />
        <StatTile label="Nights slept" value={compact(s.times_slept)} />
        <StatTile label="Villager trades" value={compact(s.villager_trades)} />
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <Card className="lg:col-span-2">
          <CardContent className="p-6">
            <h2 className="mb-4 font-pixel text-xl">Most mined blocks</h2>
            <TopCountsChart counts={s.top_blocks} />
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <h2 className="mb-4 font-pixel text-xl">Most killed mobs</h2>
            <TopCountsChart counts={s.top_mobs} />
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <h2 className="mb-4 font-pixel text-xl">Most crafted items</h2>
            <TopCountsChart counts={s.top_crafted} />
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  )
}

/**
 * Linked, but the plugin has never reported. Distinct from unlinked and
 * from a genuine run of zeroes.
 */
export function NoStatsYet({ children, backLink }: { children: ReactNode; backLink?: ReactNode }) {
  return (
    <PageContainer>
      {backLink}
      <div className="mx-auto max-w-xl text-center">
        <PixelGrassBlock className="mx-auto size-14" />
        <h1 className="mt-4 font-pixel text-2xl">No stats yet</h1>
        <p className="mt-3 text-sm text-muted-foreground">{children}</p>
      </div>
    </PageContainer>
  )
}
