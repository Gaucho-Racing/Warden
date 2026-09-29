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
  created_at: string
  updated_at: string
  identity?: Identity
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

export function useAuditLogs() {
  return useQuery({
    queryKey: ["audit-logs"],
    queryFn: async () => (await api.get<AuditLog[]>("/audit-logs")).data,
  })
}

export function skinURL(uuid: string) {
  return `https://mc-heads.net/avatar/${uuid}`
}

export function errorMessage(error: unknown, fallback: string) {
  const response = (error as { response?: { data?: { error?: string } } })?.response
  return response?.data?.error ?? fallback
}

function isNotFound(error: unknown) {
  return (error as { response?: { status?: number } })?.response?.status === 404
}
