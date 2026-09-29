import type { SVGProps } from "react"

/**
 * Renders a character grid as an SVG of 1x1 rects. Every glyph in the app is
 * authored as ASCII so the shapes stay editable by eye — the alternative,
 * hand-written path data, is unreadable at this resolution.
 *
 * "." is always transparent. Everything else is looked up in `palette`;
 * "currentColor" lets an icon inherit nav active/inactive states.
 */
function PixelArt({
  grid,
  palette,
  ...props
}: {
  grid: string[]
  palette: Record<string, string>
} & SVGProps<SVGSVGElement>) {
  const height = grid.length
  const width = grid[0].length
  const cells: { x: number; y: number; fill: string }[] = []

  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const key = grid[y][x]
      if (key === ".") continue
      const fill = palette[key]
      if (fill) cells.push({ x, y, fill })
    }
  }

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      shapeRendering="crispEdges"
      data-pixel
      aria-hidden
      {...props}
    >
      {cells.map((cell) => (
        <rect key={`${cell.x}-${cell.y}`} x={cell.x} y={cell.y} width={1} height={1} fill={cell.fill} />
      ))}
    </svg>
  )
}

const HEADS = [
  "............",
  ".XXXX..XXXX.",
  ".XXXX..XXXX.",
  ".X..X..X..X.",
  ".XXXX..XXXX.",
  ".XXXX..XXXX.",
  "............",
  ".XXXXXXXXXX.",
  ".XXXXXXXXXX.",
  ".XXXXXXXXXX.",
  "............",
  "............",
]

const GEAR = [
  "............",
  "...X.XX.X...",
  "...XXXXXX...",
  ".XXXXXXXXXX.",
  "..XXX..XXX..",
  ".XXX....XXX.",
  ".XXX....XXX.",
  "..XXX..XXX..",
  ".XXXXXXXXXX.",
  "...XXXXXX...",
  "...X.XX.X...",
  "............",
]

const HOUSE = [
  "............",
  ".....XX.....",
  "....XXXX....",
  "...XXXXXX...",
  "..XXXXXXXX..",
  ".XXXXXXXXXX.",
  "..XX....XX..",
  "..XX.XX.XX..",
  "..XX.XX.XX..",
  "..XXXXXXXX..",
  "............",
  "............",
]

const mono = { X: "currentColor" }

export function PixelHeads(props: SVGProps<SVGSVGElement>) {
  return <PixelArt grid={HEADS} palette={mono} {...props} />
}

export function PixelHouse(props: SVGProps<SVGSVGElement>) {
  return <PixelArt grid={HOUSE} palette={mono} {...props} />
}

export function PixelGear(props: SVGProps<SVGSVGElement>) {
  return <PixelArt grid={GEAR} palette={mono} {...props} />
}

const GRASS_BLOCK = [
  "HHHHHHHHHHHHHHHH",
  "GgGGGGGGGGgGGggG",
  "GgGGggGGgggGgGGG",
  "GGgGgGggGGGgGGGG",
  "DDGGGdDdDggGdgdD",
  "dDGDdDDDdDdDDdDG",
  "DDdDddDdddDDDdDD",
  "dDdDDdDdDDDdddDD",
  "DDDDDDDDDDDDdDDd",
  "DDDDDDDDDDDDDDDD",
  "DDDddDDDDdDdDdDD",
  "DDDDDDDddDdDDddD",
  "DDdDDDDdDDDDDDDD",
  "DDDdDDDDDDDdDDDD",
  "DDDdDDdDDddDDDDD",
  "SSSSSSSSSSSSSSSS",
]

// The app's brand mark. H and S are the lit top and shadowed base edges,
// which give the tile the same hard depth as the GUI bevels.
export function PixelGrassBlock(props: SVGProps<SVGSVGElement>) {
  return (
    <PixelArt
      grid={GRASS_BLOCK}
      palette={{
        H: "#7cc95a",
        G: "#5fa83d",
        g: "#4e8f32",
        D: "#866043",
        d: "#6f4f37",
        S: "#4f3827",
      }}
      {...props}
    />
  )
}
