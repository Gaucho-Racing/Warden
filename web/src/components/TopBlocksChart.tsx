import { compact } from "@/lib/format"
import type { BlockCount } from "@/lib/warden"


/**
 * Horizontal bars for the mined tally.
 *
 * One flat accent rather than a sequential ramp: bar length already encodes
 * magnitude, so a ramp would double-encode it, and the ramp's dark steps
 * measured 1.4:1 against this surface — those bars would have been invisible.
 *
 * Square ends rather than the usual rounded data-ends, because nothing in
 * this app has a radius.
 */
export function TopBlocksChart({ blocks }: { blocks: BlockCount[] }) {
  if (blocks.length === 0) return null
  const max = Math.max(...blocks.map((b) => b.count))

  return (
    <div className="space-y-2">
      {blocks.map((b) => (
        <div key={b.block} className="grid grid-cols-[9rem_minmax(0,1fr)_4rem] items-center gap-3">
          <span className="truncate font-mono text-xs text-muted-foreground">
            {b.block.replace("minecraft:", "")}
          </span>
          <div className="mc-bevel-in h-5 bg-input p-[2px]">
            <div
              className="h-full bg-primary"
              style={{ width: `${Math.max((b.count / max) * 100, 2)}%` }}
            />
          </div>
          <span className="text-right font-mono text-xs tabular-nums">{compact(b.count)}</span>
        </div>
      ))}
    </div>
  )
}
