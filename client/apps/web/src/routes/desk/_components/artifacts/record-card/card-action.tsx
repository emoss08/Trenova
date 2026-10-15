import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import type { ComponentProps, ReactNode } from "react";

/**
 * One of a card's actions, taken by the person reading it. When something
 * stands in its way the button stays where it is, disabled, and says why on
 * hover and to a screen reader, rather than vanishing and leaving the person
 * to guess where it went.
 */
export function CardActionButton({
  blocker,
  busy = false,
  className,
  children,
  ...props
}: Omit<ComponentProps<typeof Button>, "disabled"> & {
  /** Why the action cannot be taken now, or null when it can. */
  blocker: string | null;
  /** Something the card started is still running. */
  busy?: boolean;
  children: ReactNode;
}) {
  return (
    <Tooltip disabled={blocker === null}>
      <TooltipTrigger
        render={<span className="inline-flex" tabIndex={blocker === null ? undefined : 0} />}
      >
        <Button
          {...props}
          className={cn("disabled:opacity-45", className)}
          disabled={busy || blocker !== null}
          aria-description={blocker ?? undefined}
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-72">
        {blocker}
      </TooltipContent>
    </Tooltip>
  );
}
