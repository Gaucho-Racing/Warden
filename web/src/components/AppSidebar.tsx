import { KeyRound, Shield, Users, Settings } from "lucide-react"
import { Link, useLocation } from "react-router-dom"

import { useAuth } from "@/lib/auth"
import { cn } from "@/lib/utils"

const navItems = [
  { to: "/account", label: "My Account", icon: KeyRound, adminOnly: false },
  { to: "/players", label: "Players", icon: Users, adminOnly: true },
  { to: "/bindings", label: "Bindings", icon: Shield, adminOnly: true },
  { to: "/settings", label: "Settings", icon: Settings, adminOnly: false },
]

function isActive(currentPath: string, target: string) {
  return currentPath === target || currentPath.startsWith(`${target}/`)
}

export function AppSidebar() {
  const { pathname } = useLocation()
  const { isAdmin } = useAuth()

  return (
    <aside className="hidden border-r border-sidebar-border/70 bg-sidebar text-sidebar-foreground lg:flex lg:min-h-svh lg:flex-col">
      <div className="flex h-16 items-center gap-3 border-b border-sidebar-border/70 px-4">
        <div className="flex size-9 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm shadow-primary/25">
          <Shield className="size-5" />
        </div>
        <div className="min-w-0 text-base font-semibold leading-none">Warden</div>
      </div>
      <nav className="flex-1 space-y-1 px-3 py-4">
        {navItems
          .filter((item) => !item.adminOnly || isAdmin)
          .map((item) => {
            const active = isActive(pathname, item.to)
            return (
              <Link
                key={item.to}
                to={item.to}
                className={cn(
                  "flex h-10 items-center gap-2 rounded-lg px-3 text-sm font-medium transition-colors",
                  active
                    ? "bg-primary text-primary-foreground shadow-sm shadow-primary/15"
                    : "text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                )}
              >
                <item.icon className="size-4" />
                <span>{item.label}</span>
              </Link>
            )
          })}
      </nav>
    </aside>
  )
}
