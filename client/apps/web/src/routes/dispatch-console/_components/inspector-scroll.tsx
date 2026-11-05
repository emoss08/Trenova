import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import type { ReactNode } from "react";

/** The inspector's scroll area: every state of the pane scrolls inside its fixed height. */
export function InspectorScroll({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <ScrollArea className="min-h-0 flex-1" maskVariant="card" maskHeight={18}>
      <div className={className}>{children}</div>
    </ScrollArea>
  );
}
