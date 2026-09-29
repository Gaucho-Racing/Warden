import { Link, useLocation } from "react-router-dom"

import { PixelGear, PixelHead, PixelHeads, PixelKey, WardenMark } from "@/components/icons/pixel"
import { useAuth } from "@/lib/auth"
import { MINECRAFT_SERVER_ADDRESS } from "@/lib/links"
import { cn } from "@/lib/utils"

const navItems = [
  { to: "/account", label: "My Account", icon: PixelHead, adminOnly: false },
  { to: "/players", label: "Players", icon: PixelHeads, adminOnly: true },
  { to: "/bindings", label: "Bindings", icon: PixelKey, adminOnly: true },
  { to: "/settings", label: "Settings", icon: PixelGear, adminOnly: false },
]

function isActive(currentPath: string, target: string) {
  return currentPath === target || currentPath.startsWith(`${target}/`)
}

export function AppSidebar() {
  const { pathname } = useLocation()
  const { isAdmin } = useAuth()

  return (
    <aside className="hidden border-r-4 border-sidebar-border bg-sidebar text-sidebar-foreground lg:flex lg:min-h-svh lg:flex-col">
      <div className="flex h-20 items-center gap-3 border-b-4 border-sidebar-border px-4">
        <WardenMark className="size-10 shrink-0 mc-glow" />
        <div className="min-w-0">
          <div className="font-display text-[0.7rem] leading-none tracking-tight">WARDEN</div>
          <div className="mt-1.5 font-mono text-[0.6rem] text-muted-foreground">
            {MINECRAFT_SERVER_ADDRESS}
          </div>
        </div>
      </div>

      <nav className="flex-1 space-y-1.5 p-3">
        {navItems
          .filter((item) => !item.adminOnly || isAdmin)
          .map((item) => {
            const active = isActive(pathname, item.to)
            return (
              <Link
                key={item.to}
                to={item.to}
                className={cn(
                  "flex h-11 items-center gap-3 px-3 font-pixel text-base transition-none",
                  active
                    ? "mc-bevel bg-primary text-primary-foreground"
                    : "mc-bevel-thin bg-sidebar-accent/40 text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                )}
                style={
                  active
                    ? ({
                        "--mc-bevel-light": "color-mix(in oklab, var(--primary) 62%, white)",
                        "--mc-bevel-dark": "color-mix(in oklab, var(--primary) 60%, black)",
                      } as React.CSSProperties)
                    : undefined
                }
              >
                <item.icon className="size-5 shrink-0" />
                <span>{item.label}</span>
              </Link>
            )
          })}
      </nav>
    </aside>
  )
}
