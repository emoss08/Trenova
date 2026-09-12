import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const MatchingWorkspace = lazy(() => import("./_components/matching-workspace"));

export function CarrierInvoiceMatchingPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier Invoice Matching"),
        description: t(
          "Reconcile inbound carrier freight invoices against negotiated buy rates — link carriers, create matches, and resolve variances.",
        ),
      }}
    >
      <LazyComponent>
        <MatchingWorkspace />
      </LazyComponent>
    </PageLayout>
  );
}
