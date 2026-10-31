import { useSessionContextStore } from "@/stores/session-context-store";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * Takes over from the organization-changed prompt once the person keeps the page open, so
 * they can copy their work while it stays clear that nothing more saves here. It is fixed to
 * the bottom of the window, below dialogs and sheets, so every page shows it whatever its
 * frame: the app shell, the Desk, onboarding.
 */
export function OrganizationChangedBanner() {
  const t = useT();
  const shown = useSessionContextStore((state) => state.organizationChanged && state.keptOpen);

  if (!shown) {
    return null;
  }

  return (
    <Alert
      variant="warning"
      size="sm"
      role="status"
      className="fixed inset-x-0 bottom-0 z-40 rounded-none border-x-0 border-b-0"
    >
      <AlertDescription>
        {t("Your organization changed in another tab, so nothing more can be saved on this page.")}
      </AlertDescription>
      <AlertAction>
        <Button type="button" size="xs" variant="outline" onClick={() => window.location.reload()}>
          {t("Reload this page")}
        </Button>
      </AlertAction>
    </Alert>
  );
}
