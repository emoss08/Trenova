import { cn } from "@trenova/shared/lib/utils";
import type { PaletteIcon } from "./palette-model";

const TILE_SIZE = {
  sm: "size-7 [&_svg]:size-3.5",
  lg: "size-10 [&_svg]:size-5",
} as const;

/**
 * The square a row leads with. Its colour names the kind of thing the row
 * is, never how urgent it is; urgency is the badge's job.
 */
export function PaletteTile({
  icon: Icon,
  initials,
  className,
  size = "sm",
}: {
  icon?: PaletteIcon;
  initials?: string;
  className: string;
  size?: keyof typeof TILE_SIZE;
}) {
  return (
    <span
      aria-hidden
      className={cn(
        "rounded-control flex shrink-0 items-center justify-center ring-1 ring-inset",
        TILE_SIZE[size],
        className,
      )}
    >
      {initials ? (
        <span className={cn("font-medium", size === "lg" ? "text-sm" : "text-2xs")}>
          {initials}
        </span>
      ) : Icon ? (
        <Icon strokeWidth={1.75} />
      ) : null}
    </span>
  );
}
