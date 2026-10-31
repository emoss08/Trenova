import { useSessionContextStore } from "@/stores/session-context-store";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * Stands in front of a tab whose session another tab moved to a different organization.
 * Escape and a click outside do not close it: the server refuses everything this tab sends.
 * The person reloads, keeps the page open to copy their work (the bar takes over), or
 * switches back in the other tab, which closes it.
 */
export function OrganizationChangedDialog({
  open,
  onReload,
  onKeepOpen,
}: {
  open: boolean;
  onReload: () => void;
  onKeepOpen: () => void;
}) {
  const t = useT();

  return (
    <AlertDialog open={open}>
      <AlertDialogContent data-testid="organization-changed-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>{t("Your organization changed in another tab")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "Nothing more can be saved on this page. Reload it to continue in the organization you switched to, or switch back in the other tab to keep working here.",
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <Button variant="outline" onClick={onKeepOpen}>
            {t("Keep this page open")}
          </Button>
          <AlertDialogAction onClick={onReload}>{t("Reload this page")}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/**
 * Mounted once, at the root. Reloading is the person's own choice, so the browser still asks
 * before it throws away unsaved edits.
 */
export function OrganizationChangedDialogHost() {
  const open = useSessionContextStore((state) => state.organizationChanged && !state.keptOpen);
  const keepPageOpen = useSessionContextStore((state) => state.keepPageOpen);
  return (
    <OrganizationChangedDialog
      open={open}
      onReload={() => window.location.reload()}
      onKeepOpen={keepPageOpen}
    />
  );
}
