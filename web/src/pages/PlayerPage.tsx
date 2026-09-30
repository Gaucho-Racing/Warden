import { ArrowLeft } from "lucide-react"
import { Link, useParams } from "react-router-dom"

import { PageContainer } from "@/components/PageContainer"
import { NoStatsYet, PlayerStatsView } from "@/components/PlayerStatsView"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { errorMessage, usePlayerAccount, usePlayerStats } from "@/lib/warden"

const backLink = (
  <Button asChild variant="ghost" size="sm" className="mb-4 -ml-2">
    <Link to="/players">
      <ArrowLeft className="size-4" />
      Players
    </Link>
  </Button>
)

// Any player's stats, the same view as Home. Open to every signed-in member,
// like the roster it is reached from.
export default function PlayerPage() {
  const { uuid } = useParams()
  const account = usePlayerAccount(uuid)
  const stats = usePlayerStats(uuid)

  if (account.isLoading || stats.isLoading) {
    return (
      <PageContainer>
        {backLink}
        <Skeleton className="h-28 w-full" />
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
      </PageContainer>
    )
  }

  const error = account.error ?? stats.error
  if (error) {
    return (
      <PageContainer>
        {backLink}
        <Card>
          <CardContent className="p-6 text-sm text-destructive">
            {errorMessage(error, "Could not load this player.")}
          </CardContent>
        </Card>
      </PageContainer>
    )
  }

  if (!account.data || !stats.data) {
    return (
      <PageContainer>
        {backLink}
        <Card>
          <CardContent className="p-6 text-sm text-muted-foreground">
            No linked player with that UUID.
          </CardContent>
        </Card>
      </PageContainer>
    )
  }

  if (stats.data.source === "none") {
    return (
      <NoStatsYet backLink={backLink}>
        {account.data.username} is linked, but the server hasn&apos;t reported any statistics for
        them yet.
      </NoStatsYet>
    )
  }

  const name = account.data.identity?.name
  return (
    <PlayerStatsView
      stats={stats.data}
      avatarUrl={account.data.avatar_url}
      greeting={name ? `${name}. ` : undefined}
      backLink={backLink}
    />
  )
}
