import { Navigate, createBrowserRouter } from "react-router-dom"

import { AppShell } from "@/components/AppShell"
import { RequireAuth } from "@/components/RequireAuth"
import HomePage from "@/pages/HomePage"
import LinkPage from "@/pages/LinkPage"
import LoginPage from "@/pages/LoginPage"
import NotFoundPage from "@/pages/NotFoundPage"
import PlayersPage from "@/pages/PlayersPage"
import SettingsPage from "@/pages/SettingsPage"

export const router = createBrowserRouter([
  {
    element: <RequireAuth />,
    children: [
      // The link confirmation lives outside the app shell: a player arriving
      // from in-game chat should land on one focused decision, not a
      // dashboard they have no use for.
      { path: "/link/:token", element: <LinkPage /> },
      {
        element: <AppShell />,
        children: [
          { path: "/", element: <HomePage /> },
          { path: "/players", element: <PlayersPage /> },
          { path: "/settings", element: <SettingsPage /> },
          // Both folded into Settings; keep the old paths working.
          { path: "/account", element: <Navigate to="/settings" replace /> },
          { path: "/bindings", element: <Navigate to="/settings" replace /> },
        ],
      },
    ],
  },
  { path: "/auth/login", element: <LoginPage /> },
  { path: "*", element: <NotFoundPage /> },
])
