import { cn } from "@trenova/shared/lib/utils";
import { GripVerticalIcon } from "@trenova/shared/components/icons";
import { preferenceStorage } from "@trenova/shared/lib/preference-storage";
import { Group, Panel, Separator, useDefaultLayout } from "react-resizable-panels";

type ResizablePanelGroupProps = React.ComponentProps<typeof Group> & {
  /**
   * Remembers the sizes the person dragged the panels to, in this browser, under this id. Every
   * panel in the group then needs a stable `id`, which is the key its size is saved under.
   */
  persistId?: string;
};

function ResizablePanelGroup({ className, persistId, ...props }: ResizablePanelGroupProps) {
  if (persistId) return <PersistedPanelGroup persistId={persistId} className={className} {...props} />;
  return <Group className={cn("h-full w-full", className)} {...props} />;
}

/**
 * A group that starts at the layout last saved under its id and saves each layout a drag or a
 * key leaves it in; a window resize is not saved, so it never overwrites the person's choice.
 */
function PersistedPanelGroup({
  className,
  persistId,
  onLayoutChanged,
  ...props
}: ResizablePanelGroupProps & { persistId: string }) {
  const layout = useDefaultLayout({
    id: `resizable:${persistId}`,
    storage: preferenceStorage,
    onlySaveAfterUserInteractions: true,
  });
  return (
    <Group
      className={cn("h-full w-full", className)}
      defaultLayout={layout.defaultLayout}
      onLayoutChanged={(next, meta) => {
        layout.onLayoutChanged(next, meta);
        onLayoutChanged?.(next, meta);
      }}
      {...props}
    />
  );
}

const ResizablePanel = Panel;

// react-resizable-panels v4 exposes the separator's own orientation via
// aria-orientation (the inverse of the group's): a separator between
// side-by-side panels is "vertical", one between stacked panels is
// "horizontal". All orientation-dependent styling must key off that
// attribute — the v2-era data-panel-group-direction attribute no longer
// exists.
//
// It also reports its state in data-separator ("inactive", "hover", "active"
// while dragged, "focus", "disabled"), with hover read from its own hit area,
// which is wider than the line. Hovered, dragged or focused, a 3px bar in the
// brand colour is drawn over the 1px line (::before), so the layout does not
// move; the hover waits a moment, as VS Code's sash does, so crossing a
// separator on the way somewhere else does not flash it.
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
        "relative flex shrink-0 items-center justify-center bg-border after:absolute ui-focus-ring",
        "before:pointer-events-none before:absolute before:z-10 before:bg-transparent before:transition-colors before:duration-150",
        "data-[separator=hover]:before:bg-brand data-[separator=hover]:before:delay-150 data-[separator=active]:before:bg-brand data-[separator=focus]:before:bg-brand",
        "aria-[orientation=vertical]:w-px aria-[orientation=vertical]:cursor-col-resize aria-[orientation=vertical]:self-stretch aria-[orientation=vertical]:after:inset-y-0 aria-[orientation=vertical]:after:-right-1 aria-[orientation=vertical]:after:-left-1",
        "aria-[orientation=vertical]:before:inset-y-0 aria-[orientation=vertical]:before:left-1/2 aria-[orientation=vertical]:before:w-0.75 aria-[orientation=vertical]:before:-translate-x-1/2",
        "aria-[orientation=horizontal]:h-px aria-[orientation=horizontal]:w-full aria-[orientation=horizontal]:cursor-row-resize aria-[orientation=horizontal]:after:inset-x-0 aria-[orientation=horizontal]:after:-top-1 aria-[orientation=horizontal]:after:-bottom-1 [&[aria-orientation=horizontal]>div]:rotate-90",
        "aria-[orientation=horizontal]:before:inset-x-0 aria-[orientation=horizontal]:before:top-1/2 aria-[orientation=horizontal]:before:h-0.75 aria-[orientation=horizontal]:before:-translate-y-1/2",
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
