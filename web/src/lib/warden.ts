import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { api } from "@/lib/api"

export type Identity = {
  entity_id: string
  name: string
  username: string
  avatar_url: string
}

export type MinecraftAccount = {
  uuid: string
  entity_id: string
  username: string
  linked_via_token_id: string
  last_seen_at: string
  avatar_url: string
  created_at: string
  updated_at: string
  identity?: Identity
  stats?: {
    playtime_minutes: number
    sessions: number
  }
}

export type LinkTokenPreview = {
  token: string
  uuid: string
  username: string
  avatar_url: string
  expires_at: string
}

export type GroupPermissionBinding = {
  id: string
  group_id: string
  group_name: string
  luckperms_group: string
  permissions: string[]
  weight: number
  created_by_entity_id: string
  updated_by_entity_id: string
  created_at: string
  updated_at: string
}

export type BindingInput = {
  group_id: string
  group_name: string
  luckperms_group?: string
  permissions: string[]
  weight: number
}

export type SentinelGroup = {
  id: string
  name: string
  description: string
  member_count: number
}

export type Counted = {
  key: string
  count: number
}

export type StatWindow = {
  playtime_minutes: number
  deaths: number
  mob_kills: number
  blocks_mined: number
  distance_meters: number
}

export type PlayerStats = {
  uuid: string
  username: string
  /** "none" means linked but never reported on — not zeroes as fact. */
  source: "plugin" | "none"
  playtime_minutes: number
  deaths: number
  mob_kills: number
  player_kills: number
  blocks_mined: number
  items_crafted: number
  distance_meters: number
  damage_dealt: number
  damage_taken: number
  jumps: number
  times_slept: number
  villager_trades: number
  raid_wins: number
  sessions: number
  top_blocks: Counted[]
  top_mobs: Counted[]
  top_crafted: Counted[]
  first_seen: string
  last_seen: string
  reported_at: string
  last_7_days?: StatWindow
}

export type AuditLog = {
  id: string
  action: string
  actor_entity_id: string
  minecraft_uuid: string
  minecraft_name: string
  target_id: string
  detail: string
  created_at: string
}

export type ServerState = "active" | "empty" | "offline"

export type ServerStatus = {
  state: ServerState
  online: number
  max_players: number
  unique_players: number
  uptime_minutes: number
  tps: number
  recorded_at?: string
}

// The game server reports every minute; polling a little faster than that
// keeps the header within a sample of the Discord topic.
export function useServerStatus() {
  return useQuery({
    queryKey: ["server", "status"],
    queryFn: async () => (await api.get<ServerStatus>("/server/status")).data,
    refetchInterval: 30_000,
    retry: false,
  })
}

export function useMyAccount() {
  return useQuery({
    queryKey: ["account", "@me"],
    queryFn: async () => {
      try {
        const response = await api.get<MinecraftAccount>("/accounts/@me")
        return response.data
      } catch (error) {
        // A member with no Minecraft account linked yet is the expected
        // starting state, not an error worth surfacing.
        if (isNotFound(error)) return null
        throw error
      }
    },
    retry: false,
  })
}

export function useAccounts() {
  return useQuery({
    queryKey: ["accounts"],
    queryFn: async () => (await api.get<MinecraftAccount[]>("/accounts")).data,
  })
}

export function useUnlinkAccount() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (uuid: string) => {
      await api.delete(`/accounts/${uuid}`)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["accounts"] })
      void queryClient.invalidateQueries({ queryKey: ["account", "@me"] })
    },
  })
}

export function useLinkToken(token: string | undefined) {
  return useQuery({
    queryKey: ["link-token", token],
    queryFn: async () => (await api.get<LinkTokenPreview>(`/link/${token}`)).data,
    enabled: !!token,
    retry: false,
  })
}

export function useConfirmLink() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (token: string) =>
      (await api.post<MinecraftAccount>(`/link/${token}`)).data,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["account", "@me"] })
    },
  })
}

export function useBindings() {
  return useQuery({
    queryKey: ["bindings"],
    queryFn: async () => (await api.get<GroupPermissionBinding[]>("/bindings")).data,
  })
}

export function useSentinelGroups() {
  return useQuery({
    queryKey: ["sentinel-groups"],
    queryFn: async () => (await api.get<SentinelGroup[]>("/groups")).data,
  })
}

export function useCreateBinding() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: BindingInput) =>
      (await api.post<GroupPermissionBinding>("/bindings", input)).data,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["bindings"] }),
  })
}

export function useUpdateBinding() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: BindingInput }) =>
      (await api.put<GroupPermissionBinding>(`/bindings/${id}`, input)).data,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["bindings"] }),
  })
}

