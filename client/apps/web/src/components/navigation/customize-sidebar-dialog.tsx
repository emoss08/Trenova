import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  useSidebarCustomizationOptions,
  useSidebarPreferences,
} from "@/hooks/use-sidebar-preferences";
import { SlidersHorizontalIcon } from "lucide-react";
import { cn } from "@trenova/shared/lib/utils";
import { lazy, Suspense, useState, type ReactElement } from "react";

// The editor drags in dnd-kit, the select/switch/checkbox primitives and the
// quick-action icon map. It is reachable from every page through the sidebar,
// so it only gets fetched once someone actually opens it.
const CustomizeSidebarForm = lazy(() => import("./customize-sidebar-form"));

function FormSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      {Array.from({ length: 6 }, (_, index) => (
        <Skeleton key={index} className="h-8 w-full rounded-md" />
      ))}
    </div>
  );
}

/**
 * `ghost` drops the outline so the trigger can sit among the other icon
 * buttons of a rail or footer; `trigger` replaces the icon button entirely
 * for surfaces that want a labelled control.
 */
export function CustomizeSidebarDialog({
  ghost = false,
  trigger,
}: {
  ghost?: boolean;
  trigger?: ReactElement;
}) {
  const [open, setOpen] = useState(false);
  const { data: preferences, isPlaceholderData } = useSidebarPreferences();
  const { data: options } = useSidebarCustomizationOptions(open);
  const isReady = preferences != null && !isPlaceholderData && options != null;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          trigger ?? (
            <button
              type="button"
              aria-label="Customize sidebar"
              title="Customize sidebar"
              className={cn(
                "text-muted-foreground hover:text-foreground flex size-7 shrink-0 items-center justify-center rounded-md transition-colors",
                ghost
                  ? "hover:bg-accent"
                  : "border-border bg-background hover:border-ring/40 border",
              )}
            />
          )
        }
      >
        {trigger ? null : <SlidersHorizontalIcon className="size-3.5" strokeWidth={1.75} />}
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Customize sidebar</DialogTitle>
          <DialogDescription>
            Choose what the sidebar shows: which counts you watch, which shortcuts you keep, and how
            much activity you see. Your choices follow you across devices.
          </DialogDescription>
        </DialogHeader>
        {isReady ? (
          <Suspense fallback={<FormSkeleton />}>
            <CustomizeSidebarForm
              preferences={preferences}
              options={options}
              onSaved={() => setOpen(false)}
            />
          </Suspense>
        ) : (
          <FormSkeleton />
        )}
      </DialogContent>
    </Dialog>
  );
}
