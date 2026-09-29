import { Link } from "react-router-dom"

import { PixelGrassBlock } from "@/components/icons/pixel"
import { Button } from "@/components/ui/button"

export default function NotFoundPage() {
  return (
    <main className="mc-backdrop flex min-h-svh flex-col items-center justify-center gap-6 px-4 py-12">
      <PixelGrassBlock className="size-20" />
      <div className="space-y-2 text-center">
        <h1 className="font-pixel text-3xl text-white drop-shadow-[2px_2px_0_rgba(0,0,0,0.8)]">
          Chunk not found
        </h1>
        <p className="font-pixel text-base text-white/70 drop-shadow-[1px_1px_0_rgba(0,0,0,0.8)]">
          There&apos;s nothing generated at this coordinate.
        </p>
      </div>
      <Button asChild size="lg">
        <Link to="/account">Back to spawn</Link>
      </Button>
    </main>
  )
}
