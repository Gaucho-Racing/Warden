import { Monitor, Moon, Sun } from "lucide-react"

import { PageContainer, PageHeader } from "@/components/PageContainer"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { useAuth } from "@/lib/auth"
import { MINECRAFT_SERVER_ADDRESS } from "@/lib/links"
import { useTheme, type Theme } from "@/lib/theme"
import { cn } from "@/lib/utils"

const themeOptions: { value: Theme; label: string; icon: typeof Sun }[] = [
  { value: "light", label: "Light", icon: Sun },
  { value: "dark", label: "Dark", icon: Moon },
  { value: "system", label: "System", icon: Monitor },
]

export default function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const { user, isAdmin } = useAuth()

  return (
    <PageContainer>
      <PageHeader title="Settings" />

      <div className="space-y-4">
        <Card>
          <CardContent className="space-y-3 p-6">
            <div>
              <h2 className="font-medium">Appearance</h2>
              <p className="text-sm text-muted-foreground">How Warden looks on this device.</p>
            </div>
            <div className="flex gap-2">
              {themeOptions.map((option) => (
                <Button
                  key={option.value}
                  variant={theme === option.value ? "default" : "outline"}
                  onClick={() => setTheme(option.value)}
                  className={cn("flex-1")}
                >
                  <option.icon className="size-4" />
                  {option.label}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-2 p-6 text-sm">
            <h2 className="font-medium">Session</h2>
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
