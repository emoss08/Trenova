import { InfoCircleIcon } from "@trenova/shared/components/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * A small info icon beside a label that shows one short note on hover or focus. For a note that
 * runs to paragraphs, or one a person keeps open while reading, use InfoPopover.
 *
 * The trigger is a focusable span, not a button: a read-only form wraps its fields in a disabled
 * fieldset, which disables every button inside it, and the note has to stay readable there.
 */
export function InfoTooltip({
  label,
  children,
  className,
}: {
  /** What the note is about; the button's accessible name reads "About {label}". */
  label: string;
  children: ReactNode;
  className?: string;
}) {
  const t = useT();

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            tabIndex={0}
            role="img"
            aria-label={t("About {0}", label)}
            className={cn(
              "inline-flex size-4 shrink-0 cursor-help items-center justify-center rounded-sm ui-focus-ring text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
              className,
            )}
          >
            <InfoCircleIcon className="size-3.5" />
          </span>
        }
      />
      <TooltipContent className="max-w-xs">{children}</TooltipContent>
    </Tooltip>
  );
}
