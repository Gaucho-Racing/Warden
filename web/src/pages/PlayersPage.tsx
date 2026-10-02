import { useState } from "react"
import { Link } from "react-router-dom"

import { PageContainer, PageHeader } from "@/components/PageContainer"
import { SkinFrame } from "@/components/SkinFrame"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { errorMessage, type MinecraftAccount, useAccounts, useServerStatus } from "@/lib/warden"

function formatPlaytime(minutes: number) {
  const hours = Math.floor(minutes / 60)
  if (hours === 0) return `${minutes}m`
  if (hours < 100) return `${hours}h ${minutes % 60}m`
  return `${hours.toLocaleString()}h`
}

// Go's zero time, which the API sends for a player who has never joined.
function seenAt(account: MinecraftAccount) {
  const value = account.last_seen_at
  return value && !value.startsWith("0001") ? new Date(value).getTime() : null
}

function formatLastSeen(seen: number | null) {
  if (seen === null) return "Never seen"
  const minutes = Math.floor((Date.now() - seen) / 60_000)
  if (minutes < 1) return "Last seen just now"
  if (minutes < 60) return `Last seen ${minutes}m ago`
  if (minutes < 24 * 60) return `Last seen ${Math.floor(minutes / 60)}h ago`
  return `Last seen ${new Date(seen).toLocaleDateString()}`
}

function RosterStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="w-20 text-right">
      <div className="font-pixel text-base leading-tight">{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  )
}

export default function PlayersPage() {
  const accounts = useAccounts()
  const serverStatus = useServerStatus()
  const onlinePlayers = new Set(serverStatus.data?.online_players ?? [])
  const [filter, setFilter] = useState("")

  const term = filter.trim().toLowerCase()
  const rows = (accounts.data ?? [])
    .filter((account) => {
      if (!term) return true
      return (
        account.username.toLowerCase().includes(term) ||
        account.uuid.includes(term) ||
        (account.identity?.name ?? "").toLowerCase().includes(term) ||
        (account.identity?.username ?? "").toLowerCase().includes(term)
      )
    })
    // Most recently seen first; players who have never joined go last.
    .sort((a, b) => (seenAt(b) ?? 0) - (seenAt(a) ?? 0))

  return (
    <PageContainer>
      <PageHeader
        title="Players"
        description="Every Minecraft account linked to a Sentinel account."
        action={
          <Input
            placeholder="Search players"
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
            className="sm:w-64"
          />
        }
      />

      {accounts.isLoading && <Skeleton className="h-40 w-full rounded-xl" />}

      {accounts.isError && (
        <Card>
          <CardContent className="p-6 text-sm text-destructive">
            {errorMessage(accounts.error, "Could not load linked players.")}
          </CardContent>
        </Card>
      )}

      {accounts.data && rows.length === 0 && (
        <Card>
          <CardContent className="p-6 text-sm text-muted-foreground">
            {term ? "No players match that search." : "No accounts have been linked yet."}
          </CardContent>
        </Card>
      )}

      <div className="space-y-2">
        {rows.map((account) => (
          <Link key={account.uuid} to={`/players/${account.uuid}`} className="block">
            <Card className="transition-colors hover:border-primary/60">
              <CardContent className="flex flex-wrap items-center gap-4 p-4">
                <SkinFrame src={account.avatar_url} alt={account.username} className="size-12" />
                <div className="min-w-0 flex-1">
                  <div className="font-pixel text-lg leading-tight">{account.username}</div>
                  <div className="font-mono text-xs text-muted-foreground">{account.uuid}</div>
                </div>
                <div className="min-w-0 flex-1">
                  <div className="font-pixel text-base leading-tight">{account.identity?.name ?? account.entity_id}</div>
                  {account.identity?.username && (
                    <div className="text-xs text-muted-foreground">
                      @{account.identity.username}
                    </div>
                  )}
                </div>
                <RosterStat
                  label="Playtime"
                  value={account.stats ? formatPlaytime(account.stats.playtime_minutes) : "—"}
                />
                <RosterStat
                  label="Sessions"
                  value={account.stats ? account.stats.sessions.toLocaleString() : "—"}
                />
                {onlinePlayers.has(account.uuid) ? (
                  <Badge>
                    <span className="size-1.5 rounded-full bg-current" />
                    Online
                  </Badge>
                ) : (
                  <div className="text-xs text-muted-foreground">{formatLastSeen(seenAt(account))}</div>
                )}
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </PageContainer>
  )
}
