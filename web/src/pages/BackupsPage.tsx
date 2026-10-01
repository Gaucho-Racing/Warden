import { CircleAlert, HardDriveDownload, Loader2 } from "lucide-react"
import { useState } from "react"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { PageContainer, PageHeader } from "@/components/PageContainer"
import { SchedulePreview } from "@/components/backups/SchedulePreview"
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
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/lib/auth"
import { bytes, duration, relativeTime, relativeTimeCoarse } from "@/lib/format"
import { useNow } from "@/lib/useNow"
import { cn } from "@/lib/utils"
import {
  backupIsActive,
  errorMessage,
  useBackups,
  useStartBackup,
  type BackupJob,
  type BackupStatus,
} from "@/lib/warden"

const STATUS_LABELS: Record<BackupStatus, string> = {
  pending: "Starting",
  archiving: "Archiving",
  uploading: "Uploading",
  succeeded: "Succeeded",
  failed: "Failed",
  timed_out: "Timed out",
}

export default function BackupsPage() {
  const { isMinecraftAdmin } = useAuth()
  const backups = useBackups()
  // A minute is as precise as finished backups are reported, so the whole
  // history does not re-render every second. The in-progress card keeps its
  // own faster clock.
  const now = useNow(60_000)
  const [confirming, setConfirming] = useState(false)

  if (backups.isLoading) {
    return (
      <PageContainer>
        <Skeleton className="h-28 w-full" />
        <Skeleton className="mt-4 h-48 w-full" />
      </PageContainer>
    )
  }

  if (backups.isError || !backups.data) {
    return (
      <PageContainer>
        <PageHeader title="Backups" />
        <Card>
          <CardContent className="p-6 text-sm text-destructive">
            {errorMessage(backups.error, "Could not load backups.")}
          </CardContent>
        </Card>
      </PageContainer>
    )
  }

  const { enabled, bucket, schedule, server_connected, active, last, jobs } = backups.data

  return (
    <PageContainer>
      <PageHeader
        title="Backups"
        description={`The whole server, archived and uploaded to the ${bucket} bucket in Depot.`}
        action={
          isMinecraftAdmin && (
            <Button
              disabled={!enabled || !server_connected || !!active}
              onClick={() => setConfirming(true)}
            >
              <HardDriveDownload className="size-4" />
              Back up now
            </Button>
          )
        }
      />

      {!enabled && (
        <Notice>
          Backups are not configured on this deployment, so nothing is being archived.
        </Notice>
      )}
      {enabled && !server_connected && (
        <Notice>
          The game server is not connected. It is the only thing that can read the world files, so
          backups cannot run until it comes back.
        </Notice>
      )}

      <div className="space-y-8">
        {active ? (
          <ActiveBackup job={active} />
        ) : (
          <div className="space-y-2">
            <SchedulePreview
              cron={schedule.cron}
              timezone={schedule.timezone}
              runs={schedule.next_runs}
              error={schedule.error}
              disabled={!schedule.enabled}
            />
            {/* The editor lives in Settings with the other admin controls,
                so say where it went rather than leaving a dead end. */}
            {isMinecraftAdmin && (
              <div className="text-right">
                <Button asChild variant="link" className="h-auto p-0 text-xs">
                  <Link to="/settings">Change the schedule in Settings</Link>
                </Button>
              </div>
            )}
          </div>
        )}

        {last && (
          <section className="space-y-3">
            <h2 className="font-pixel text-xl">Last backup</h2>
            <JobCard job={last} now={now} />
          </section>
        )}

        <Separator />

        <section className="space-y-3">
          <h2 className="font-pixel text-xl">History</h2>
          {jobs.length === 0 ? (
            <Card>
              <CardContent className="p-6 text-sm text-muted-foreground">
                No backups have run yet.
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-2">
              {jobs.map((job) => (
                <JobCard key={job.id} job={job} now={now} />
              ))}
            </div>
          )}
        </section>
      </div>

      <ConfirmBackupDialog open={confirming} onOpenChange={setConfirming} />
    </PageContainer>
  )
}

function Notice({ children }: { children: React.ReactNode }) {
  return (
    <Card className="mb-6 border-destructive/50">
      <CardContent className="flex items-start gap-3 p-4 text-sm">
        <CircleAlert className="mt-0.5 size-4 shrink-0 text-destructive" />
        <span className="text-muted-foreground">{children}</span>
      </CardContent>
    </Card>
  )
}

