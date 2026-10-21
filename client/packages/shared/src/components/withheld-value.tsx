import { Lock01Icon } from "@trenova/shared/components/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { referenceKindAccessLabel } from "@trenova/shared/lib/reference-access";
import { cn } from "@trenova/shared/lib/utils";
import type { ReferenceKind } from "@trenova/shared/types/generated/reference-views";
import type { ReactNode } from "react";

type LockedReferenceProps = {
  /** The kind of record whose read the person lacks. */
  kind: ReferenceKind;
  children: ReactNode;
  className?: string;
  /**
   * `inline` (the default) draws the lock before a name and truncates the name; `row` keeps
   * the caller's own row layout (an avatar, a name and a caption) and ends it with the lock.
   */
  layout?: "inline" | "row";
};

/**
 * Something about a linked record drawn without its kind's read: a lock, the content, and a
 * tooltip saying whose read is missing. See docs/engineering/reference-views.md.
 */
export function LockedReference({
  kind,
  children,
  className,
  layout = "inline",
}: LockedReferenceProps) {
  const t = useT();
  const reason = t("You don't have access to {0}", referenceKindAccessLabel(kind));
  const lock = (
    <Lock01Icon
      aria-hidden
      data-slot="locked-reference-icon"
      className="size-3 shrink-0 text-foreground-muted"
    />
  );

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          layout === "row" ? (
            <div
              data-slot="locked-reference"
              tabIndex={0}
              aria-description={reason}
              className={cn("cursor-default ui-focus-ring", className)}
            />
          ) : (
            <span
              data-slot="locked-reference"
              tabIndex={0}
              aria-description={reason}
              className={cn(
                "inline-flex min-w-0 cursor-default items-center gap-1 rounded-badge ui-focus-ring",
                className,
              )}
            />
          )
        }
      >
        {layout === "row" ? (
          <>
            {children}
            {lock}
          </>
        ) : (
          <>
            {lock}
            <span className="truncate">{children}</span>
          </>
        )}
      </TooltipTrigger>
      <TooltipContent>{reason}</TooltipContent>
    </Tooltip>
  );
}

type WithheldPanelProps = {
  /** What the person cannot see and whose access it needs, as one translated sentence. */
  children: ReactNode;
  className?: string;
};

/**
 * A whole card or section withheld because the person may not read a kind of record it is
 * about: a lock and a sentence naming the access it needs, in place of the content, so it
 * never reads as an empty card.
 */
export function WithheldPanel({ children, className }: WithheldPanelProps) {
  return (
    <div
      data-slot="withheld-panel"
      className={cn(
        "flex flex-1 flex-col items-center justify-center gap-2 px-3 py-6 text-center",
        className,
      )}
    >
      <Lock01Icon aria-hidden className="size-4 shrink-0 text-foreground-muted" />
      <p className="max-w-64 text-xs text-foreground-muted">{children}</p>
    </div>
  );
}

type WithheldValueProps = {
  /** The kind of record the withheld field belongs to. */
  kind: ReferenceKind;
  className?: string;
};

/**
 * A linked record's field the server withheld because the person may not read its kind
 * (`null`, named in the response's `extensions.withheldByAccess`). It says so rather than
 * drawing a dash, a zero or "None", which would read as the record's actual value.
 */
export function WithheldValue({ kind, className }: WithheldValueProps) {
  const t = useT();
  return (
    <LockedReference kind={kind} className={cn("text-xs text-foreground-muted", className)}>
      {t("Hidden")}
    </LockedReference>
  );
}
