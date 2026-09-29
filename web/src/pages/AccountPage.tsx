import { Loader2, Unlink } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { PageContainer, PageHeader } from "@/components/PageContainer"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { MINECRAFT_SERVER_ADDRESS } from "@/lib/links"
import { errorMessage, skinURL, useMyAccount, useUnlinkAccount } from "@/lib/warden"

export default function AccountPage() {
  const account = useMyAccount()
  const unlink = useUnlinkAccount()
  const [confirming, setConfirming] = useState(false)

  return (
    <PageContainer>
      <PageHeader
        title="My Account"
        description="The Minecraft account connected to your Gaucho Racing identity."
      />

      {account.isLoading && <Skeleton className="h-32 w-full rounded-xl" />}

      {!account.isLoading && !account.data && (
        <Card>
          <CardContent className="space-y-3 p-6">
            <h2 className="font-medium">No Minecraft account linked</h2>
            <p className="text-sm text-muted-foreground">
              Join <span className="font-mono">{MINECRAFT_SERVER_ADDRESS}</span> and click the link
              that appears in chat. It signs you in here and connects your account automatically.
            </p>
          </CardContent>
        </Card>
      )}

      {account.data && (
        <Card>
          <CardContent className="flex flex-wrap items-center gap-4 p-6">
            <img
              src={skinURL(account.data.uuid)}
              alt={account.data.username}
              className="size-16 rounded-lg border border-border"
            />
            <div className="min-w-0 flex-1">
              <div className="font-medium">{account.data.username}</div>
              <div className="font-mono text-xs text-muted-foreground">{account.data.uuid}</div>
              <div className="mt-1 text-xs text-muted-foreground">
                Linked {new Date(account.data.created_at).toLocaleString()}
              </div>
            </div>
            <Button
              variant={confirming ? "destructive" : "outline"}
              disabled={unlink.isPending}
              onClick={() => {
                if (!confirming) {
                  setConfirming(true)
                  return
                }
                unlink.mutate(account.data!.uuid, {
                  onSuccess: () => {
                    toast.success("Minecraft account unlinked")
                    setConfirming(false)
                  },
                  onError: (error) =>
                    toast.error(errorMessage(error, "Could not unlink that account")),
                })
              }}
            >
              {unlink.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Unlink className="size-4" />
              )}
              {confirming ? "Confirm unlink" : "Unlink"}
            </Button>
          </CardContent>
        </Card>
      )}
    </PageContainer>
  )
}
