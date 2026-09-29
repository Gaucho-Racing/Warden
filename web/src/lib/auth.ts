import { useQuery, useQueryClient } from "@tanstack/react-query"

import { api } from "@/lib/api"

const SESSION_KEY = "warden_session"

// Mirrors api.AdminGroup on the server. The server is the authority — this
// only decides what the UI bothers to render.
export const ADMIN_GROUP = "Admins"

export type Session = {
  accessToken: string
  refreshToken: string
  expiresIn: number
}

export type CurrentUser = {
  id: string
  entity_id: string
  username: string
  first_name: string
  last_name: string
  email: string
  phone_number: string
  gender: string
  birthday: string
  graduate_level: string
  graduation_year: number
  major: string
  shirt_size: string
  jacket_size: string
  sae_registration_number: string
  occupation_title: string
  occupation_company: string
  avatar_url: string
  initial_role: string
  groups: string[]
  updated_at: string
  created_at: string
}

export function saveSession(session: Session) {
  localStorage.setItem(SESSION_KEY, JSON.stringify(session))
}

export function loadSession(): Session | null {
  const raw = localStorage.getItem(SESSION_KEY)
  if (!raw) return null
  try {
    const session = JSON.parse(raw) as Partial<Session>
    if (!session.accessToken || !session.refreshToken) {
      localStorage.removeItem(SESSION_KEY)
      return null
    }
    return {
      accessToken: session.accessToken,
      refreshToken: session.refreshToken,
      expiresIn: Number(session.expiresIn) || 0,
    }
  } catch {
    localStorage.removeItem(SESSION_KEY)
    return null
  }
}

export function clearSession() {
  localStorage.removeItem(SESSION_KEY)
}

export function useAuth() {
  const queryClient = useQueryClient()
  const tokenSession = loadSession()
  const userQuery = useQuery({
    queryKey: ["currentUser", tokenSession?.accessToken],
    queryFn: async () => {
      const response = await api.get<CurrentUser>("/users/@me")
      return response.data
    },
    enabled: !!tokenSession?.accessToken,
    retry: false,
    staleTime: 5 * 60 * 1000,
  })

  function logout() {
    clearSession()
    queryClient.clear()
    window.location.href = "/auth/login"
  }

  return {
    user: userQuery.data,
    tokenSession,
    isLoading: userQuery.isLoading,
    isAuthenticated: !!tokenSession,
    isAdmin: !!userQuery.data?.groups?.includes(ADMIN_GROUP),
    refresh: () => queryClient.invalidateQueries({ queryKey: ["currentUser"] }),
    logout,
  }
}
