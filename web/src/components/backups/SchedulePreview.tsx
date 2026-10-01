import { AlertTriangle, Clock } from "lucide-react"

import { describeCron, relativeTime } from "@/lib/format"
import { cn } from "@/lib/utils"
import { useNow } from "@/lib/useNow"

/**
 * Shows what a cron expression actually means, as the next few times it
 * would fire.
 *
 * The times come from the server, which evaluates them with the same parser
 * the scheduler uses. A cron implementation in the browser would eventually
 * disagree with it — over a DST boundary, or on day-of-month versus
 * day-of-week — and a preview that is confidently wrong is worse than none.
 */
export function SchedulePreview({
  cron,
  timezone,
  runs,
  error,
  pending,
  disabled,
}: {
  cron: string
  timezone: string
  runs: string[]
  error?: string
  pending?: boolean
  disabled?: boolean
}) {
  const now = useNow(1000)
  const summary = describeCron(cron)

  if (disabled) {
    return (
      <Shell>
        <p className="text-sm text-muted-foreground">
          Scheduled backups are off. Turn them on to see when they would run.
        </p>
      </Shell>
    )
  }

  if (error) {
    return (
      <Shell>
        <div className="flex items-start gap-2 text-sm text-destructive">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          <span>{error}</span>
        </div>
      </Shell>
    )
  }

  return (
    <Shell>
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <div className="font-pixel text-lg leading-none">
          {summary ?? <span className="font-mono text-base">{cron}</span>}
        </div>
        <div className="font-mono text-xs text-muted-foreground">{timezone}</div>
      </div>

      <div className={cn("mt-5", pending && "opacity-50 transition-opacity")}>
        {runs.length === 0 ? (
          <p className="text-sm text-muted-foreground">No upcoming runs.</p>
        ) : (
          <ol className="grid gap-px sm:grid-cols-3">
            {runs.map((run, index) => (
              <Stop key={run} at={new Date(run)} now={now} index={index} />
            ))}
          </ol>
        )}
      </div>
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="mc-bevel bg-card p-4">
      <div className="mb-3 flex items-center gap-2 text-xs tracking-wide text-muted-foreground uppercase">
        <Clock className="size-3.5" />
        Next backups
      </div>
      {children}
    </div>
  )
}

/**
 * One upcoming run. The track runs behind the markers rather than between
 * them so the three read as points on one timeline instead of three
 * unrelated cards, and the first is filled because it is the one that
 * actually matters.
 */
function Stop({ at, now, index }: { at: Date; now: number; index: number }) {
  const next = index === 0
  return (
    <li className="relative pt-6">
      <div className="absolute top-[0.6875rem] right-0 left-0 h-0.5 bg-border" aria-hidden />
      <div
        className={cn(
          "absolute top-1.5 left-0 size-3.5",
          next ? "mc-bevel bg-primary" : "mc-bevel-thin bg-muted",
        )}
        style={
          next
            ? ({
                "--mc-bevel-light": "color-mix(in oklab, var(--primary) 62%, white)",
                "--mc-bevel-dark": "color-mix(in oklab, var(--primary) 60%, black)",
              } as React.CSSProperties)
            : undefined
        }
        aria-hidden
      />
      <div className="pr-4">
        <div className="text-xs text-muted-foreground">
          {at.toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" })}
        </div>
        <div className={cn("font-pixel text-2xl leading-tight", !next && "text-muted-foreground")}>
          {at.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" })}
        </div>
        <div className={cn("font-mono text-xs", next ? "text-primary" : "text-muted-foreground")}>
          {relativeTime(at, now)}
        </div>
      </div>
    </li>
  )
}
