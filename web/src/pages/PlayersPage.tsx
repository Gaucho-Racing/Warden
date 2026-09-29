import { Loader2, Unlink } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { PageContainer, PageHeader } from "@/components/PageContainer"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { errorMessage, skinURL, useAccounts, useUnlinkAccount } from "@/lib/warden"

export default function PlayersPage() {
  const accounts = useAccounts()
  const unlink = useUnlinkAccount()
  const [filter, setFilter] = useState("")

  const term = filter.trim().toLowerCase()
  const rows = (accounts.data ?? []).filter((account) => {
    if (!term) return true
    return (
      account.username.toLowerCase().includes(term) ||
      account.uuid.includes(term) ||
      (account.identity?.name ?? "").toLowerCase().includes(term) ||
      (account.identity?.username ?? "").toLowerCase().includes(term)
    )
  })

  return (
    <PageContainer>
      <PageHeader
        title="Players"
        description="Every Minecraft account linked to a Sentinel identity."
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
          <Card key={account.uuid}>
            <CardContent className="flex flex-wrap items-center gap-4 p-4">
              <img
                src={skinURL(account.uuid)}
                alt={account.username}
                className="mc-bevel-thin size-11"
              />
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
              <div className="text-xs text-muted-foreground">
                {account.last_seen_at && !account.last_seen_at.startsWith("0001")
                  ? `Seen ${new Date(account.last_seen_at).toLocaleDateString()}`
                  : "Never seen"}
              </div>
              <Button
                variant="ghost"
                size="icon"
                disabled={unlink.isPending}
                onClick={() =>
                  unlink.mutate(account.uuid, {
                    onSuccess: () => toast.success(`Unlinked ${account.username}`),
                    onError: (error) =>
                      toast.error(errorMessage(error, "Could not unlink that account")),
                  })
                }
              >
                {unlink.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Unlink className="size-4" />
                )}
                <span className="sr-only">Unlink {account.username}</span>
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
    </PageContainer>
  )
}
