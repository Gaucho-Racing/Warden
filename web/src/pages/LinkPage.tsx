import { ArrowRight, CheckCircle2, Loader2, ShieldAlert } from "lucide-react"
import { useState } from "react"
import { Link, useParams } from "react-router-dom"

import { WardenMark } from "@/components/icons/pixel"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/lib/auth"
import { MINECRAFT_SERVER_ADDRESS } from "@/lib/links"
import { errorMessage, useConfirmLink, useLinkToken, type MinecraftAccount } from "@/lib/warden"

export default function LinkPage() {
  const { token } = useParams<{ token: string }>()
  const { user } = useAuth()
  const linkToken = useLinkToken(token)
  const confirmLink = useConfirmLink()
  const [linked, setLinked] = useState<MinecraftAccount | null>(null)

  if (linked) {
    return (
      <LinkLayout>
        <div className="flex flex-col items-center gap-4 text-center">
          <CheckCircle2 className="size-10 text-gr-pink" />
          <div className="space-y-1">
            <h1 className="font-pixel text-2xl">Account linked</h1>
            <p className="text-sm text-muted-foreground">
              <span className="font-medium text-foreground">{linked.username}</span> is now linked
              to your Gaucho Racing account. Rejoin{" "}
              <span className="font-mono">{MINECRAFT_SERVER_ADDRESS}</span> to pick up your
              permissions.
            </p>
          </div>
          <Button asChild variant="outline">
            <Link to="/account">View my account</Link>
          </Button>
        </div>
      </LinkLayout>
    )
  }

  if (linkToken.isLoading) {
    return (
      <LinkLayout>
        <div className="space-y-4">
          <Skeleton className="mx-auto size-16 rounded-lg" />
          <Skeleton className="mx-auto h-4 w-40 rounded-full" />
          <Skeleton className="h-10 w-full rounded-lg" />
        </div>
      </LinkLayout>
    )
  }

  if (linkToken.isError || !linkToken.data) {
    return (
      <LinkLayout>
        <div className="flex flex-col items-center gap-4 text-center">
          <ShieldAlert className="size-10 text-destructive" />
          <div className="space-y-1">
            <h1 className="font-pixel text-2xl">This link isn&apos;t valid</h1>
            <p className="text-sm text-muted-foreground">
              {errorMessage(
                linkToken.error,
                "It may have expired or already been used. Rejoin the server to get a fresh one.",
              )}
            </p>
          </div>
          <Button asChild variant="outline">
            <Link to="/account">Back to my account</Link>
          </Button>
        </div>
      </LinkLayout>
    )
  }

  const preview = linkToken.data
  const sentinelName =
    `${user?.first_name ?? ""} ${user?.last_name ?? ""}`.trim() || user?.username || "your account"

  return (
    <LinkLayout>
      <div className="space-y-6">
        <div className="space-y-1 text-center">
          <h1 className="font-pixel text-2xl">Link your Minecraft account</h1>
          <p className="text-sm text-muted-foreground">
            Confirm this is you, and we&apos;ll connect it to your Gaucho Racing account.
          </p>
        </div>

        <div className="flex items-center justify-center gap-4">
          <div className="flex flex-col items-center gap-2">
            <img
              src={preview.avatar_url}
              alt={preview.username}
              className="mc-bevel-thin size-16"
            />
            <div className="text-center">
              <div className="font-pixel text-base">{preview.username}</div>
              <div className="font-mono text-[10px] text-muted-foreground">
                {preview.uuid.slice(0, 8)}
              </div>
            </div>
          </div>

          <ArrowRight className="size-5 shrink-0 text-muted-foreground" />

          <div className="flex flex-col items-center gap-2">
            <Avatar className="mc-bevel-thin size-16">
              <AvatarImage src={user?.avatar_url} alt={sentinelName} />
              <AvatarFallback className="font-pixel">
                {sentinelName.slice(0, 2).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div className="text-center">
              <div className="font-pixel text-base">{sentinelName}</div>
              <div className="text-[10px] text-muted-foreground">Sentinel</div>
            </div>
          </div>
        </div>

        {confirmLink.isError && (
          <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-center text-sm text-destructive">
            {errorMessage(confirmLink.error, "Could not link that account. Try again.")}
          </div>
        )}

        <Button
          size="lg"
          className="h-12 w-full text-base"
          disabled={confirmLink.isPending}
          onClick={() =>
            confirmLink.mutate(preview.token, { onSuccess: (account) => setLinked(account) })
          }
        >
          {confirmLink.isPending && <Loader2 className="size-4 animate-spin" />}
          Link {preview.username}
        </Button>

        <p className="text-center text-xs text-muted-foreground">
          Not you?{" "}
          <Link to="/account" className="underline underline-offset-2">
            Cancel
          </Link>
        </p>
      </div>
    </LinkLayout>
  )
}

function LinkLayout({ children }: { children: React.ReactNode }) {
  return (
    <main className="mc-backdrop flex min-h-svh flex-col items-center justify-center gap-6 px-4 py-12">
      <div className="flex flex-col items-center gap-2">
        <WardenMark className="size-14 mc-glow" />
        <span className="font-display text-[0.7rem] text-white drop-shadow-[2px_2px_0_rgba(0,0,0,0.8)]">
          WARDEN
        </span>
      </div>
      <Card className="w-full max-w-md">
        <CardContent className="p-6">{children}</CardContent>
      </Card>
    </main>
  )
}
