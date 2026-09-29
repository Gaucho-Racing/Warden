import { LogOut, Menu, Settings } from "lucide-react"
import { Link, useLocation, useNavigate } from "react-router-dom"

import {
  PixelGear,
  PixelGrassBlock,
  PixelHead,
  PixelHouse,
  PixelHeads,
  PixelKey,
} from "@/components/icons/pixel"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/lib/auth"
import { cn } from "@/lib/utils"

const mobileItems = [
  { to: "/", label: "Home", icon: PixelHouse, adminOnly: false },
  { to: "/account", label: "My Account", icon: PixelHead, adminOnly: false },
  { to: "/players", label: "Players", icon: PixelHeads, adminOnly: true },
  { to: "/bindings", label: "Bindings", icon: PixelKey, adminOnly: true },
  { to: "/settings", label: "Settings", icon: PixelGear, adminOnly: false },
]

function sectionTitle(pathname: string) {
  if (pathname === "/") return "Home"
  if (pathname.startsWith("/settings")) return "Settings"
  if (pathname.startsWith("/bindings")) return "Bindings"
  if (pathname.startsWith("/players")) return "Players"
  if (pathname.startsWith("/account")) return "My Account"
  return "Warden"
}

function initials(name: string) {
  return name
    .split(" ")
    .map((part) => part[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase()
}

function HeaderUserMenu() {
  const navigate = useNavigate()
  const { user, isLoading, logout } = useAuth()

  if (isLoading || !user) {
    return <Skeleton className="size-8 rounded-full" />
  }

  const name = `${user.first_name} ${user.last_name}`.trim() || user.username || "Warden user"
  const email = user.email || user.username

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="rounded-full outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring/35 focus-visible:ring-offset-2">
          <span className="mc-slot inline-flex p-[3px]">
            <Avatar className="size-8 cursor-pointer">
              <AvatarImage src={user.avatar_url} alt={name} />
              <AvatarFallback className="font-pixel text-xs">{initials(name)}</AvatarFallback>
            </Avatar>
          </span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={10} className="w-56">
        <DropdownMenuLabel className="flex flex-col">
          <span className="text-sm font-medium">{name}</span>
          <span className="text-xs font-normal text-muted-foreground">{email}</span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => navigate("/settings")}>
          <Settings className="size-4" />
          Settings
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={logout} className="text-destructive focus:text-destructive">
          <LogOut className="size-4" />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function AppHeader() {
  const { pathname } = useLocation()
  const { isAdmin } = useAuth()
  const section = sectionTitle(pathname)

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center gap-3 border-b-4 border-border bg-background px-4 lg:px-6">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="icon" className="lg:hidden">
            <Menu className="size-4" />
            <span className="sr-only">Open navigation</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-52">
          <DropdownMenuLabel className="font-display text-[0.6rem]">WARDEN</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {mobileItems
            .filter((item) => !item.adminOnly || isAdmin)
            .map((item) => (
              <DropdownMenuItem key={item.to} asChild>
                <Link to={item.to} className={cn(
                    (item.to === "/" ? pathname === "/" : pathname.startsWith(item.to)) &&
                      "text-primary",
                  )}>
                  <item.icon className="size-4 shrink-0" />
                  {item.label}
                </Link>
              </DropdownMenuItem>
            ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <Link to="/" className="flex items-center gap-2 lg:hidden">
        <PixelGrassBlock className="size-6" />
        <span className="font-display text-[0.8rem]">WARDEN</span>
      </Link>

      <div className="hidden min-w-0 lg:block">
        <div className="font-pixel text-xl leading-none">{section}</div>
      </div>

      <div className="flex-1" />

      <HeaderUserMenu />
    </header>
  )
}
