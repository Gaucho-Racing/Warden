import { cn } from "@/lib/utils"

/**
 * A Minecraft inventory slot holding a player head: recessed bevel, dark
 * well, nearest-neighbour scaling.
 *
 * The frame is a wrapper rather than a bevel on the <img> itself because an
 * inset box-shadow paints beneath replaced content — putting it on the image
 * renders nothing at all.
 */
export function SkinFrame({
  src,
  alt,
  className,
}: {
  src: string
  alt: string
  className?: string
}) {
  return (
    <div className={cn("mc-slot shrink-0 p-[3px]", className)}>
      <img src={src} alt={alt} data-pixel className="size-full object-cover" />
    </div>
  )
}