export function useDeleteBinding() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/bindings/${id}`)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["bindings"] }),
  })
}

// A player's account and stats by UUID, for the player details page. Both
// answer 404 for a UUID nobody has linked, which the page shows as
// "not found" rather than an error.
export function usePlayerAccount(uuid: string | undefined) {
  return useQuery({
    queryKey: ["account", uuid],
    queryFn: async () => {
      try {
        return (await api.get<MinecraftAccount>(`/accounts/${uuid}`)).data
      } catch (error) {
        if (isNotFound(error)) return null
        throw error
      }
    },
    enabled: !!uuid,
    retry: false,
  })
}

export function usePlayerStats(uuid: string | undefined) {
  return useQuery({
    queryKey: ["stats", uuid],
    queryFn: async () => {
      try {
        return (await api.get<PlayerStats>(`/stats/${uuid}`)).data
      } catch (error) {
        if (isNotFound(error)) return null
        throw error
      }
    },
    enabled: !!uuid,
    retry: false,
  })
}

export function useMyStats() {
  return useQuery({
    queryKey: ["stats", "@me"],
    queryFn: async () => {
      try {
        return (await api.get<PlayerStats>("/stats/@me")).data
      } catch (error) {
        // No linked account is the onboarding state, not a failure.
        if (isNotFound(error)) return null
        throw error
      }
    },
    retry: false,
  })
}

export function useAuditLogs() {
  return useQuery({
    queryKey: ["audit-logs"],
    queryFn: async () => (await api.get<AuditLog[]>("/audit-logs")).data,
  })
}

export type BackupStatus =
  | "pending"
  | "archiving"
  | "uploading"
  | "succeeded"
  | "failed"
  | "timed_out"

export type BackupJob = {
  id: string
  status: BackupStatus
  trigger: "schedule" | "manual"
  requested_by_entity_id: string
  scheduled_for?: string
  depot_file_id: string
  depot_bucket: string
  file_name: string
  size_bytes: number
  archive_millis: number
  upload_millis: number
  error?: string
  /** Absent until the job leaves "pending" — see ActiveBackup. */
  started_at?: string
  finished_at?: string
  created_at: string
}

export type BackupSchedule = {
  cron: string
  timezone: string
  enabled: boolean
  updated_by_entity_id: string
  updated_at: string
  next_runs: string[]
  error?: string
}

export type BackupOverview = {
  enabled: boolean
  bucket: string
  schedule: BackupSchedule
  server_connected: boolean
  active?: BackupJob
  last?: BackupJob
  jobs: BackupJob[]
}

export type SchedulePreview = {
  cron: string
  timezone: string
  valid: boolean
  error?: string
  next_runs: string[]
}

export function backupIsActive(status: BackupStatus) {
  return status === "pending" || status === "archiving" || status === "uploading"
}

// Polls fast while a backup is in flight so the phase and the finish land
// without a refresh, and slowly otherwise — the next scheduled run only
// moves once a day.
export function useBackups() {
  return useQuery({
    queryKey: ["backups"],
    queryFn: async () => (await api.get<BackupOverview>("/backups")).data,
    refetchInterval: (query) =>
      query.state.data?.active ? 3_000 : 30_000,
  })
}

// The schedule on its own, for the settings editor. Keyed under "backups"
// so saving, which invalidates that prefix, refreshes both this and the
// backups page without either knowing about the other.
export function useBackupSchedule() {
  return useQuery({
    queryKey: ["backups", "schedule"],
    queryFn: async () => (await api.get<BackupSchedule>("/backups/schedule")).data,
  })
}

// Previews on the server rather than parsing cron in the browser, so what
// the field shows is exactly what the scheduler will do.
export function useSchedulePreview(cron: string, timezone: string, enabled: boolean) {
  return useQuery({
    queryKey: ["backups", "preview", cron, timezone],
    queryFn: async () =>
      (
        await api.get<SchedulePreview>("/backups/schedule/preview", {
          params: { cron, timezone, count: 3 },
        })
      ).data,
    enabled: enabled && cron.trim().length > 0,
    // A half-typed expression is answered with valid=false, not an error,
    // so there is nothing worth retrying.
    retry: false,
    placeholderData: (previous) => previous,
  })
}

export function useSaveBackupSchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (input: { cron: string; timezone: string; enabled: boolean }) =>
      (await api.put<BackupSchedule>("/backups/schedule", input)).data,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["backups"] }),
  })
}

export function useStartBackup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () => (await api.post<BackupJob>("/backups")).data,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["backups"] }),
  })
}

export function errorMessage(error: unknown, fallback: string) {
  const response = (error as { response?: { data?: { error?: string } } })?.response
  return response?.data?.error ?? fallback
}

function isNotFound(error: unknown) {
  return (error as { response?: { status?: number } })?.response?.status === 404
}
