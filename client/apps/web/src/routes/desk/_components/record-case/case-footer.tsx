import { cn } from "@trenova/shared/lib/utils";
import { Children, type ReactNode } from "react";

/**
 * The foot of a plan: a sentence saying where things stand, and what can be
 * done about it. The actions are one group. They sit beside the sentence when
 * the two fit on a line, and otherwise go under it together, so a row of
 * buttons is never split with some beside the sentence and some below.
 */
export function CaseFooter({
  note,
  lead,
  plain = false,
  children,
}: {
  /** Where things stand. Left out, the actions stand alone. */
  note?: ReactNode;
  /** Something said above the foot's line, at full width: why the last try did not go through. */
  lead?: ReactNode;
  /** The note brings its own type (an outcome box), so the foot's quiet text is not put on it. */
  plain?: boolean;
  /** The actions, first to last. */
  children?: ReactNode;
}) {
  const hasActions = Children.toArray(children).length > 0;
  const hasNote = note !== undefined && note !== null && note !== false && note !== "";
  if (!hasActions && !hasNote && !lead) {
    return null;
  }

  return (
    <div className="border-t border-dsk-b-sub bg-dsk-canvas px-4 py-3">
      {lead}
      {(hasNote || hasActions) && (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2.5">
          {hasNote && (
            <div className={cn("min-w-0 flex-auto", !plain && "text-dsk-sm text-dsk-subtle")}>
              {note}
            </div>
          )}
          {hasActions && (
            <div className="flex max-w-full flex-wrap items-center gap-2">{children}</div>
          )}
        </div>
      )}
    </div>
  );
}
