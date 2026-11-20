import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { LeaveControlSkeleton } from "./skeleton";

const LeaveControlForm = lazy(() => import("./_components/leave-control-form"));

export function LeaveControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Leave settings")}
      description={t(
        "How family and medical leave is measured, and what an employee must do to qualify",
      )}
      skeleton={<LeaveControlSkeleton />}
    >
      <LeaveControlForm />
    </SettingsPage>
  );
}
