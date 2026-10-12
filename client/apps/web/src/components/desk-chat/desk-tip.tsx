import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { ReactElement } from "react";

/**
 * A Desk control's tooltip, in place of the browser's own title: the shared
 * tooltip, below the control. The control is the trigger itself, so a menu or
 * popover trigger keeps working underneath it.
 */
export function DeskTip({
  label,
  side = "bottom",
  children,
}: {
  label: string;
  side?: "top" | "bottom" | "left" | "right";
  children: ReactElement;
}) {
  return (
    <Tooltip>
      <TooltipTrigger render={children} />
      <TooltipContent side={side}>{label}</TooltipContent>
    </Tooltip>
  );
}
