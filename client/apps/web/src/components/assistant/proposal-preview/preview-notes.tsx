import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import type { PreviewWarningTone } from "./preview-warnings";

const DOT: Record<PreviewWarningTone, string> = {
  danger: "bg-danger",
  warning: "bg-warning",
  info: "bg-foreground-subtle",
};

/**
 * One thing a person deciding should know, said in a line: a small dot for
 * how much it matters, the sentence, and at most one way to act on it. It is
 * not a banner. A change used to arrive under a stack of coloured boxes, each
 * shouting about something the reader mostly could not change; a note takes
 * the room of the sentence it is, and only the dot carries colour.
 *
 * `danger` is the one note that stops the change (it is announced as an
 * alert); the others are read as status.
 */
export function PreviewNote({
  tone,
  title,
  action,
  children,
}: {
  tone: PreviewWarningTone;
  /** What the note is about, when the sentence under it needs a heading. */
  title?: ReactNode;
  action?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div
      role={tone === "danger" ? "alert" : "status"}
      data-slot="preview-note"
      data-tone={tone}
      className="flex min-w-0 items-start gap-2.5 text-sm"
    >
      <span aria-hidden className={cn("mt-[7px] size-1.5 shrink-0 rounded-full", DOT[tone])} />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        {title && <span className="text-foreground font-medium">{title}</span>}
        {children && <div className="text-foreground-muted min-w-0 break-words">{children}</div>}
      </div>
      {action && <div className="flex shrink-0 items-center gap-1.5">{action}</div>}
    </div>
  );
}

/**
 * The notes of one change, kept together under one quiet heading so they read
 * as a short list and not as a column of alarms.
 */
export function PreviewNotes({ label, children }: { label: string; children: ReactNode }) {
  return (
    <section
      data-slot="preview-notes"
      aria-label={label}
      className="border-border flex min-w-0 flex-col gap-2 rounded-lg border px-3 py-2.5"
    >
      <span className="text-foreground-subtle text-xs">{label}</span>
      {children}
    </section>
  );
}
