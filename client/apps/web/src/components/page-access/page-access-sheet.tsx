import { usePermission } from "@/hooks/use-permission";
import { useAppDialogOpen, useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Sheet, SheetContent } from "@trenova/shared/components/ui/sheet";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useLocation } from "react-router";
import { PageAccessPanel } from "./page-access-panel";

/**
 * The page access panel's sheet, read on its first opening. Only an
 * administrator, who may read roles, may view the screen as a role.
 */
export function PageAccessSheet() {
  const { pathname } = useLocation();
  const { allowed: canPreview } = usePermission(Resource.Role, Operation.Read);
  const open = useAppDialogOpen("page-access");
  const setDialogOpen = useAppDialogsStore((state) => state.setDialogOpen);

  return (
    <Sheet open={open} onOpenChange={(next) => setDialogOpen("page-access", next)}>
      <SheetContent side="right" className="w-full gap-0 p-0 sm:max-w-lg">
        {open ? <PageAccessPanel location={pathname} canPreview={canPreview} /> : null}
      </SheetContent>
    </Sheet>
  );
}
