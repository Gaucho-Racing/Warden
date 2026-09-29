import { Link, useLocation } from "react-router-dom"

import {
  PixelGear,
  PixelGrassBlock,
  PixelHouse,
  PixelHeads,
} from "@/components/icons/pixel"
import { useAuth } from "@/lib/auth"
import { cn } from "@/lib/utils"

const navItems = [
  { to: "/", label: "Home", icon: PixelHouse, adminOnly: false },
  { to: "/players", label: "Players", icon: PixelHeads, adminOnly: true },
  { to: "/settings", label: "Settings", icon: PixelGear, adminOnly: false },
]

function isActive(currentPath: string, target: string) {
  // "/" is only ever active on exactly "/" — as a prefix it matches everything.
  if (target === "/") return currentPath === "/"
  return currentPath === target || currentPath.startsWith(`${target}/`)
}

export function AppSidebar() {
  const { pathname } = useLocation()
  const { isMinecraftAdmin } = useAuth()

  return (
    <aside className="hidden border-r-4 border-sidebar-border bg-sidebar text-sidebar-foreground lg:flex lg:min-h-svh lg:flex-col">
      <div className="flex h-16 items-center gap-3 border-b-4 border-sidebar-border px-4">
        <PixelGrassBlock className="size-7 shrink-0" />
        <div className="font-display text-[0.9rem] leading-none tracking-tight">WARDEN</div>
      </div>

      <nav className="flex-1 space-y-1.5 p-3">
        {navItems
          .filter((item) => !item.adminOnly || isMinecraftAdmin)
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
