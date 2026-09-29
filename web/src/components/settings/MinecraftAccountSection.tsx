import { Loader2, Unlink } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { SkinFrame } from "@/components/SkinFrame"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { MINECRAFT_SERVER_ADDRESS } from "@/lib/links"
import { errorMessage, useMyAccount, useUnlinkAccount } from "@/lib/warden"

export function MinecraftAccountSection() {
  const account = useMyAccount()
  const unlink = useUnlinkAccount()
  const [confirming, setConfirming] = useState(false)

  return (
    <section className="space-y-3">
      <div>
        <h2 className="font-pixel text-xl">Minecraft account</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          The Minecraft account connected to your Sentinel account.
        </p>
      </div>

      {account.isLoading && <Skeleton className="h-28 w-full" />}

      {!account.isLoading && !account.data && (
        <Card>
          <CardContent className="p-6 text-sm text-muted-foreground">
            Nothing linked yet. Join{" "}
            <span className="mc-bevel-in bg-input px-2 py-0.5 font-mono text-xs">
              {MINECRAFT_SERVER_ADDRESS}
            </span>{" "}
            and click the link that appears in chat.
          </CardContent>
        </Card>
      )}

      {account.data && (
        <Card>
          <CardContent className="flex flex-wrap items-center gap-4 p-6">
            <SkinFrame
              src={account.data.avatar_url}
              alt={account.data.username}
              className="size-16"
            />
            <div className="min-w-0 flex-1">
              <div className="font-pixel text-xl">{account.data.username}</div>
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
    </section>
  )
}
