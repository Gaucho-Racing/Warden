/** 1284 -> 1,284 · 12873 -> 12.9K · 1200000 -> 1.2M */
export function compact(n: number) {
  if (n < 10_000) return n.toLocaleString()
  if (n < 1_000_000) return `${(n / 1_000).toFixed(1).replace(/\.0$/, "")}K`
  return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`
}
