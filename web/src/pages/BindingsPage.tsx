import { Loader2, Plus, Trash2 } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { PageContainer, PageHeader } from "@/components/PageContainer"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Textarea } from "@/components/ui/textarea"
import {
  errorMessage,
  useBindings,
  useCreateBinding,
  useDeleteBinding,
  useSentinelGroups,
} from "@/lib/warden"

export default function BindingsPage() {
  const bindings = useBindings()
  const [creating, setCreating] = useState(false)

  return (
    <PageContainer>
      <PageHeader
        title="Bindings"
        description="What each Sentinel group grants in Minecraft. Warden only manages the LuckPerms groups it creates — anything granted by hand in-game is left alone."
        action={
          <Button onClick={() => setCreating(true)}>
            <Plus className="size-4" />
            New binding
          </Button>
        }
      />

      {bindings.isLoading && <Skeleton className="h-40 w-full rounded-xl" />}

      {bindings.data?.length === 0 && (
        <Card>
          <CardContent className="p-6 text-sm text-muted-foreground">
            No bindings yet. Until one exists, no linked player receives any permission.
          </CardContent>
        </Card>
      )}

      <div className="space-y-2">
        {bindings.data?.map((binding) => (
          <BindingRow key={binding.id} binding={binding} />
        ))}
      </div>

      <BindingDialog open={creating} onOpenChange={setCreating} />
    </PageContainer>
  )
}

function BindingRow({
  binding,
}: {
  binding: NonNullable<ReturnType<typeof useBindings>["data"]>[number]
}) {
  const remove = useDeleteBinding()

  return (
    <Card>
      <CardContent className="flex flex-wrap items-center gap-4 p-4">
        <div className="min-w-0 flex-1">
          <div className="font-pixel text-lg leading-tight">{binding.group_name}</div>
          <div className="font-mono text-xs text-muted-foreground">{binding.group_id}</div>
        </div>
        <div className="min-w-0 flex-1">
          <Badge variant="secondary" className="font-mono">
            {binding.luckperms_group}
          </Badge>
          {binding.permissions.length > 0 && (
            <div className="mt-1 font-mono text-xs text-muted-foreground">
              {binding.permissions.join(", ")}
            </div>
          )}
        </div>
        <div className="mc-bevel-in bg-input px-2 py-1 font-mono text-xs text-muted-foreground">w{binding.weight}</div>
        <Button
          variant="ghost"
          size="icon"
          disabled={remove.isPending}
          onClick={() =>
            remove.mutate(binding.id, {
              onSuccess: () => toast.success(`Removed binding for ${binding.group_name}`),
              onError: (error) => toast.error(errorMessage(error, "Could not remove binding")),
            })
          }
        >
          {remove.isPending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Trash2 className="size-4" />
          )}
          <span className="sr-only">Delete binding</span>
        </Button>
      </CardContent>
    </Card>
  )
}

function BindingDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const groups = useSentinelGroups()
  const create = useCreateBinding()
  const [groupID, setGroupID] = useState("")
  const [permissions, setPermissions] = useState("")
  const [weight, setWeight] = useState("0")

  const selected = groups.data?.find((group) => group.id === groupID)

  function submit() {
    if (!selected) return
    create.mutate(
      {
        group_id: selected.id,
        group_name: selected.name,
        // Left blank so the server derives the warden- prefixed LuckPerms
        // group. Keeping that rule in one place avoids a client that can
        // name a group Warden won't recognize as its own.
        permissions: permissions
          .split(/[\s,]+/)
          .map((node) => node.trim())
          .filter(Boolean),
        weight: Number(weight) || 0,
      },
      {
        onSuccess: () => {
          toast.success(`Bound ${selected.name}`)
          setGroupID("")
          setPermissions("")
          setWeight("0")
          onOpenChange(false)
        },
        onError: (error) => toast.error(errorMessage(error, "Could not create binding")),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New binding</DialogTitle>
          <DialogDescription>
            Members of the Sentinel group receive the matching LuckPerms group the next time
            Warden reconciles.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Sentinel group</Label>
            <Select value={groupID} onValueChange={setGroupID}>
              <SelectTrigger>
                <SelectValue placeholder="Select a group" />
              </SelectTrigger>
              <SelectContent>
                {groups.data?.map((group) => (
                  <SelectItem key={group.id} value={group.id}>
                    {group.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>Extra permission nodes</Label>
            <Textarea
              value={permissions}
              onChange={(event) => setPermissions(event.target.value)}
              placeholder="essentials.fly, worldedit.*"
              className="font-mono text-xs"
            />
            <p className="text-xs text-muted-foreground">
              Optional. Comma or whitespace separated, applied on top of the LuckPerms group.
            </p>
          </div>

          <div className="space-y-2">
            <Label>Weight</Label>
            <Input
              value={weight}
              onChange={(event) => setWeight(event.target.value)}
              inputMode="numeric"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={!selected || create.isPending} onClick={submit}>
            {create.isPending && <Loader2 className="size-4 animate-spin" />}
            Create binding
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
