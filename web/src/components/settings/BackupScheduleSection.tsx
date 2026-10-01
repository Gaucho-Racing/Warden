import { Loader2 } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { SchedulePreview } from "@/components/backups/SchedulePreview"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
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
import { useDebounced } from "@/lib/useDebounced"
import {
  errorMessage,
  useBackupSchedule,
  useSaveBackupSchedule,
  useSchedulePreview,
  type BackupSchedule,
} from "@/lib/warden"

// The zones anybody setting this schedule is plausibly in. Kept short on
// purpose: a full IANA list is 600 entries and the answer is almost always
// the first one.
const TIMEZONES = [
  "America/Los_Angeles",
  "America/Denver",
  "America/Chicago",
  "America/New_York",
  "UTC",
]

const PRESETS = [
  { label: "Daily, 4am", cron: "0 4 * * *" },
  { label: "Twice daily", cron: "0 4,16 * * *" },
  { label: "Every 6 hours", cron: "0 */6 * * *" },
  { label: "Weekly, Sunday 4am", cron: "0 4 * * 0" },
]

export function BackupScheduleSection() {
  const schedule = useBackupSchedule()

  return (
    <section className="space-y-3">
      <div>
        <h2 className="font-pixel text-xl">Backup schedule</h2>
        <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
          When the whole server is archived to Depot. Players are warned five minutes before a
          scheduled backup, and again when it starts and finishes.
        </p>
      </div>

      {schedule.isLoading && <Skeleton className="h-80 w-full rounded-xl" />}

      {schedule.isError && (
        <Card>
          <CardContent className="p-6 text-sm text-destructive">
            {errorMessage(schedule.error, "Could not load the backup schedule.")}
          </CardContent>
        </Card>
      )}

      {schedule.data && (
        <Card>
          <CardContent className="p-5">
            {/*
              Keyed on the saved values so a save of its own, or another
              admin's edit arriving, remounts the form onto the new baseline.
              Typing does not change the key, so nothing resets mid-edit.
            */}
            <ScheduleEditor
              key={`${schedule.data.cron}|${schedule.data.timezone}|${schedule.data.enabled}`}
              schedule={schedule.data}
            />
          </CardContent>
        </Card>
      )}
    </section>
  )
}

/** Draft state is seeded from the saved schedule; see the key above. */
function ScheduleEditor({ schedule }: { schedule: BackupSchedule }) {
  const [cron, setCron] = useState(schedule.cron)
  const [timezone, setTimezone] = useState(schedule.timezone)
  const [enabled, setEnabled] = useState(schedule.enabled)
  const save = useSaveBackupSchedule()

  // Debounced so the preview follows typing without a request per keystroke.
  const debouncedCron = useDebounced(cron, 300)
  const preview = useSchedulePreview(debouncedCron, timezone, enabled)

  const dirty =
    cron !== schedule.cron || timezone !== schedule.timezone || enabled !== schedule.enabled
  const previewSettled = debouncedCron === cron && !preview.isFetching
  const canSave = dirty && (!enabled || (previewSettled && preview.data?.valid === true))

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
        <div className="space-y-2">
          <Label htmlFor="backup-cron">Cron expression</Label>
          <Input
            id="backup-cron"
            value={cron}
            spellCheck={false}
            autoComplete="off"
            onChange={(event) => setCron(event.target.value)}
            className="font-mono"
            placeholder="0 4 * * *"
          />
          <p className="text-xs text-muted-foreground">
            Five fields: minute, hour, day of month, month, day of week.
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="backup-timezone">Timezone</Label>
          <Select value={timezone} onValueChange={setTimezone}>
            <SelectTrigger id="backup-timezone" className="sm:w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {/* A zone saved before this list existed still has to be selectable. */}
              {(TIMEZONES.includes(timezone) ? TIMEZONES : [timezone, ...TIMEZONES]).map((zone) => (
                <SelectItem key={zone} value={zone}>
                  {zone}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        {PRESETS.map((preset) => (
          <Button
            key={preset.cron}
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => {
              setCron(preset.cron)
              setEnabled(true)
            }}
          >
            {preset.label}
          </Button>
        ))}
      </div>

      <SchedulePreview
        cron={cron}
        timezone={timezone}
        runs={preview.data?.valid ? preview.data.next_runs : []}
        error={previewSettled ? preview.data?.error : undefined}
        pending={!previewSettled}
        disabled={!enabled}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <Button
          type="button"
          variant={enabled ? "secondary" : "default"}
          onClick={() => setEnabled((current) => !current)}
        >
          {enabled ? "Turn off scheduled backups" : "Turn on scheduled backups"}
        </Button>
        <div className="flex items-center gap-2">
          {dirty && <span className="text-xs text-muted-foreground">Unsaved changes</span>}
          <Button
            type="button"
            disabled={!canSave || save.isPending}
            onClick={() =>
              save.mutate(
                { cron, timezone, enabled },
                {
                  onSuccess: () => toast.success("Backup schedule saved"),
                  onError: (error) =>
                    toast.error(errorMessage(error, "Could not save the schedule")),
                },
              )
            }
          >
            {save.isPending && <Loader2 className="size-4 animate-spin" />}
            Save schedule
          </Button>
        </div>
      </div>
    </div>
  )
}
