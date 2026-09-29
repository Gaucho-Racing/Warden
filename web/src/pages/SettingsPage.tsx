import { PageContainer, PageHeader } from "@/components/PageContainer"
import { Card, CardContent } from "@/components/ui/card"
import { useAuth } from "@/lib/auth"
import { MINECRAFT_SERVER_ADDRESS } from "@/lib/links"
import { cn } from "@/lib/utils"

export default function SettingsPage() {
  const { user, isAdmin } = useAuth()

  return (
    <PageContainer>
      <PageHeader title="Settings" />

      <div className="space-y-4">
        <Card>
          <CardContent className="space-y-2 p-6 text-sm">
            <h2 className="font-pixel text-xl">Session</h2>
            <Row label="Signed in as" value={user?.username ?? "—"} />
            <Row label="Entity" value={user?.entity_id ?? "—"} mono />
            <Row label="Role" value={isAdmin ? "Admin" : "Member"} />
            <Row label="Server" value={MINECRAFT_SERVER_ADDRESS} mono />
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  )
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className={cn(mono && "font-mono text-xs")}>{value}</span>
    </div>
  )
}