/**
 * A running backup replaces the schedule preview rather than sitting beside
 * it: while one is in flight, when the next one starts is not the question
 * anybody on this page is asking.
 */
function ActiveBackup({ job }: { job: BackupJob }) {
  // The only place a second hand earns its keep: this is the one thing on
  // the page actively changing, and it is watched while it runs.
  const now = useNow(1000)
  // A job still warning players has not started yet, so it carries no
  // started_at and the row's own creation time is the honest answer.
  const started = new Date(job.started_at ?? job.created_at)
  const phase =
    job.status === "uploading"
      ? `Uploading ${job.size_bytes ? bytes(job.size_bytes) : "the archive"} to Depot`
      : job.status === "archiving"
        ? "Compressing the server files"
        : "Warning players"

  return (
    <div className="mc-bevel bg-card p-4">
      <div className="mb-3 flex items-center gap-2 text-xs tracking-wide text-muted-foreground uppercase">
        <Loader2 className="size-3.5 animate-spin" />
        Backup in progress
      </div>
      <div className="font-pixel text-2xl leading-none">{phase}</div>
      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 font-mono text-xs text-muted-foreground">
        <span>started {relativeTime(started, now)}</span>
        <span>{job.trigger === "manual" ? "manual" : "scheduled"}</span>
        {job.archive_millis > 0 && <span>archived in {duration(job.archive_millis)}</span>}
      </div>
    </div>
  )
}

function JobCard({ job, now }: { job: BackupJob; now: number }) {
  const finished = job.finished_at ? new Date(job.finished_at) : null
  const total =
    job.archive_millis + job.upload_millis ||
    (finished && job.started_at ? finished.getTime() - new Date(job.started_at).getTime() : 0)

  return (
    <Card>
      <CardContent className="flex flex-wrap items-center gap-x-4 gap-y-2 p-4">
        <StatusBadge status={job.status} />
        <div className="min-w-0 flex-1">
          <div className="truncate font-mono text-sm">{job.file_name || job.id}</div>
          <div className="truncate font-mono text-xs text-muted-foreground">{job.id}</div>
        </div>
        <Figure label="Type" value={job.trigger === "manual" ? "Manual" : "Scheduled"} />
        {job.status === "succeeded" && (
          <>
            <Figure label="Size" value={bytes(job.size_bytes)} />
            <Figure label="Took" value={total ? duration(total) : "—"} />
          </>
        )}
        <Figure label="When" value={relativeTimeCoarse(new Date(job.created_at), now)} />
        {/* Full width on its own line: a message competing with four
            figures for the same row wraps to something unreadable. */}
        {job.error && (
          <div className="min-w-0 basis-full text-xs text-destructive">{job.error}</div>
        )}
      </CardContent>
    </Card>
  )
}

function StatusBadge({ status }: { status: BackupStatus }) {
  const failed = status === "failed" || status === "timed_out"
  return (
    <Badge
      variant={failed ? "destructive" : status === "succeeded" ? "default" : "secondary"}
      className={cn("font-mono", backupIsActive(status) && "animate-pulse")}
    >
      {STATUS_LABELS[status]}
    </Badge>
  )
}

/**
 * One labelled figure in a job card's right-hand run.
 *
 * min-w rather than a fixed w: "Scheduled" is a good deal wider than
 * "Manual", and a fixed column would wrap it onto a second line and make
 * that one row taller than its neighbours.
 */
function Figure({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-24 text-right whitespace-nowrap">
      <div className="font-pixel text-base leading-tight">{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  )
}

function ConfirmBackupDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const start = useStartBackup()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Back up the server now?</DialogTitle>
          <DialogDescription>
            Players get a fifteen second warning, then the world is flushed to disk and frozen while
            the archive is written. Expect a few seconds of lag. The upload runs afterwards and
            does not affect the game.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={start.isPending}
            onClick={() =>
              start.mutate(undefined, {
                onSuccess: () => {
                  toast.success("Backup starting in 15 seconds")
                  onOpenChange(false)
                },
                onError: (error) =>
                  toast.error(errorMessage(error, "Could not start a backup")),
              })
            }
          >
            {start.isPending && <Loader2 className="size-4 animate-spin" />}
            Start backup
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
