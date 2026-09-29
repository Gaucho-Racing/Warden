import { compact } from "@/lib/format"
import type { Counted } from "@/lib/warden"

/**
 * Horizontal bars for a Minecraft tally.
 *
 * One flat accent rather than a sequential ramp: bar length already encodes
 * magnitude, so a ramp would double-encode it, and the ramp's dark steps
 * measured 1.4:1 against this surface — those bars would be invisible.
 *
 * Square ends, because nothing in this app has a radius.
 */
export function TopCountsChart({ counts }: { counts: Counted[] }) {
  if (!counts || counts.length === 0) {
    return <p className="text-sm text-muted-foreground">Nothing recorded yet.</p>
  }
  const max = Math.max(...counts.map((c) => c.count))

  return (
    <div className="space-y-2">
      {counts.map((c) => (
        <div key={c.key} className="grid grid-cols-[8rem_minmax(0,1fr)_3.5rem] items-center gap-3">
          <span className="truncate font-mono text-xs text-muted-foreground" title={c.key}>
            {c.key.replace(/^minecraft:/, "").replace(/_/g, " ")}
          </span>
          <div className="mc-bevel-in h-5 bg-input p-[2px]">
            <div
              className="h-full bg-primary"
              style={{ width: `${Math.max((c.count / max) * 100, 2)}%` }}
            />
          </div>
          <span className="text-right font-mono text-xs tabular-nums">{compact(c.count)}</span>
        </div>
      ))}
    </div>
  )
}
