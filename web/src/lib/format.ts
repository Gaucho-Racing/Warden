/** 1284 -> 1,284 · 12873 -> 12.9K · 1200000 -> 1.2M */
export function compact(n: number) {
  if (n < 10_000) return n.toLocaleString()
  if (n < 1_000_000) return `${(n / 1_000).toFixed(1).replace(/\.0$/, "")}K`
  return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`
}

/** 0 -> 0 B · 1536 -> 1.5 KB · 3221225472 -> 3.0 GB */
export function bytes(n: number) {
  if (!n) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  let value = n
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  return `${unit === 0 ? value : value.toFixed(1)} ${units[unit]}`
}

/** 45000 -> 45s · 195000 -> 3m 15s · 7500000 -> 2h 5m */
export function duration(milliseconds: number) {
  const total = Math.max(0, Math.round(milliseconds / 1000))
  if (total < 60) return `${total}s`
  const minutes = Math.floor(total / 60)
  if (minutes < 60) {
    const seconds = total % 60
    return seconds ? `${minutes}m ${seconds}s` : `${minutes}m`
  }
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest ? `${hours}h ${rest}m` : `${hours}h`
}

/** Signed distance from now: "in 6h 12m", "3m ago", "now". */
export function relativeTime(target: Date, now: number = Date.now()) {
  const delta = target.getTime() - now
  const magnitude = Math.abs(delta)
  if (magnitude < 45_000) return delta >= 0 ? "now" : "just now"
  const text = duration(magnitude)
  return delta > 0 ? `in ${text}` : `${text} ago`
}

const WEEKDAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"]

/**
 * Plain English for the common cron shapes, or null when the expression is
 * more interesting than this is willing to describe.
 *
 * Deliberately best effort: the authoritative answer to "when does this
 * run" is the list of next run times the server computes, and this only
 * labels it. Anything it cannot phrase confidently falls back to showing
 * the raw expression, which is never wrong.
 */
export function describeCron(spec: string): string | null {
  const trimmed = spec.trim()
  const descriptors: Record<string, string> = {
    "@yearly": "Every year on 1 January, at midnight",
    "@annually": "Every year on 1 January, at midnight",
    "@monthly": "On the 1st of every month, at midnight",
    "@weekly": "Every Sunday at midnight",
    "@daily": "Every day at midnight",
    "@midnight": "Every day at midnight",
    "@hourly": "Every hour, on the hour",
  }
  if (descriptors[trimmed]) return descriptors[trimmed]
  if (trimmed.startsWith("@every ")) return `Every ${trimmed.slice(7).trim()}`

  const [minute, hour, dayOfMonth, month, dayOfWeek, ...extra] = trimmed.split(/\s+/)
  if (extra.length || !dayOfWeek) return null
  if (month !== "*") return null

  const everyMinutes = /^\*\/(\d+)$/.exec(minute)
  if (everyMinutes && hour === "*" && dayOfMonth === "*" && dayOfWeek === "*") {
    return `Every ${everyMinutes[1]} minutes`
  }

  const exactMinute = /^\d{1,2}$/.exec(minute)
  if (!exactMinute) return null

  const everyHours = /^\*\/(\d+)$/.exec(hour)
  if (everyHours && dayOfMonth === "*" && dayOfWeek === "*") {
    return `Every ${everyHours[1]} hours, at ${clock(0, Number(minute))} past`
  }
  if (!/^\d{1,2}$/.test(hour)) return null

  const at = clock(Number(hour), Number(minute))
  if (dayOfMonth === "*" && dayOfWeek === "*") return `Every day at ${at}`
  if (dayOfMonth === "*") {
    const days = dayOfWeek.split(",").map(weekdayName)
    if (days.some((day) => day === null)) return null
    return `Every ${joinList(days as string[])} at ${at}`
  }
  if (dayOfWeek === "*" && /^\d{1,2}$/.test(dayOfMonth)) {
    return `On the ${ordinal(Number(dayOfMonth))} of every month, at ${at}`
  }
  return null
}

function weekdayName(field: string): string | null {
  const named = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"].indexOf(
    field.slice(0, 3).toLowerCase(),
  )
  if (named >= 0) return WEEKDAYS[named]
  if (!/^\d$/.test(field)) return null
  // Cron accepts both 0 and 7 for Sunday.
  return WEEKDAYS[Number(field) % 7]
}

function clock(hour: number, minute: number) {
  if (hour > 23 || minute > 59) return `${hour}:${String(minute).padStart(2, "0")}`
  const date = new Date(2000, 0, 1, hour, minute)
  return date.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" })
}

function ordinal(n: number) {
  const suffix = n % 100 >= 11 && n % 100 <= 13 ? "th" : ["th", "st", "nd", "rd"][n % 10] ?? "th"
  return `${n}${suffix}`
}

function joinList(items: string[]) {
  if (items.length <= 1) return items[0] ?? ""
  if (items.length === 2) return `${items[0]} and ${items[1]}`
  return `${items.slice(0, -1).join(", ")} and ${items[items.length - 1]}`
}
