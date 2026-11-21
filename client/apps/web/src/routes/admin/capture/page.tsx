import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { CaptureSkeleton } from "./skeleton";

const CaptureConsole = lazy(() => import("./_components/capture-console"));

/**
 * Scanning and printing, for the people who run it: whether it is on and how it files, the
 * scan profiles people pick from, and every computer paired to scan. Each part is shown to
 * whoever may read it.
 */
export function CaptureAdminPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Scanning and printing")}
      description={t(
        "Turn on Trenova Capture, set the scan profiles people pick from, and see every computer paired to scan",
      )}
      skeleton={<CaptureSkeleton />}
    >
      <CaptureConsole />
    </SettingsPage>
  );
}
