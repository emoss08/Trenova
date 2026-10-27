import { cn } from "@trenova/shared/lib/utils";
import { kindLook } from "./lading-file-kinds";

type GlyphKind = Parameters<typeof kindLook>[0];

const SIZE = {
  sm: { tile: "size-4 rounded", icon: "size-2.5" },
  md: { tile: "size-7 rounded-md", icon: "size-3.5" },
  lg: { tile: "size-9 rounded-lg", icon: "size-4.5" },
} as const;

/** A kind's icon on its tinted tile, the one mark the tree, the palette and its form share. */
export function KindGlyph({
  kind,
  size = "sm",
  className,
}: {
  kind: GlyphKind;
  size?: keyof typeof SIZE;
  className?: string;
}) {
  const look = kindLook(kind);
  const Icon = look.icon;
  return (
    <span
      aria-hidden
      className={cn(
        "flex shrink-0 items-center justify-center",
        SIZE[size].tile,
        look.tile,
        className,
      )}
    >
      <Icon className={SIZE[size].icon} />
    </span>
  );
}
