import { SettingsPage } from "@/components/settings/settings-page";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { InvoiceAdjustmentControlSkeleton } from "./skeleton";

const InvoiceAdjustmentControlForm = lazy(
  () => import("./_components/invoice-adjustment-control-form"),
);

export function InvoiceAdjustmentControlPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Invoice adjustment controls")}
      description={t("Set the rules for credits, rebills and write-offs on invoices.")}
      skeleton={<InvoiceAdjustmentControlSkeleton />}
    >
      <InvoiceAdjustmentControlForm />
    </SettingsPage>
  );
}
