import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type EmptySheetProps = {
  title: string;
  description: string;
  /** The way forward, usually one button or link. */
  action?: ReactNode;
  /** A faint drawing of what the page will hold: the blank form, the unfilled list. */
  sketch: ReactNode;
  className?: string;
  sketchClassName?: string;
};

/**
 * An empty page drawn as what it is going to be. A page shows a faint sketch
 * of the thing it will hold, fading out where the entries would go, with the
 * words underneath saying what to do next. The sketch is decoration for the
 * eye only and is hidden from assistive technology; the words carry the
 * meaning.
 */
export function EmptySheet({
  title,
  description,
  action,
  sketch,
  className,
  sketchClassName,
}: EmptySheetProps) {
  return (
    <div
      className={cn(
        "cc-fade-in flex w-full flex-col items-center gap-5 px-4 py-8 text-center",
        className,
      )}
    >
      <div
        aria-hidden
        className={cn(
          "w-full max-w-lg [mask-image:linear-gradient(to_bottom,black_35%,transparent)]",
          sketchClassName,
        )}
      >
        {sketch}
      </div>
      <div className="flex max-w-md flex-col items-center gap-1.5">
        <h3 className="text-sm font-medium">{title}</h3>
        <p className="text-muted-foreground text-xs leading-relaxed whitespace-pre-line">
          {description}
        </p>
        {action ? <div className="mt-2">{action}</div> : null}
      </div>
    </div>
  );
}

/** A line of unwritten text. */
export function GhostLine({ className }: { className?: string }) {
  return <span className={cn("bg-muted block h-1.5 rounded-full", className)} />;
}

/** An unticked box. */
export function GhostBox({ className }: { className?: string }) {
  return <span className={cn("border-border/70 block size-3.5 rounded-sm border", className)} />;
}

/** An unfilled track, with an optional faint share drawn in. */
export function GhostBar({ share = 0, className }: { share?: number; className?: string }) {
  return (
    <span className={cn("bg-muted flex h-1 w-24 overflow-hidden rounded-full", className)}>
      {share > 0 ? (
        <span className="bg-brand/30 h-full rounded-full" style={{ width: `${share}%` }} />
      ) : null}
    </span>
  );
}
