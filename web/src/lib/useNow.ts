import { useEffect, useState } from "react"

/**
 * A clock that re-renders on an interval, for relative times that would
 * otherwise sit at "in 6h 12m" until something else happened to re-render.
 */
export function useNow(intervalMs = 1000) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), intervalMs)
    return () => clearInterval(timer)
  }, [intervalMs])
  return now
}
