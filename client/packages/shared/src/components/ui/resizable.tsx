import { cn } from "@trenova/shared/lib/utils";
import { GripVerticalIcon } from "lucide-react";
import { Group, Panel, Separator } from "react-resizable-panels";

function ResizablePanelGroup({ className, ...props }: React.ComponentProps<typeof Group>) {
  return <Group className={cn("h-full w-full", className)} {...props} />;
}

const ResizablePanel = Panel;

// react-resizable-panels v4 exposes the separator's own orientation via
// aria-orientation (the inverse of the group's): a separator between
// side-by-side panels is "vertical", one between stacked panels is
// "horizontal". All orientation-dependent styling must key off that
// attribute — the v2-era data-panel-group-direction attribute no longer
// exists.
function ResizableHandle({
  withHandle,
  className,
  ...props
}: React.ComponentProps<typeof Separator> & {
  withHandle?: boolean;
}) {
  return (
    <Separator
      className={cn(
        "relative flex shrink-0 items-center justify-center bg-border after:absolute focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:outline-hidden",
        "aria-[orientation=vertical]:w-px aria-[orientation=vertical]:cursor-col-resize aria-[orientation=vertical]:self-stretch aria-[orientation=vertical]:after:inset-y-0 aria-[orientation=vertical]:after:-right-1 aria-[orientation=vertical]:after:-left-1",
        "aria-[orientation=horizontal]:h-px aria-[orientation=horizontal]:w-full aria-[orientation=horizontal]:cursor-row-resize aria-[orientation=horizontal]:after:inset-x-0 aria-[orientation=horizontal]:after:-top-1 aria-[orientation=horizontal]:after:-bottom-1 [&[aria-orientation=horizontal]>div]:rotate-90",
        className,
      )}
      {...props}
    >
      {withHandle && (
        <div className="z-10 flex h-4 w-3 items-center justify-center rounded-sm border bg-border">
          <GripVerticalIcon className="size-2.5" />
        </div>
      )}
    </Separator>
  );
}

export { ResizablePanelGroup, ResizablePanel, ResizableHandle };
