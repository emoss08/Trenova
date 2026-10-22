import { usePermission } from "@/hooks/use-permission";
import { lazyComponent } from "@/lib/lazy-component";
import { useAppDialogOpen, useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Sheet, SheetContent } from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Suspense, useEffect } from "react";
import { useLocation } from "react-router";

// The panel, its queries and the role picker are a chunk of their own, so the
// shell's import of this sheet names one file to preload, not the panel's.
const Panel = lazyComponent(
  () => import("./page-access-panel"),
  (module) => module.PageAccessPanel,
);

/** The panel's layout while its code loads: its header, the role picker and the report. */
export function PageAccessPanelSkeleton() {
  const t = useT();

  return (
    <div aria-busy="true" aria-label={t("Loading page access")} className="flex flex-col">
      <div aria-hidden className="border-border flex flex-col gap-1.5 border-b p-4 pr-12">
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-3 w-48" />
      </div>
      <div aria-hidden className="flex flex-col gap-4 p-4">
        <Skeleton className="h-8 w-full rounded-md" />
        <div className="flex gap-2">
          <Skeleton className="h-5 w-28" />
          <Skeleton className="h-5 w-36" />
        </div>
        <Skeleton className="h-24 w-full rounded-md" />
        <Skeleton className="h-40 w-full rounded-md" />
      </div>
    </div>
  );
}

/**
 * The page access panel's sheet, mounted once the panel is asked for. Only
 * an administrator, who may read roles, may view the screen as a role.
 */
export function PageAccessSheet() {
  const { pathname } = useLocation();
  const { allowed: canPreview } = usePermission(Resource.Role, Operation.Read);
  const open = useAppDialogOpen("page-access");
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);

  useEffect(() => {
    void Panel.preload().catch(() => undefined);
  }, []);

  return (
    <Sheet open={open} onOpenChange={(next) => setDialogOpen("page-access", next)}>
      <SheetContent side="right" className="w-full gap-0 p-0 sm:max-w-lg">
        {open ? (
          <Suspense fallback={<PageAccessPanelSkeleton />}>
            <Panel.Component location={pathname} canPreview={canPreview} />
          </Suspense>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
