import { Kbd, KbdGroup, kbdOnSolidClassName } from "@trenova/shared/components/ui/kbd";
import type { MatchRange } from "@trenova/shared/lib/fuzzy-score";
import { cn } from "@trenova/shared/lib/utils";
import { memo } from "react";
import { NEUTRAL_TILE_CLASS } from "./palette-entities";
import type { PaletteIcon } from "./palette-model";

/**
 * The text with the characters the matcher used marked, and nothing else. A
 * field the query did not win is drawn plain, so "ai" never lights up
 * "Trailers" because it happens to contain the letters.
 */
export const MatchText = memo(function MatchText({
  text,
  ranges,
  className,
}: {
  text: string;
  ranges?: readonly MatchRange[];
  className?: string;
}) {
  if (!ranges || ranges.length === 0 || text === "") {
    return <span className={className}>{text}</span>;
  }

  const parts: React.ReactNode[] = [];
  let at = 0;
  for (const [start, end] of ranges) {
    if (start >= text.length) {
      break;
    }
    if (start > at) {
      parts.push(text.slice(at, start));
    }
    parts.push(
      <mark key={start} className="bg-highlight-fill text-foreground -mx-px rounded-[3px] px-px">
        {text.slice(start, Math.min(end, text.length))}
      </mark>,
    );
    at = Math.min(end, text.length);
  }
  if (at < text.length) {
    parts.push(text.slice(at));
  }

  return <span className={className}>{parts}</span>;
});

export type StatusState = "neutral" | "warn" | "bad" | "ok";

const STATUS_DOT: Record<StatusState, string> = {
  neutral: "bg-foreground-faint",
  warn: "bg-warning",
  bad: "bg-danger",
  ok: "bg-success",
};

export function StatusDot({ state, className }: { state: StatusState; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn("size-1.25 shrink-0 rounded-full", STATUS_DOT[state], className)}
    />
  );
}

/**
 * A state as an outlined label with a dot. The dot carries the colour, and
 * only when the state needs someone; the label itself stays neutral, so a
 * list of statuses does not become a list of colours.
 */
export function StatusLabel({
  label,
  state = "neutral",
  className,
}: {
  label: string;
  state?: StatusState;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "border-border text-foreground-muted inline-flex h-4.5 shrink-0 items-center gap-1.5 rounded-full border px-1.5 text-xs whitespace-nowrap",
        className,
      )}
    >
      <StatusDot state={state} />
      {label}
    </span>
  );
}

export function CountChip({ value, className }: { value: number; className?: string }) {
  return (
    <span
      className={cn(
        "bg-sunken text-foreground-muted inline-flex h-4.5 min-w-4.5 shrink-0 items-center justify-center rounded-full px-1.5 font-mono text-xs tabular-nums",
        className,
      )}
    >
      {value.toLocaleString()}
    </span>
  );
}

export function ShortcutKeys({
  keys,
  className,
  onSolid = false,
}: {
  keys: readonly string[];
  className?: string;
  onSolid?: boolean;
}) {
  return (
    <KbdGroup className={className}>
      {keys.map((key) => (
        <Kbd key={key} className={onSolid ? kbdOnSolidClassName : undefined}>
          {key}
        </Kbd>
      ))}
    </KbdGroup>
  );
}

const TILE_SIZE = {
  row: "size-7 [&_svg]:size-3.5",
  related: "size-6.5 [&_svg]:size-3.25",
  chip: "size-5 [&_svg]:size-3",
} as const;

/** The square a row leads with. It says what the row is, never how urgent it is. */
export function PaletteTile({
  icon: Icon,
  mark,
  initials,
  className,
  size = "row",
}: {
  icon?: PaletteIcon;
  /** A mark drawn as it is, sized to the tile, for one that is not a plain icon. */
  mark?: React.ReactNode;
  initials?: string;
  className?: string;
  size?: keyof typeof TILE_SIZE;
}) {
  return (
    <span
      aria-hidden
      className={cn(
        "rounded-control flex shrink-0 items-center justify-center ring-1 ring-inset",
        TILE_SIZE[size],
        NEUTRAL_TILE_CLASS,
        className,
      )}
    >
      {initials ? (
        <span className="text-2xs font-medium tracking-tight">{initials}</span>
      ) : mark ? (
        mark
      ) : Icon ? (
        <Icon strokeWidth={1.75} />
      ) : null}
    </span>
  );
}

/** Parts of a context line, joined by a quiet middle dot. */
export function ContextLine({
  parts,
  className,
}: {
  parts: (React.ReactNode | null | undefined | false)[];
  className?: string;
}) {
  const present = parts.filter((part): part is React.ReactNode => Boolean(part));
  if (present.length === 0) {
    return null;
  }
  return (
    <span
      className={cn(
        "text-foreground-subtle flex min-w-0 items-center gap-1.5 truncate text-sm",
        className,
      )}
    >
      {present.map((part, index) => (
        <span key={index} className="flex min-w-0 items-center gap-1.5">
          {index > 0 && (
            <span aria-hidden className="text-foreground-faint">
              ·
            </span>
          )}
          <span className="truncate">{part}</span>
        </span>
      ))}
    </span>
  );
}
