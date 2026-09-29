import { cn } from "@/lib/utils"

/**
 * Stat tile: label, value, optional delta against a named period.
 *
 * Values use the app's heading face and proportional figures — tabular
 * figures give every digit the width of a zero, which reads loose at display
 * size. Reserve those for columns that must align.
 */
export function StatTile({
  label,
  value,
  unit,
  delta,
  deltaPeriod = "7d",
  upIsGood = true,
  className,
}: {
  label: string
  value: string
  unit?: string
  delta?: number
  deltaPeriod?: string
  upIsGood?: boolean
  className?: string
}) {
  const good = delta === undefined || delta === 0 ? null : delta > 0 === upIsGood

  return (
    <div className={cn("mc-bevel bg-card px-4 py-3", className)}>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1.5 flex items-baseline gap-1.5">
        <span className="font-pixel text-2xl leading-none">{value}</span>
        {unit && <span className="text-xs text-muted-foreground">{unit}</span>}
      </div>
      {delta !== undefined && (
        <div
          className={cn(
            "mt-1.5 font-mono text-[0.7rem]",
            good === null ? "text-muted-foreground" : good ? "text-primary" : "text-destructive",
          )}
        >
          {delta > 0 ? "+" : ""}
          {delta.toLocaleString()} <span className="text-muted-foreground">{deltaPeriod}</span>
        </div>
      )}
    </div>
  )
}

